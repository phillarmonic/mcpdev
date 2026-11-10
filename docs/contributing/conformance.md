---
icon: lucide/badge-check
---

# Official MCP conformance

mcpdev scenarios, Go tests, and the official `@modelcontextprotocol/conformance` runner answer different questions. Do not treat a pass in one of them as a pass in the others.

## What each layer covers

| Layer | Command | What a pass means |
| --- | --- | --- |
| Fixture and protocol unit tests | `go test ./...` | mcpdev and `internal/testkit` behave as this repo specifies |
| Declarative scenarios | `mcpdev test` | A target matches mcpdev's YAML/JSON assertions | 
| Official wire-schema conformance | `xdrun conformance` | The official requirement set for **one** spec revision, scored at that revision's wire version |

The official runner is an extra standards signal. It is not a substitute for `xdrun ci`, and `xdrun ci` does not run it. The catalog HTTP fixtures (`legacy-http`, `modern-http`) are primitive labs, not complete MCP servers.

## Pin

The npm package and per-revision mapping live in `testdata/conformance/pin.json`.

| Field | Value |
| --- | --- |
| Package | `@modelcontextprotocol/conformance` |
| Version | `0.2.0-alpha.11` (exact; bump `pin.json` and the baselines together) |
| `2025-11-25` fixture | `legacy-http` (stateful initialize, session id) |
| `2026-07-28` fixture | `modern-http` (stateless POST, `_meta` + headers) |
| Owner | mcpdev maintainers |

Stable `0.1.16` has no `--requirements` flag and rejects `2026-07-28`. This repo stays on `0.2.0-alpha.11` until a stable 0.2 release publishes both requirement sets.

`npx` is invoked as `npx --yes @modelcontextprotocol/conformance@0.2.0-alpha.11`. Do not use `@latest`. Each revision is a separate process: `--requirements 2025-11-25` is never combined with `2026-07-28` on the same argv, and each revision writes to its own `artifacts/conformance/<revision>/` directory.

A scenario that exists in both requirement sets must be run twice. A pass under `2025-11-25` is not evidence for `2026-07-28`, and the reverse is also true.

## Run

Requires Go and Node.js 22 or newer (`npx`); the pinned runner imports `fs.globSync`, which Node 20 does not export. `.github/workflows/conformance.yml` pins Node `24`. The task starts `testdata/servers/http-lab` for the matching fixture, then the pinned runner:

```bash
xdrun conformance
xdrun conformance revision=2025-11-25
xdrun conformance revision=2026-07-28
go run ./internal/conformance/cmd/run -revision 2025-11-25
```

Human logs stay on stderr. Official `results/` files land under `artifacts/conformance/<revision>/` so the two wires cannot overwrite each other. That tree is gitignored; scheduled CI uploads it as a workflow artifact.

`xdrun ci` stays the developer gate (vet, lint, tests, build, security, docs). Conformance is `workflow_dispatch` plus a weekly schedule in `.github/workflows/conformance.yml`.

## Expected-failure baselines

| Revision | File |
| --- | --- |
| `2025-11-25` | `testdata/conformance/baselines/2025-11-25.yml` |
| `2026-07-28` | `testdata/conformance/baselines/2026-07-28.yml` |

Entries are official scenario names or `scenario:check-id` pairs. Header comments record owner and rationale. Add a name only when the failure is verified upstream or intentionally deferred for the catalog fixture. Do not copy the other revision's list to make a run green.

The runner's rules still apply: unexpected failures and stale baseline entries fail the process.

## HTTP lab

`testdata/servers/http-lab` serves a catalog HTTP fixture at `/mcp` and writes `ready.url` in its working directory. Set `MCPDEV_FIXTURE` to a catalog HTTP name (`legacy-http` by default). This is the subprocess counterpart of `testkit.StartHTTP`; production packages must not import `internal/testkit`.
