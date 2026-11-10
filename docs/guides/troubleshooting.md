---
icon: lucide/life-buoy
---

# Troubleshooting

This page is the operator map for failures that look like "the server is broken" but are usually framing, negotiation, rebuild, auth, timeouts, or a leftover child. Exit classes are in [CLI](../reference/cli.md). Redacted records are in [Diagnostics](../reference/diagnostics.md).

Match the **exit number** in scripts. Do not scrape stderr. Incomplete work is labeled incomplete; it is never an empty successful catalog.

## Framing errors (stdio stdout is not JSON-RPC)

The child's **stdout** is the MCP stream. The first payload byte must be `{` or `[`. Anything else is protocol class `20`.

| Symptom | Typical exit | What to do |
| --- | --- | --- |
| Banner, log line, or `go run` compile text on stdout | `20` | Move logs to stderr. Rebuild, then `inspect` again. |
| Child prints JSON-RPC on stderr | `0` unless another failure | stderr is never parsed as protocol. `child_stderr` on the diagnostic is informational. |
| Wrapper script echoes before exec | `20` | Exec the server with no extra stdout. argv is never `sh -c`. |
| Windows path missing `.exe` | `10` | Pass the same path `go build -o` wrote. |

Details: [Stdio servers](stdio.md).

## Protocol negotiation

`--protocol auto` (default) probes `server/discover` with `2026-07-28` `_meta`, then may fall back to `initialize`. Pin `--protocol 2025-11-25` or `--protocol 2026-07-28` when you already know the era.

| Symptom | Typical exit | What to do |
| --- | --- | --- |
| HTTP `400` with `UnsupportedProtocolVersionError` (`-32022`), `HeaderMismatch` (`-32020`), or `MissingRequiredClientCapabilityError` (`-32021`) | `20` | This is a **modern** server. Retry with a version from `supported`. Do **not** treat it as legacy HTTP+SSE. |
| HTTP `400` with an empty or non-JSON body | `10` / `20` | Then (and only then) auto may send `initialize`. Enable `--allow-http-sse` only for the deprecated two-endpoint transport. |
| stdio: unrecognized error or probe timeout | `10` / `20` | Auto may send `initialize`. `--protocol 2026-07-28` will not. |
| `--no-discover` | recorded skip, then maybe `60` | Discover is `skipped`, not success. |
| `--command` and `--url` together | `2` | Pick one target. |

Session diagnostics should show `requested`, `discovered`, `negotiated`, and `fallback`. Full rules: [Compatibility](../reference/compatibility.md).

## Rebuild and watch failures

Watch keeps the **last good** artifact unless you pass `--no-keep-last-good`.

| Symptom | What stays true |
| --- | --- |
| Compiler error, missing `go`, rebuild timeout | Class on the build event (`nonzero`, `command_not_found`, `timeout`). Staging is deleted. Last good binary and process stay. |
| Probe / handshake fail after a good build | New generation is killed. Previous ready generation stays current. |
| Incomplete inventory or failed `--smoke` | Not promoted. Not a new snapshot baseline. |
| Events under `.git`, `bin/`, `dist/` | Ignored. |

`--rebuild` and `--command` are argv arrays. Do not point `--command` at the staging file while it is being written. Guide: [Watch and rebuild](watch.md).

## Auth, TLS, and Origin

| Symptom | Typical exit | What to do |
| --- | --- | --- |
| Non-loopback `http` or skipped TLS verify | `2` / `10` | Pass `--insecure` explicitly. It is never implied. |
| `401` / `403` | `10` | Repeatable `--header 'Authorization: Bearer …'` or profile `header_env`. Values are redacted in logs. |
| Origin mismatch | `10` / `20` | Origin is derived from the URL unless you set it in config. |
| Putting a token in a committed `mcpdev.toml` | n/a | Use `${TOKEN:?}` or `header_env`. |

Guide: [Streamable HTTP](http.md). Configuration: [Configuration](../reference/configuration.md).

## Timeouts and cancel

| Flag | Bounds |
| --- | --- |
| `--connect-timeout` | Handshake, including `server/discover` |
| `--timeout` | Individual catalog or invoke RPCs |
| Watch rebuild timeout | Staging discarded; last good kept |

Interrupt (Ctrl-C) cancels the context. Stdio shutdown is stdin close, grace, interrupt the process group, then kill. Exit `10` on connect timeout. Exit `60` if a list walk is canceled mid-page. Tests must not use wall-clock `Sleep` to hide these races.

## Stale processes

Every `inspect` / `call` / `test` case / watch generation starts a **new** child. mcpdev is not a daemon. If a previous run was killed with `kill -9` on mcpdev itself, the child process group may survive.

| Symptom | What to do |
| --- | --- |
| Port still bound after a canceled HTTP lab | Stop the lab process; mcpdev does not own unrelated servers. |
| Fixture binary still running after Ctrl-C | Unusual if mcpdev received SIGINT; check wrappers that ignore signals. |
| Watch "ready" but you edited source | Wait for debounce + rebuild + probe. Failed rebuild keeps the old process on purpose. |

`ps` for the artifact path, then stop that pid only if mcpdev is already gone. Do not `kill -9` as the first step; stdin close is the graceful MCP shutdown.

## Incomplete catalogs and diffs

| `incomplete_reasons` / outcome | Exit (default) |
| --- | --- |
| `pagination_stopped`, `cursor_loop`, `item_limit`, failed list | `60` |
| Diff with an incomplete side (`indeterminate`) | `60` |
| Diff at or above `--fail-on` | `40` |
| Empty list, `status=complete` | `0` |

`--allow-incomplete` keeps `incomplete: true` in JSON and exits `0`. It does not rewrite the result as success-with-empty-data.

## Safe reports

Include `code`, `operation`, `transport`, `era`, `elapsed_ms`, redacted `message` / `cause`, and `incomplete_reasons`. Omit raw `Authorization` values, URL userinfo, and unbounded child stderr. See [Diagnostics](../reference/diagnostics.md).
