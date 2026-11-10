---
icon: lucide/terminal
---

# CLI

mcpdev is a command-line client for MCP servers. It connects over stdio or Streamable HTTP, inspects tools/prompts/resources, invokes them, snapshots and diffs public interfaces, runs declarative tests, and watches a changing binary. The full contract (flags, precedence, JSON rules, exit classes, MVP non-goals) is below. Protocol eras are in [Compatibility](compatibility.md). Failure records and redaction are in [Diagnostics](diagnostics.md). Launching a local binary, `go run`, or wrapper is in [Stdio servers](../guides/stdio.md).
Streamable HTTP (`2025-11-25` sessions and `2026-07-28` POST-only) is in [Streamable HTTP](../guides/http.md). Framing, negotiation, rebuild, auth, timeout, and leftover-process failures are in [Troubleshooting](../guides/troubleshooting.md).

This page is the operator summary: commands, configuration, output, and how to script it.

## Install and run

Install paths, archive names, and `version` metadata are in [Installation](installation.md).

```bash
go install github.com/phillarmonic/mcpdev/cmd/mcpdev@latest
mcpdev --format json version
mcpdev inspect --command ./bin/server
```

`inspect`, `call`, `prompt`, `read`, `snapshot`, `diff`, `test`, `watch`, and `version` are implemented.

## Commands

| Command | What it does | Success stdout | Notable exits |
| --- | --- | --- | --- |
| `inspect [tools\|prompts\|resources\|templates]` | List server primitives (omit kind for all). | Catalog, or JSON envelope. | `60` if pagination/discover is incomplete. |
| `call NAME` | `tools/call`. `--args` JSON (`{}`, `@file`, or `-`), `--args-file`, `--arg k=json`. | Tool result. | `0` even when the tool sets `isError`, unless `--fail-on-tool-error` (`30`). |
| `prompt NAME` | `prompts/get`. `--arg k=string` (no JSON coercion). | Prompt messages. | Protocol/connection classes on RPC failure. |
| `read URI` | `resources/read`. `--template` plus `--arg` for RFC 6570 expansion. `--save` for binary. | Text body, or JSON with base64 for binary. | Protocol/connection classes on RPC failure. |
| `snapshot` | Canonical public-interface document. `--out PATH` is an atomic replace. | Snapshot JSON (also when writing a file). | `60` if a catalog did not finish. Existing `--out` files are not replaced on incomplete collection. |
| `diff A B` | Semantic public-interface diff of two files, file-vs-live, or two live targets. | Human summary or structured JSON (`outcome`, `verdict`, `result`). | `40` at or above `--fail-on`; `0` for no-change or below-threshold; `60` if a side is incomplete. |
| `test PATH` | Declarative scenarios ([format](scenario-format.md)). | Pass/fail records. | `30` assertion failure; `60` if an assertion could not be evaluated. Invalid files fail at load (class `2`) before connect. |
| `watch` | Rebuild/restart/reconnect in the foreground. | JSON Lines events in `--format json`; status on stderr in human mode. | Stays running unless `--fail-fast` or `--exit-on-change`. Fatal errors use `10` / `20` / `70`. |
| `version` | Build identity. | Human fields, or JSON `result` with `version`, `commit`, `build_time`, `build_time_policy`, `snapshot_schema`, `goos`, `goarch`. | `0`. |

No command: help on stderr, exit `2`. `--help`: help on stdout, exit `0`.

MVP **does not** include a browser UI, LLM eval command, hosted tunnel, background daemon, or `--transport grpc`. gRPC is not a native MCP transport; no default CLI transport is named `grpc`.

## Connection flags

Pick **one** target:

```bash
mcpdev --command go --command run --command ./cmd/server inspect
mcpdev --url http://127.0.0.1:8080/mcp inspect
mcpdev --profile local inspect
```

