---
icon: lucide/waypoints
---

# Stdio developer loop

This is the workflow mcpdev exists for: rebuild a local MCP server, inspect what
it advertises, call a safe tool, snapshot the interface, and watch the next edit
without restarting your AI harness.

The example server is the in-repo fixture at `testdata/servers/stdio-lab`, so the
whole loop runs from this checkout. No sibling repository and no personal
absolute path is involved.

Prerequisites: Go 1.27+ and this checkout. The HTTP half needs a Streamable HTTP
fixture from [Test fixtures](../contributing/testing.md).

## Build the fixture

```bash
mkdir -p ./bin
go build -o ./bin/stdio-lab ./testdata/servers/stdio-lab
```

The binary serves the `legacy-stdio` catalog by default. Set `MCPDEV_FIXTURE` to
another catalog name to change the interface it advertises; this page uses
`legacy-paginate` to force a diff.

## Point mcpdev at it with a profile

`testdata/integration/mcpdev.toml` ships an opt-in `lab` profile whose argv is
`./bin/stdio-lab`. Command and `cwd` values resolve from the process working
directory, so run mcpdev from the repository root.

```bash
export MCPDEV_CONFIG=$PWD/testdata/integration/mcpdev.toml
mcpdev --profile lab inspect
```

`xdrun ci` does not need that variable. `go test ./internal/integration` and
`xdrun integration` read the same file for the laboratory loop tests.

## Inspect after a source edit

Each mcpdev command starts a fresh child process:

```bash
mcpdev --profile lab inspect
mcpdev --format json --profile lab inspect tools
mcpdev --profile lab inspect prompts
mcpdev --profile lab inspect resources
mcpdev --profile lab inspect templates
```

You should see tools `echo` and `fail`, prompt `greet`, resources
`fixture://stable/note` and `fixture://stable/blob`, and template
`fixture://items/{id}`. Incomplete collection is labeled (exit class `60`); it is
never a silent empty catalog. Details are in [Inspect a server](../guides/inspect.md)
and [Inventory](../reference/inventory.md).

## Call, prompt, and read

Safe operations only. Do not pass server-returned strings to a shell.

```bash
mcpdev --format json --profile lab call echo --args '{"text":"loop"}'
mcpdev --profile lab prompt greet --arg name=mcpdev
mcpdev --profile lab read fixture://stable/note
mcpdev --profile lab read --template 'fixture://items/{id}' --arg id=n1
```

`--arg` values for prompts and URI templates are strings. `call` `--args` is a
JSON object. See [Call, prompt, and read](../guides/operations.md).

The same sequence as a declarative suite is
`testdata/scenarios/valid/stdio.json`. `testdata/scenarios/valid/stdio-profile.yaml`
is the same file format with a `profile:` target instead of an argv target:

```bash
mcpdev --command ./bin/stdio-lab test testdata/scenarios/valid/stdio.json
```

## Snapshot, change the interface, diff

Capture a baseline from the default catalog, then compare it to the paginated
one. Exit class `40` means the public interface moved at or above `--fail-on`
(default `additive`).

```bash
mcpdev --profile lab snapshot --out /tmp/stdio-lab-baseline.json
mcpdev --format json --command ./bin/stdio-lab \
  --env MCPDEV_FIXTURE=legacy-paginate diff /tmp/stdio-lab-baseline.json
```

The second command exits `40` because the paginated catalog advertises a
different tool set. [Snapshots and diffs](../guides/snapshots-and-diffs.md) covers
`--ignore`, `--rename`, and the classification rules.

## Watch rebuild, restart, reconnect

Watch is a foreground command. `--rebuild` is argv. Staging is renamed onto
`--artifact` only after a successful build; a compiler error keeps the last good
binary.

```bash
mcpdev --format json --command ./bin/stdio-lab watch \
  --rebuild go --rebuild build --rebuild -o --rebuild ./bin/stdio-lab.partial \
  --rebuild ./testdata/servers/stdio-lab \
  --staging ./bin/stdio-lab.partial --artifact ./bin/stdio-lab \
  --root ./testdata/servers --include '*.go' --debounce 200ms
```

Edit `testdata/servers/stdio-lab/main.go`. JSON Lines events on stdout include
`fs`, `build`, `restart`, `negotiate`, `probe`, and `diff`. Ctrl-C cancels
in-flight work and shuts the child down. You do not restart Cursor, Claude, or
another harness to see the new catalog. See [Watch and rebuild](../guides/watch.md).

## HTTP: stateless modern era and redacted headers

Streamable HTTP for `2026-07-28` is POST-only. Put the bearer token in the
environment, not in the TOML file.

```bash
export MCPDEV_LAB_URL=http://127.0.0.1:8080/mcp
export MCPDEV_LAB_TOKEN="Bearer fixture-token"
mcpdev --format json --trace --trace-format json --profile http inspect tools
```

Inspect JSON and traces must show `[REDACTED]` in place of the token. Loopback
`http` is allowed; other insecure combinations stay opt-in. See
[Streamable HTTP](../guides/http.md) and [Configuration](../reference/configuration.md).

## Failure modes

| Symptom | What to check |
| --- | --- |
| `unknown profile lab` | `MCPDEV_CONFIG` does not point at `testdata/integration/mcpdev.toml`, and the current directory has no `mcpdev.toml` with that profile. |
| Connect exit `10` | `./bin/stdio-lab` is missing or failed to start. Rebuild it from the repository root. |
| Diff exit `40` | The public interface changed. That is the CI signal, not a crashed child. |
| Diff exit `60` | A catalog did not finish. Raise `--timeout` or reconnect before treating the run as clean. |
| Token visible in JSON | Stop. Headers must come from `header_env` or `${NAME:?}`, never a literal in the file. |

Do not execute server-provided strings as shell commands. Do not commit secrets,
machine home directories, or login names in integration assets.
