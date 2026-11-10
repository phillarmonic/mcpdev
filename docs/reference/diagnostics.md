---
icon: lucide/stethoscope
---

# Diagnostics and redaction

mcpdev records failures and status in a shared diagnostic object. One-shot commands, watch events, traces, and scenario reports use the same fields so operators and machines see the same facts. The CLI stream rules (stdout vs stderr, JSON envelopes, exit classes) are in [CLI](cli.md).

## Record fields

| Field | Meaning |
| --- | --- |
| `code` | Stable token such as `connection_failed`, `protocol_error`, `incomplete`, `child_stderr`. |
| `severity` | `info`, `warn`, or `error`. |
| `message` | Short operator text. |
| `operation` | What mcpdev was doing (`connect`, `initialize`, `discover`, `call`, …). |
| `transport` | `stdio`, `http`, or another documented adapter name. |
| `era` | `2025-11-25` or `2026-07-28` when known. |
| `elapsed_ms` | Duration of that operation. |
| `retryable` | Whether a later retry is reasonable (for example a connect deadline). |
| `cause` | Wrapped error text after redaction. |
| `child_stderr` | Bounded copy of the stdio child's stderr. |
| `details` | Nested maps (headers, config echoes, `_meta`, argv). |
| `truncated` | True when a string preview was cut to `preview_limit`. |
| `preview_limit` | Maximum bytes kept per string preview (default 512). |

Human mode writes one labeled block to stderr. JSON mode writes one object (or one JSON Lines object per watch event). Truncation shortens string fields **before** encoding so the document stays valid JSON. The same record encoded twice with the same limit is byte-identical (`encoding/json/v2` with deterministic map keys).

## Child stderr is not a protocol failure

The child's stdout is the MCP JSON-RPC stream. Its stderr is diagnostic text: logs, warnings, panic stacks. mcpdev copies a bounded prefix onto the diagnostic as `child_stderr` and onto mcpdev's own stderr.

That copy is never parsed as JSON-RPC. A child that prints `{"jsonrpc":...}` on stderr does not become `protocol_error` by itself. Connection or protocol codes come from handshake, RPC, or transport failures. `child_stderr` as a code is informational (exit class `0` unless another code is already set).

## Redaction

Every renderer runs the same redaction pass before printing or persisting.

Keys (header names, env names, query names, nested map keys), matched case-insensitively:

- `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`
- `API-Key`, `X-Api-Key`, `token`, `access_token`, `refresh_token`, `id_token`, `auth_token`
- `secret`, `client_secret`, `password`, `passwd`, `pwd`
- extra names from configuration (`sensitive = [...]` in mcpdev.toml, stdio `--env` keys, and profile header/env names)

Values are replaced with `[REDACTED]`. mcpdev does **not** scan arbitrary text for the word `token` (so `tokenizer` and `token-bucket` stay). `authorization_endpoint` is not treated as a credential key.

URLs: userinfo (`https://user:pass@host`) is stripped; query parameters with the names above are redacted. Command argv: `KEY=VALUE`, `--header`, and `--env` are redacted the same way.

Redaction applies to headers, configuration echoes, protocol `_meta`, trace payloads, child-process command descriptions, and `--record` replay fixtures. Replay files are never written until that pass finishes; see [Replay fixtures](../guides/replay.md).

## Safe bug reports

Include:

- `code`, `severity`, `operation`, `transport`, `era`, `elapsed_ms`, `retryable`, `truncated`
- the redacted `message` / `cause`
- `incomplete_reasons` when collection stopped
- argv **after** redaction (`mcpdev --command go --command run --command .` is fine; pasted `Authorization` headers are not)

Omit or replace before sending:

- raw `Authorization` / cookie / API key / token / password values
- URL userinfo and secret query strings
- live `MCPDEV_*` secrets and profile header tables
- unbounded child stderr (the diagnostic already bounds it)
- server tool argument dumps that may contain user secrets

`--format json` failures should still be one JSON object with `ok: false` on stdout plus this diagnostic on stderr. Do not paste a truncated JSON slice; if `truncated` is true, say so and keep the object intact.

## Packages

| Package | Role |
| --- | --- |
| `internal/diagnostic` | Record, codes, human/JSON renderers, child-stderr attachment. |
| `internal/trace` | Opt-in exchange log on stderr. Same redaction pass as diagnostics. |
| `internal/redact` | Key matching and recursive redaction. |
| `internal/errclass` | Exit-class mapping (`ExitCode`). |