| Flag | Environment | Notes |
| --- | --- | --- |
| `--command`, `-c` | `MCPDEV_COMMAND` | stdio argv. Repeat `-c` or use `--` plus args. Never a shell string from the server. |
| `--url` | `MCPDEV_URL` | Streamable HTTP endpoint. |
| `--profile`, `-p` | `MCPDEV_PROFILE` | Name from `mcpdev.toml` or user config. |
| `--header` | | Repeatable `Name: value` for HTTP. Redacted in logs/snapshots. |
| `--env` | | Repeatable `KEY=VALUE` for the stdio child. |
| `--cwd` | `MCPDEV_CWD` | Child working directory. |
| `--timeout` | `MCPDEV_TIMEOUT` | Per-RPC deadline (for example `10s`). |
| `--connect-timeout` | `MCPDEV_CONNECT_TIMEOUT` | Handshake / discover probe deadline. |
| `--protocol` | `MCPDEV_PROTOCOL` | `auto` (default), `2025-11-25`, `2026-07-28`. |
| `--format`, `-o` | `MCPDEV_FORMAT` | `human` (default) or `json`. |
| `--allow-http-sse` | `MCPDEV_ALLOW_HTTP_SSE` | Deprecated HTTP+SSE; off by default. |
| `--insecure` | `MCPDEV_INSECURE` | Insecure TLS / Origin; off by default, never implied. |
| `--no-discover` | `MCPDEV_NO_DISCOVER` | Skip `server/discover`; recorded as skipped, not success. |
| `--trace` | `MCPDEV_TRACE` | Opt-in protocol trace on stderr (512-byte payload previews). Off by default. Does not change stdout. |
| `--verbose`, `-v` | `MCPDEV_VERBOSE` | Same stderr trace with full redacted request and response bodies. Implies tracing. |
| `--trace-format` | `MCPDEV_TRACE_FORMAT` | `human` waterfall or `json` JSON Lines. Independent of `--format`. |
| `--allow-incomplete` | | Keep `incomplete: true` in JSON but exit `0` instead of `60`. |
| `--args` | | JSON object for `call` / `prompt`. `@PATH` or `-` reads a file or stdin. |
| `--args-file` | | JSON object file (or `-` for stdin). |
| `--arg` | | Repeatable. `call`: `KEY=JSON`. `prompt` and `read --template`: raw string. |
| `--fail-on-tool-error` | | `call` only: treat `isError` as exit `30`. |
| `--template` | | `read` only: URI template expanded with string `--arg` values. |
| `--save` | | Write the full result body to a local file. |
| `--max-bytes` | | Bound stdout bodies (default `65536`). Saved files are not truncated. |
| `--out` | | `snapshot`: destination file (atomic replace). |
| `--from` / `--to` | | `diff`: snapshot file paths (or positionals `A B`). |
| `--from-command` / `--to-command` | | `diff`: live stdio argv for each side (repeatable slots). |
| `--from-url` / `--to-url` | | `diff`: live Streamable HTTP endpoints. |
| `--from-profile` / `--to-profile` | | `diff`: named profiles for each live side. |
| `--fail-on` | | `diff`: `metadata`, `additive` (default), or `breaking`. |
| `--ignore` | | `diff`: drop metadata changes at a JSON Pointer (repeatable). |
| `--rename` | | `diff`: `kind:from=to` identity map (repeatable). |
| `--no-exit-on-change` | | `diff`: keep exit `0` on `threshold-failure`. |
| `--junit PATH` | | `test`: write JUnit XML. |
| `--jobs N` | | `test`: max parallel cases (default 4). Steps inside a case stay serial. |
| `--replay PATH` | | Play a sanitized fixture. Exclusive with `--command` / `--url` / `--profile`. |
| `--record PATH` | | Write a sanitized fixture after a live command. Off by default; path required (`-` is rejected). |

`--command` together with `--url` is a usage error (exit `2`). There is no `--transport grpc`. Recording and replay are documented in [Replay fixtures](../guides/replay.md).

Operator-facing file format, interpolation, auth/TLS references, and field-scoped errors are in [Configuration](configuration.md).

## Configuration precedence

Highest wins:

1. Flags on the command line
2. `MCPDEV_*` environment variables
3. Nearest project `mcpdev.toml` or `.mcpdev.toml` (walk up from cwd)
4. User `$XDG_CONFIG_HOME/mcpdev/config.toml` (or `~/.config/mcpdev/config.toml`)
5. Built-ins: `protocol=auto`, `format=human`, HTTP+SSE and insecure **off**

```toml
protocol = "auto"
timeout = "10s"

[profiles.local]
command = ["go", "run", "."]

[profiles.http]
url = "http://127.0.0.1:8080/mcp"
```

