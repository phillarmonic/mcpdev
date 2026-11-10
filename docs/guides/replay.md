---
icon: lucide/cassette-tape
---

# Replay fixtures

`--replay PATH` connects to a **sanitized** JSON fixture instead of a live stdio or HTTP server. `--record PATH` writes that fixture after a live command. Recording is **off** until you pass `--record` with a real file path (`-` is rejected).

Use this for regressions around negotiation, pagination, errors, and timing without keeping credentials or depending on a running process. Protocol tracing is a stderr log; replay is a portable document. See [Protocol tracing](tracing.md) and [Diagnostics](../reference/diagnostics.md).

## Record

```bash
mcpdev --command ./bin/server --record ./testdata/replays/session.json inspect tools
```

The file is replaced atomically (temp + rename) with mode `0600`. Headers, env, `_meta`, and JSON keys such as `authorization`, `token`, `password`, and `api_key` are replaced with `[REDACTED]` **before** the bytes hit disk. Extra `--env` / `--header` names are treated as sensitive keys too.

Review the file before you commit or share it. Redaction is by **key name**, not by scanning values. A tool argument named `note` that happens to contain a password is not removed. Do not share a fixture until you have read every `params` and `payload` object.

## Replay

```bash
mcpdev --replay ./testdata/replays/legacy-init.json inspect tools
mcpdev --replay ./testdata/replays/pagination.json --format json inspect tools
```

`--replay` cannot be combined with `--command`, `--url`, `--profile`, or `--record`. The session metadata in the document (era, handshake, discover state) is restored; JSON-RPC methods are matched in order.

A mismatch names the **first** problem: wrong method, `_meta` (metadata) JSON Pointer, or params/payload JSON Pointer. Example: `replay: step 2 method tools/list: payload mismatch at /cursor`.

## Document shape

`schema_version` is `1`. `kind` is `mcpdev.replay`.

Each `steps[]` entry has:

- `expect.method` (required when `expect` is present): `initialize`, `server/discover`, `tools/list`, `tools/call`, and the other MCP methods.
- `expect.params` / `expect._meta`: subset match. Every expected key must equal the actual value; extra actual keys are ignored.
- `deliver[]` (required): ordered `result`, `error`, `notification`, `sse`, `malformed`, or `process_exit`.
- `delay_ms` on a delivery: wait via the injectable clock (no wall-clock `Sleep` in tests).

Checked-in examples live in `testdata/replays/`:

| File | What it covers |
| --- | --- |
| `legacy-init.json` | Initialize-era session + `tools/list`. |
| `modern-discover.json` | `server/discover` result (`supportedVersions`). |
| `modern-session.json` | Discover-era session after handshake. |
| `pagination.json` | Two `tools/list` pages and a cursor. |
| `http-sse.json` | SSE event after a list result. |
| `malformed-frame.json` | Non-JSON server line. |
| `process-exit.json` | Child exit during `tools/list` (connection class `10`). |

## Timing

Delays, timeouts, and cancellation use `port.Clock` (`After`). Tests should drive `testing/synctest` or `clock.Virtual` / `Advance`. Do not add `time.Sleep` to make a replay test pass.

## Failure modes

| Situation | What you see |
| --- | --- |
| No `--record` | Nothing is written. |
| `--record -` or missing path | Usage, exit `2`. |
| `--replay` plus `--command` | Usage, exit `2`. |
| Method or payload mismatch | Protocol class `20`, pointer to the first field. |
| `process_exit` | Connection class `10`. |
| Secret key in params | `[REDACTED]` in the saved file. |
