---
icon: lucide/activity
---

# Protocol tracing

`--trace` writes a redacted log of the MCP exchange to **stderr**. Command results stay on stdout. Tracing is **off** until you pass `--trace`, `-v`/`--verbose`, or set `MCPDEV_TRACE` / `MCPDEV_VERBOSE`.

The CLI contract is in [CLI](../reference/cli.md). Secret handling is in [Diagnostics](../reference/diagnostics.md).

## Enable

Human waterfall (default, 512-byte payload previews):

```bash
mcpdev --trace --command ./bin/weather inspect tools
```

Full redacted request and response bodies (`-v` / `--verbose`):

```bash
mcpdev -v --command ./bin/weather inspect tools
```

JSON Lines (one object per event, still on stderr):

```bash
mcpdev --trace --trace-format json --format json --url http://127.0.0.1:8080/mcp inspect |
  jq .
```

`--format` is the command result. `--trace-format` is only the stderr trace. Enabling the trace does not change the JSON or human document on stdout.

Environment: `MCPDEV_TRACE`, `MCPDEV_VERBOSE`, `MCPDEV_TRACE_FORMAT=json`.

## What is recorded

Each event has a monotonic `seq` and `elapsed_ms` from the start of the process.

| `kind` | Meaning |
| --- | --- |
| `frame` | One stdio JSON-RPC line (client `→` or server `←`). |
| `http_request` | One Streamable HTTP POST or GET, with redacted headers. |
| `http_response` | Status and content type. JSON bodies get a preview; SSE bodies do not (they are `sse` events). |
| `sse` | One server-sent event from POST or GET. |
| `retry` | Discover version retry or GET SSE reconnect. Not a second copy of the frame. |
| `fallback` | Initialize after a legacy-shaped discover failure, or `--no-discover`. |
| `cancel` | Context deadline or interrupt. Not a JSON-RPC `notifications/cancelled`. |
| `stderr` | Bounded stdio **child** stderr. Never parsed as JSON-RPC. |

HTTP traces are request/response/SSE only. Stdio traces are frames only. The NDJSON pipe used to speak HTTP to the protocol layer is not also recorded as frames.

`--trace` payload previews are capped (default 512 bytes) **after** redaction. `truncated: true` means the preview was cut; the JSON document stays valid. `-v` / `--verbose` (or `MCPDEV_VERBOSE=true`) still redacts secrets but does not cap the payload.

## Security

Headers such as `Authorization`, cookies, API keys, tokens, and passwords are replaced with `[REDACTED]` before the event is stored or printed. JSON object keys with those names are redacted the same way. Child stderr is treated as untrusted text, not as a protocol frame.

Do not paste live traces that still contain secrets from server tool arguments whose keys are not in that list.

## Failure modes

| Situation | What you see |
| --- | --- |
| Trace off | No extra stderr from this feature. |
| Trace on, command succeeds | Result on stdout; ordered events on stderr (512-byte payload previews). |
| Verbose on, command succeeds | Result on stdout; ordered events on stderr with full redacted bodies. |
| Secret in `--header` | Value is `[REDACTED]` in the trace. |
| Child logs JSON-RPC on stderr | `stderr` event, not `frame` / `protocol_error`. |