`command` is always an argv array, not a shell line.

## stdout vs stderr

**stdout** is only the result you asked for (human text, one JSON document, a snapshot, or watch JSON Lines).

**stderr** is diagnostics, watch status (`rebuilding`, reconnect counts), usage text after a bad invocation, opt-in `--trace` / `--verbose` output, and the stdio child's stderr (bounded, not protocol).

`--format json` never prints logs on stdout. Incomplete catalogs set `incomplete: true` and a non-empty `incomplete_reasons` array. An empty list with `ok: true` means the server really listed nothing and collection finished.

## Exit codes

| Code | Class |
| --- | --- |
| `0` | Success (complete). |
| `2` | Usage or configuration. |
| `10` | Connection (spawn, HTTP, TLS, connect timeout). |
| `20` | Protocol (handshake, discover, JSON-RPC, unsupported primitive). |
| `30` | Assertion (`test`, or `call --fail-on-tool-error`). |
| `40` | Public interface changed (`diff`, or watch `--exit-on-change`). |
| `60` | Incomplete or indeterminate (stopped pagination, failed/skipped discover when needed, unevaluable assertion). |
| `70` | Internal. |

Match these numbers in CI. Do not scrape stderr.

## Examples

One-shot inspection (human):

```bash
mcpdev -c go -c run -c ./cmd/example-server inspect tools
```

JSON for a pipeline (stdout is safe to pipe):

```bash
mcpdev --format json --url http://127.0.0.1:8080/mcp inspect |
  jq '.incomplete, .result.tools'
```

CI interface diff (non-zero when the surface changed past the threshold):

```bash
mcpdev --format json --url "$MCP_URL" snapshot --out testdata/mcpdev/baseline.json
# ... rebuild the server ...
mcpdev --format json diff testdata/mcpdev/baseline.json --command ./bin/server
# outcomes: 0 no-change or below-threshold, 40 threshold-failure, 60 incomplete, 2 bad files, 70 write/internal
```

See [Snapshots and diffs](../guides/snapshots-and-diffs.md) for live-to-live flags, `--fail-on`, and `--ignore`.

Declarative tests ([guide](../guides/testing.md)):

```bash
mcpdev --profile local test ./testdata/scenarios
mcpdev --format json --command ./bin/server --junit ./tmp/junit.xml test ./suite.yaml
```

Foreground watch (status on stderr; events on stdout in JSON mode):

```bash
mcpdev --format json --command ./bin/server watch --rebuild go --rebuild build --rebuild -o --rebuild ./bin/server.partial --rebuild .
```

Watch flags: `--debounce`, `--fail-fast`, `--keep-last-good` / `--no-keep-last-good`, `--smoke`, `--baseline`, `--staging`, `--artifact`, `--on-success reconnect|none`, `--root`. See [Watch and rebuild](../guides/watch.md).


Call a tool, render a prompt, read a template:

```bash
mcpdev -c ./bin/weather call echo --args '{"text":"hi"}'
mcpdev -c ./bin/weather prompt greet --arg name=true
mcpdev -c ./bin/weather read --template 'fixture://items/{id}' --arg 'id=a b'
mcpdev --format json -c ./bin/weather read fixture://stable/blob --save ./out.bin
```

Fail closed on a truncated catalog:

```bash
mcpdev --format json -c ./server inspect
# exit 60 if pagination stopped; add --allow-incomplete only when a partial catalog is acceptable
```

## Failure modes

- Missing target or both `--command` and `--url`: exit `2`, message on stderr.
- Child never speaks MCP: exit `10` or `20` depending on whether the process died or the handshake/discover failed. JSON still has `ok: false` and no fake empty catalog.
- Modern HTTP `400` with a recognized protocol error: treated as protocol (`20`), not as "try HTTP+SSE" unless `--allow-http-sse` and the body is not a modern JSON-RPC error.
- `inspect` that stops paging: JSON `incomplete_reasons` includes `pagination_stopped`, exit `60`.

## Security

Do not pass server-returned strings to a shell. `--insecure` and `--allow-http-sse` are opt-in. Headers such as `Authorization` are redacted in snapshots and logs. Child stderr is untrusted diagnostic text.
