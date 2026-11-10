---
icon: lucide/terminal-square
---

# Stdio servers

mcpdev launches a **new** MCP child for every invocation (and every watch restart). It does not keep a hidden daemon around, and it never feeds the argv through a shell. Protocol bytes stay on the child's stdin/stdout. Logs stay on stderr.

The operator flags, stream rules, and exit classes are in [CLI](../reference/cli.md). Failure records and redaction are in [Diagnostics](../reference/diagnostics.md).

## Argv, not a shell string

`--command` / `-c` is repeatable. Each value is one argv slot. Paths with spaces are one slot; they are not split. On Windows the child path should include `.exe` when you pass a built file (the same suffix `go build -o` writes).

A local binary:

```bash
mcpdev inspect --command ./bin/weather
```

`go run` without wrapping it in `sh -c`:

```bash
mcpdev inspect --command go --command run --command ./cmd/weather
```

A wrapper script (still exec of that path, not `bash -c "..."`):

```bash
mcpdev inspect --command ./scripts/run-weather.sh
```

You can also put the argv after `--`:

```bash
mcpdev inspect -- ./bin/weather --flag
```

Profiles use the same argv shape in `mcpdev.toml` (see [Configuration](../reference/configuration.md)):

```toml
[profiles.local]
command = ["go", "run", "."]
```

`command = "go run ."` is invalid. mcpdev will not parse a shell string from config, from the environment, or from server output.

## Environment and working directory

`--env KEY=VALUE` is repeatable and overlays the parent environment. `--cwd` sets the child's working directory (absolute or relative to mcpdev's cwd).

```bash
mcpdev inspect \
  --command ./bin/weather \
  --cwd ./services/weather \
  --env API_URL=http://127.0.0.1:8080
```

Overrides win over inherited keys. The merged environment is applied as a sorted `KEY=VALUE` list so two launches with the same spec are deterministic. Child env values that look like credentials are redacted in diagnostics.

## Fresh binary every time

Each `inspect` / `call` / `watch` reconnect starts `exec` again with the argv you passed. Rebuild the server, then run mcpdev again (or let `watch --rebuild` do it). You do not restart mcpdev to pick up `./bin/weather`.

Watch still tears the previous child down (stdin close, then interrupt, then kill) before starting the next argv.

## Streams

| Child stream | Role |
| --- | --- |
| stdin | JSON-RPC from mcpdev |
| stdout | JSON-RPC to mcpdev |
| stderr | Untrusted diagnostics, copied with a byte bound |

Do not log on stdout. A first stdout byte that is not JSON (`{` or `[`) is a **protocol** error (exit class `20`). An early exit with no JSON-RPC is a **connection** error (exit class `10`). Bytes on stderr never become `protocol_error` by themselves, even if they look like JSON-RPC.

## Shutdown

On cancel, timeout, or the end of a command, mcpdev:

1. Closes the child's stdin (graceful MCP servers exit).
2. Waits a short grace period.
3. Interrupts the process group.
4. Waits again, then kills the group if it is still running.

Cancellation and timeout tests must leave no child process. If shutdown blocks, mcpdev still kills; the error is connection-class, not success.

## Failure modes

| Symptom | Typical class | What to check |
| --- | --- | --- |
| Binary missing or not executable | `10` connection | argv[0], `--cwd`, file mode |
| Child exits before handshake | `10` connection | bounded `child_stderr` |
| Garbage on stdout | `20` protocol | server logs should move to stderr |
| Child ignores interrupt | connection after kill | wrapper that reaps children; mcpdev kills the process group |

Report failures with the redacted diagnostic, not a raw env dump. See [Diagnostics](../reference/diagnostics.md).
