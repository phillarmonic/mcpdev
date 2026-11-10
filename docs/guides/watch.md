---
icon: lucide/eye
---

# Watch and rebuild

Watch mode turns source edits into **controlled rebuilds**, then into a **generation-based process swap**. It does not start a daemon, a browser, or a tunnel. `mcpdev watch` is a foreground command: JSON Lines events on stdout in `--format json`, concise status on stderr otherwise.

Child argv rules stay those in [Stdio servers](stdio.md): never a shell string, never a string taken from server output.

## What is watched

The watcher walks configured roots recursively and normalizes platform events into create, write, remove, rename, and chmod. New directories are added to the watch set. macOS often emits chmod alongside writes; those events share the debounce window. Tests inject an event source and a clock; they do not call `time.Sleep`.

Include globs are optional. When they are empty, every path that is not excluded is eligible. Exclude globs are extra generated or vendor trees you pass in. These directory names (and a few file globs) are ignored by default:

- VCS: `.git`, `.hg`, `.svn`
- Binaries and docs build: `bin`, `dist`, `site`
- Graphify output: `graphify-out`
- Scratch trees: `tmp`, `temp`, `.tmp`, `.zensical`
- Common generated/vendor: `vendor`, `node_modules`, `.idea`, `.vscode`

Editor save noise (`*~`, `*.swp`, `#*#`, numeric Vim `4913` files, `.DS_Store`) is dropped even when an include glob would have matched.

## Coalescing

A burst of events on the same debounce window (default 200ms, injectable clock) becomes **one** rebuild. Events that arrive while a rebuild is running are collapsed into **one follow-up** that starts when the in-flight rebuild finishes. A failed rebuild does not leave the loop; the last successful artifact stays in place.

```text
save, save, chmod  -->  wait debounce  -->  build once
build running + more saves  -->  one follow-up build
build fails  -->  keep watching, keep last good binary
```

## Rebuild argv

The rebuild command is an argv array. mcpdev looks up `argv[0]` and `exec`s it. There is no `sh -c`.

Write the new binary to a **staging** path. On success, mcpdev renames staging onto the last-good path. On command-not-found, nonzero exit, timeout, or cancel, staging is deleted and the last-good file is left untouched (including a half-written staging file).

```bash
go build -o ./bin/weather.partial ./cmd/weather
# success: weather.partial is renamed to ./bin/weather
# failure: ./bin/weather from the previous success remains
```

Stdout and stderr from the rebuild are copied with a byte bound. Classification is explicit: `ok`, `command_not_found`, `nonzero`, `timeout`, `canceled`. Incomplete or failed work is never reported as a clean success.

## Failure modes

| Symptom | What stays true |
| --- | --- |
| Compiler error | Watch continues. Last good artifact is unchanged. |
| Missing build tool | Class `command_not_found`. Watch continues. |
| Rebuild exceeds timeout | Class `timeout`. Staging is discarded. |
| Operator cancel | In-flight rebuild is cancelled. |
| Event in `.git` or `bin` | No rebuild. |

Do not point `--command` at the staging file while a rebuild is writing it. After a successful promote, stdio reconnect execs the last-good path through the generation supervisor.

## Generations and restart

`internal/supervisor` starts the last-good argv as a stdio child, probes it into a protocol session, and promotes at most one generation as current. Generations are monotonic. A newer rebuild cancels an unpromoted start (in-flight probe included). A callback for an older generation is rejected as incomplete work, not as a successful no-op.

States are explicit: `idle`, `building`, `starting`, `ready`, `stopping`, `failed`, `closed`.

By default the last good process stays current until the replacement handshake succeeds. Session close then process close run with a bounded deadline before the new generation is promoted. `ReplaceInPlace` stops the current child first (a gap with no process).

Each generation keeps the rebuild log, bounded child stderr, negotiated protocol metadata (`SessionInfo`), and a termination reason. Startup hang, crash, cancel, and close during transition leave no live child: the failed generation is recorded, and watch can keep serving the previous ready process.

```text
build ok  -->  start gen N+1  -->  probe  -->  stop gen N  -->  ready N+1
probe fail  -->  kill N+1  -->  keep ready N
newer build during probe  -->  cancel N+1  -->  start N+2
```

## Probe, snapshot, and diff

Readiness is handshake plus a **configurable probe**. The default probe is a complete inventory capture (the same public-interface document as `mcpdev snapshot`). You can require named smoke scenarios instead (still on the generation already under probe, not a second child). Extra named scenarios may run after that probe and before promotion.

Each ready generation is snapshotted in memory and compared with the **previous ready** generation. The operator-facing classes are `unchanged`, `additive`, `metadata`, `breaking`, and `indeterminate`. Identical consecutive class+path fingerprints are not printed again.

A failed handshake, incomplete inventory, or failed scenario does **not** become ready and does **not** replace the comparison baseline. The last good process stays current. Watch keeps a bounded in-memory history of timings, outcomes, and failure reasons (default 16 rows).

```text
handshake  -->  inventory or named readiness scenarios  -->  optional smoke  -->  snapshot  -->  diff vs last ready  -->  promote
incomplete list / smoke fail / initialize fail  -->  keep last ready snapshot and process
```

## Command

`--command` is the stdio argv that runs after a successful rebuild. `--rebuild` is a separate argv that compiles into `--staging` and, on success, is renamed onto `--artifact` (the last-good path). `--on-success reconnect` (default) then starts that last-good argv; `--on-success none` only rebuilds.

```bash
mcpdev --command ./bin/weather watch \
  --rebuild go --rebuild build --rebuild -o --rebuild ./bin/weather.partial --rebuild ./cmd/weather \
  --staging ./bin/weather.partial --artifact ./bin/weather --debounce 200ms

mcpdev --format json --command ./bin/weather watch --rebuild true --fail-fast
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--debounce` | `200ms` | Coalesce window. |
| `--fail-fast` | off | Exit after the first failed rebuild or probe. |
| `--keep-last-good` | on | Keep serving the current child until the replacement probe succeeds. `--no-keep-last-good` stops it first. |
| `--smoke NAME` | none | After inventory, `tools/call NAME` with `{}` on the generation already under probe. |
| `--baseline PATH` | none | Comparison snapshot before the first ready generation. |
| `--readiness` | `inventory` | `inventory` or `scenario` (uses `--smoke` names as readiness). |
| `--exit-on-change` | off for watch | Exit class `40` when a ready generation's interface is `metadata`, `additive`, or `breaking`. |
| `--root` | cwd | Watch root (repeatable). |

TTY and JSON Lines emit the **same** kinds: `fs`, `build`, `restart`, `negotiate`, `probe`, `diff`, `scenario`, `recovery`, `shutdown`. Each record has a timestamp and a stable `id` (`e1`, `e2`, …). Human status is append-only on a pipe. Color and in-place `building` lines apply only when stderr is a TTY and `NO_COLOR` is unset.

Ctrl-C and SIGTERM cancel in-flight rebuilds, close the protocol session, and wait a bounded time for the child. Watch does not prompt.
