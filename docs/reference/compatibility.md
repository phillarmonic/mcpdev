---
icon: lucide/shield-check
---

# Compatibility

mcpdev speaks two first-class MCP wire eras: **2025-11-25** (stateful `initialize` / `initialized`) and **2026-07-28** (stateless per-request `_meta`). Both are required. The era matrix, negotiation rules, and fallback behavior are below.

## What mcpdev supports

| Kind | Status | Notes |
| --- | --- | --- |
| MCP `2025-11-25` | First-class | Handshake lifecycle, optional `Mcp-Session-Id`, session-scoped list-change notifications. |
| MCP `2026-07-28` | First-class | `server/discover`, `_meta` on every request, Streamable HTTP POST-only, `subscriptions/listen`. |
| stdio | Built-in | Standard transport in both eras. |
| Streamable HTTP | Built-in | Standard transport; era selects session/GET vs POST-only. |
| HTTP+SSE (`2024-11-05`) | Compatibility mode | Deprecated. Off unless explicitly enabled. |
| gRPC | Adapter only | **Not** an MCP-standard transport. No default CLI transport named `grpc`. Mapping is unspecified until a real protobuf service exists. |

## Negotiation (do not treat every 400 as legacy)

**HTTP.** Try a 2026-07-28 request first. On HTTP 400, read the body. Modern errors (`UnsupportedProtocolVersionError` `-32022`, `MissingRequiredClientCapabilityError` `-32021`, `HeaderMismatch` `-32020`) mean a modern server: retry or fix the request. Fall back to `initialize` only when the body is empty or not a recognized modern JSON-RPC error.

**stdio.** Probe `server/discover` with per-request `_meta` (`io.modelcontextprotocol/protocolVersion`, `clientInfo`, `clientCapabilities`). A `DiscoverResult` or a recognized modern error stays in the stateless era. Any other error or a timeout may fall back to `initialize`.

**Version policy.** mcpdev prefers `2026-07-28`. It walks that preference list and takes the first string also present in `supportedVersions` (or in `UnsupportedProtocolVersionError.data.supported`). It retries discover once with the selected version. No mutually supported modern version is a protocol error, not a silent initialize.

Session diagnostics record `requested` (the first version asked), `discovered` (`supportedVersions` when present), `negotiated` (`protocolVersion` after success), and `fallback` (true only when initialize was used after a legacy-shaped discover failure).

Sources: [Streamable HTTP backward compatibility](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http#backward-compatibility), [stdio backward compatibility](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio#backward-compatibility). Retrieved and re-checked 2026-09-19.

## Legacy initialize diagnostics

A 2025-11-25 session is ready only after a successful `initialize` result and a following `notifications/initialized` notification. mcpdev records that path as `handshake: initialize`. Inventory commands can then read negotiated `protocolVersion`, `serverInfo`, `instructions`, and declared capabilities (tools, prompts, resources, and list-change/subscribe flags).

Older initialize-era dates the official Go SDK still speaks (`2025-06-18`, `2025-03-26`) stay on the same stateful lifecycle. The session era remains `2025-11-25`; `protocolVersion` keeps the dated string the server returned. HTTP+SSE (`2024-11-05`) is still off unless `--allow-http-sse` is set.

Handshake failures stay classified. They are never turned into an empty successful session.

| Failure | Typical class | What you see |
| --- | --- | --- |
| Connect timeout or canceled handshake | Connection (`10`) | Deadline or cancel; no session. |
| Server exit or closed pipe during initialize | Connection (`10`) | Transport ended before a valid result. |
| Unsupported `protocolVersion` | Protocol (`20`) | Dated version is not a supported initialize-era string. |
| Malformed JSON or `capabilities` that is not an object | Protocol (`20`) | Decode failed; no session. |
| Tool, prompt, or resource RPC before `initialized` | Protocol (`20`) | Stateful servers reject the call (`-32600` on fixtures). |
| RPC after `Close` | Connection (`10`) | Session is closed. |

`--protocol 2025-11-25` sends `initialize` first. `--protocol auto` may probe `server/discover` and then fall back to `initialize` when the peer is a handshake server. `notifications/initialized` is sent only after a valid initialize result.

## Incomplete results

Stopped pagination, a failed `server/discover`, an unsupported primitive, or an unused subscription MUST be reported as such. They are not empty successful catalogs.

## Official references

Retrieved and re-checked 2026-09-19 (overview pages still distinguish stateful `2025-11-25` from stateless `2026-07-28`; gRPC is not listed as a standard transport).

- [Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [Specification 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28)
- [2026-07-28 announcement](https://blog.modelcontextprotocol.io/posts/2026-07-28/)
- [Go SDK](https://go.sdk.modelcontextprotocol.io/protocol/)
- [Inspector](https://github.com/modelcontextprotocol/inspector)
- [Conformance suite](https://github.com/modelcontextprotocol/conformance)
