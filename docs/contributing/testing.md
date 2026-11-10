---
icon: lucide/flask-conical
---

# Test fixtures

Transport, inspect, diff, scenario, and watch tests reuse these fixtures
instead of inventing ad hoc servers.

The laboratory lives in `internal/testkit`. Production packages (`internal/app`,
`internal/cli`, adapters) must not import it. The package does not import
`github.com/modelcontextprotocol/go-sdk`; that import stays in
`internal/protocol`.

Protocol eras and transports follow [Compatibility](../reference/compatibility.md).

## Named fixtures

Look up a descriptor by catalog name. An unknown name is an error, not a silent
default.

| Name | Era | Transport | What it exercises |
| --- | --- | --- | --- |
| `legacy-stdio` | `2025-11-25` | stdio | `initialize`, `initialized`, tools, prompts, resources, templates |
| `legacy-http` | `2025-11-25` | HTTP | Session id, `MCP-Protocol-Version`, same primitives |
| `legacy-paginate` | `2025-11-25` | stdio | `nextCursor` on `tools/list` |
| `legacy-notify` | `2025-11-25` | stdio | `notifications/tools/list_changed` after `initialized` |
| `modern-stdio` | `2026-07-28` | stdio | `server/discover`, per-request `_meta`, listings; `initialize` is `-32601` |
| `modern-http` | `2026-07-28` | HTTP | Standard headers, GET/DELETE `405`, `HeaderMismatch` |
| `modern-sse` | `2026-07-28` | HTTP | Request-scoped SSE on `tools/call` when `Accept` includes `text/event-stream` |
| `modern-auth` | `2026-07-28` | HTTP | `Authorization: Bearer fixture-token` |
| `modern-need-cap` | `2026-07-28` | stdio | `server/discover` returns `-32021` MissingRequiredClientCapabilityError |
| `modern-version-retry` | `2026-07-28` | stdio | First `server/discover` returns `-32022` with `supported: ["2026-07-28"]`; the next discover succeeds |
| `delay-stdio` | `2025-11-25` | stdio | Intentional delay before each RPC (cancellation tests) |
| `malformed-stdio` | `2025-11-25` | stdio | Non-JSON reply to the first RPC |
| `malformed-http` | `2026-07-28` | HTTP | `400` with an empty body |
| `unrecognized-http` | `2025-11-25` | HTTP | `400` with an unrecognized JSON-RPC error |
| `legacy-unsupported` | `2025-11-25` | stdio | `initialize` returns `protocolVersion` `1999-01-01` |
| `legacy-bad-caps` | `2025-11-25` | stdio | `initialize` returns `capabilities` as an array |
| `legacy-exit` | `2025-11-25` | stdio | Closes the stream after the first RPC |

Stable inventory (every well-formed catalog that advertises primitives):

- Tools: `echo`, `fail` (`isError: true`). Pagination fixtures add `extra-00`…
- Prompt: `greet` with required argument `name`
- Resources: `fixture://stable/note` (text body `stable-note`) and `fixture://stable/blob` (binary, `application/octet-stream`)
- Template: `fixture://items/{id}`

## In-process stdio

```go
d, err := testkit.Named("legacy-stdio")
if err != nil {
    t.Fatal(err)
}
lab, err := testkit.StartStdio(d)
if err != nil {
    t.Fatal(err)
}
t.Cleanup(func() { _ = lab.Close() })
```

`StartStdio` uses `io.Pipe`. Protocol JSON-RPC is newline-delimited on the
client's write/read ends. Diagnostics that would be child stderr are captured
in `lab.Stderr`. `Close` cancels the serve loop and waits for it; tests must
not `time.Sleep` for shutdown. Prefer `testing/synctest` for delay and
cancellation.

## Subprocess stdio

`testdata/servers/stdio-lab` is a tiny `main` that calls `testkit.RunStdio`.
Set `MCPDEV_FIXTURE` to a catalog name (default `legacy-stdio`).

```go
lab := testkit.StartProcess(t, "legacy-stdio")
```

The helper builds the binary into a temp dir, starts it, and registers cleanup.
Stdout is only JSON-RPC. Stderr is a bounded buffer of fixture logs. Never treat
stderr as protocol.

## HTTP

```go
lab := testkit.StartHTTP(t, d)
```

`StartHTTP` uses `httptest.NewServer` on loopback and registers `Server.Close` on the test.
`testdata/servers/http-lab` is the subprocess counterpart (`MCPDEV_FIXTURE`, URL written to `ready.url` in the process working directory). Official requirement-set scoring uses that helper; see [Official MCP conformance](conformance.md).
`StartHTTPS` is the same with `httptest.NewTLSServer` (use `lab.Server.Client()` when the test must trust the fixture cert).
The server binds the fixture's era:

- `2025-11-25`: optional `Mcp-Session-Id` after `initialize`; GET may hold an SSE stream with `id` fields. `DropGETOnce` closes the first GET after one event so reconnect tests can send `Last-Event-ID`. DELETE tears the session down; later POST/GET/DELETE are `404`.
- `2026-07-28`: POST only; every request needs `_meta` plus `MCP-Protocol-Version` and `Mcp-Method`. `tools/call`, `prompts/get`, and `resources/read` also need `Mcp-Name` (Base64 sentinel when the value is not header-safe ASCII). Header/body version or name mismatch is `-32020` / HTTP `400`. JSON-RPC notifications (no `id`) return `202` with an empty body. GET/DELETE are `405`.

Auth, empty-body, and unrecognized-error fixtures keep those defects explicit.
Callers must not treat an empty `400` as a successful catalog.

## Add a fixture

1. Add a `Descriptor` to the catalog map in `internal/testkit/catalog.go`.
2. Reuse `Capabilities`, `Delay`, `Malformed`, `Session`, `PageSize`, and
   `Shutdown` instead of forking a new handler when a flag is enough.
3. If you need new RPC behavior, add a small handler file; do not grow a single
   kitchen-sink file past the architecture size guardrail.
4. Add a focused `*_test.go` that proves the new path (including shutdown).
5. Document the name in the table above in the same change.

## Commands

```bash
go test ./internal/testkit
xdrun matrix
xdrun ci
xdrun conformance
```

`xdrun matrix` is the OS-portable process, transport, and watch set also run on Ubuntu, macOS, and Windows in GitHub Actions. It must not skip those packages per GOOS. Full `xdrun ci` stays Ubuntu-oriented (Unix `gofmt` check, docs, security). Release archives and `go install` are documented in [Installation](../reference/installation.md).
