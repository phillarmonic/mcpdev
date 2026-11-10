---
icon: lucide/globe
---

# Streamable HTTP

mcpdev speaks **Streamable HTTP** as a standard MCP transport in both first-class eras. Operator flags are in [CLI](../reference/cli.md). Era rules are in [Compatibility](../reference/compatibility.md). `--protocol` (or negotiation) selects the adapter: sessionful `2025-11-25` or POST-only `2026-07-28`.

## Why this shape still exists

Protocol version `2026-07-28` removed protocol sessions. POST is request-scoped; there is no `Mcp-Session-Id`, no GET SSE hang, and no DELETE. Many servers, SDKs, and dated conformance suites still speak `2025-11-25`. mcpdev keeps that adapter so you can inspect those endpoints without pretending they are modern.

This is **not** the deprecated two-endpoint **HTTP+SSE** transport from `2024-11-05`. That path stays off unless you pass `--allow-http-sse`. Snapshots and default `connect` must not label HTTP+SSE as current.

| Era | HTTP shape |
| --- | --- |
| `2025-11-25` | Session optional. `MCP-Protocol-Version` on requests. POST JSON or SSE. GET may hold SSE. DELETE ends a minted session. Bounded GET reconnect may send `Last-Event-ID`. |
| `2026-07-28` | POST only. No protocol session. Request-scoped SSE. `MCP-Protocol-Version`, `Mcp-Method`, and encoded `Mcp-Name` on requests. Do not send `Mcp-Session-Id`. |
| `2024-11-05` HTTP+SSE | Deprecated. Opt-in compatibility only. |

## Connect

```bash
mcpdev --url http://127.0.0.1:8080/mcp inspect
```

Loopback `http` is allowed. Non-loopback `http` and skipped TLS verification need `--insecure` (or `MCPDEV_INSECURE`). They are never implied.

```bash
mcpdev --url https://mcp.example.com/mcp inspect
mcpdev --insecure --url http://10.0.0.5:8080/mcp inspect
```

Repeatable `--header 'Name: value'` is sent on every request. `Authorization` and other credential names are redacted in diagnostics. See [Diagnostics](../reference/diagnostics.md).

```bash
mcpdev --url https://mcp.example.com/mcp \
  --header 'Authorization: Bearer TOKEN' \
  inspect tools
```

Profiles (auth belongs in `header_env` or `${TOKEN:?}`, not inline; see [Configuration](../reference/configuration.md)):

```toml
[profiles.http]
url = "http://127.0.0.1:8080/mcp"

[profiles.http.header_env]
Authorization = "MCPDEV_TOKEN"
```

`--command` together with `--url` is a usage error (exit `2`).

## Session lifecycle (2025-11-25)

1. POST `initialize` with `MCP-Protocol-Version: 2025-11-25` and `Origin` set from the endpoint (or an explicit origin). The server may return `Mcp-Session-Id`.
2. Later POST/GET/DELETE include that session id and the same protocol version. JSON and `text/event-stream` POST bodies are both accepted; SSE is decoded to the JSON-RPC payload, not treated as empty success.
3. GET opens the optional standalone SSE stream for server-initiated messages (for example `notifications/tools/list_changed`). If the stream drops, mcpdev reconnects a bounded number of times and sends `Last-Event-ID` when it has one. `2026-07-28` must not use this reconnect path.
4. Close sends DELETE for a minted session. A `404` on DELETE means the session is already gone; that is teardown, not a successful inspect.

An unknown or expired session is HTTP `404` (connection class `10`), not an empty catalog.

## Stateless POST (2026-07-28)

Every RPC is one POST to the MCP endpoint. mcpdev does not mint, store, or send `Mcp-Session-Id`. Two POSTs to the same URL (or to two load-balanced backends) are independent. GET and DELETE are not used; a modern-only peer answers those with `405`.

Each request includes:

- `Accept: application/json, text/event-stream`
- `MCP-Protocol-Version` matching `params._meta` `io.modelcontextprotocol/protocolVersion`
- `Origin` derived from the URL (or an explicit origin)
- `Mcp-Method` from the JSON-RPC `method`
- `Mcp-Name` for `tools/call`, `prompts/get`, and `resources/read` (`params.name` or `params.uri`)

If `Mcp-Name` is not a safe ASCII header value (non-ASCII, control characters, leading or trailing whitespace, or a value that already looks like `=?base64?...?=`), it is sent as `=?base64?{standard-Base64}?=` (markers are lowercase and case-sensitive).

The server may answer a request with a JSON object or a request-scoped SSE stream. mcpdev decodes both to the JSON-RPC payload. Notifications on that stream belong to the request; they are not a catalog. A JSON-RPC notification POST (no `id`) expects HTTP `202` with an empty body.

Cancellation closes the HTTP response stream (the client cancels the request context). mcpdev does **not** POST `notifications/cancelled` on Streamable HTTP. That notification is stdio-only in this era.

`--protocol 2026-07-28` stays on this adapter. `--protocol auto` still probes modern first; recognized `400` errors (`HeaderMismatch` `-32020` and kin) stay modern. See [Compatibility](../reference/compatibility.md).

## Security

- Send `Origin` on every request. Servers must reject a present-but-invalid Origin with `403` (DNS rebinding). mcpdev sets Origin from the endpoint unless you override it.
- Bind local MCP servers to loopback (`127.0.0.1` / `localhost`), not `0.0.0.0`.
- Loopback `http` is allowed. Non-loopback `http` and skipped TLS verification need `--insecure` (or `MCPDEV_INSECURE`). They are never implied.
- TLS certificate errors are connection failures. `--insecure` skips verify only when you opt in.
- Send credentials with `--header`. Auth (`401`/`403`) and TLS failures stay actionable (`authentication failed`) without echoing `Authorization` values.
- Do not pass server-returned strings to a shell.

## Failure modes

| Symptom | Typical class | What to check |
| --- | --- | --- |
| Dial/TLS/refused | `10` connection | URL, certs, `--insecure` only if you mean it |
| HTTP `401`/`403` | `10` connection | `--header`; logs will show `[REDACTED]` |
| HTTP `404` after initialize | `10` connection | Session expired; re-handshake |
| HTTP `400` JSON-RPC | `20` protocol | Version/header mismatch; do not assume HTTP+SSE |
| GET/DELETE `405` | `20` protocol | Peer is likely `2026-07-28` POST-only |

Modern `400` bodies with recognized JSON-RPC errors are **not** a cue to fall back to HTTP+SSE. See [compatibility](../reference/compatibility.md).
