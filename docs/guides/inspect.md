---
icon: lucide/search
---

# Inspect a server

`mcpdev inspect` connects to one stdio binary or Streamable HTTP endpoint, lists the advertised tools, prompts, resources, and templates, and prints a labeled catalog. You do not start an Inspector UI or reconfigure an AI harness.

The command contract is in [CLI](../reference/cli.md). Collection rules (capability gating, pagination, incomplete reasons) are in [Inventory](../reference/inventory.md). Launching a child is in [Stdio servers](stdio.md). HTTP is in [Streamable HTTP](http.md).

## One command after a rebuild

Rebuild the server, then inspect the new argv. mcpdev starts a fresh child every time.

```bash
go build -o ./bin/weather ./cmd/weather
mcpdev inspect --command ./bin/weather
```

`go run` without a shell string:

```bash
mcpdev inspect --command go --command run --command ./cmd/weather
```

Streamable HTTP:

```bash
mcpdev --url http://127.0.0.1:8080/mcp inspect
```

`--protocol auto` (default) probes `server/discover` then initializes. Pin an era with `--protocol 2025-11-25` or `--protocol 2026-07-28`. `--no-discover` skips the probe and records `discover: skipped` (it is not stored as success).

## Human output

Human mode writes a short header (target, server identity, protocol, handshake, discover) then one block per family:

- `status=complete` with `0` items means the server listed nothing.
- `status=unsupported` with `reasons=capability_absent` means the capability object was missing; mcpdev did not call list.
- `status=failed` keeps any items already read and sets overall incomplete.
- `status=incomplete` keeps items and names the stop reason (`pagination_stopped`, `item_limit`, `cursor_loop`, and others).

`--expand` adds input schemas under tools. Incomplete catalogs are labeled on stdout so a redirected run is not a silent empty table. Exit class `60` unless you pass `--allow-incomplete`.

```bash
mcpdev inspect --command ./bin/weather tools
mcpdev inspect --command ./bin/weather --name echo --expand
```

Section and `--name` filters change what is printed. They do not rewrite collection status: a stopped tools walk is still incomplete if you ask only for `echo`.

## JSON for pipelines and snapshots

`--format json` writes one compact object to stdout (no logs, no ANSI). Field order is stable for a stable server. `incomplete_reasons` is always an array.

```bash
mcpdev --format json --url http://127.0.0.1:8080/mcp inspect |
  jq '{ok, incomplete, tools: .result.tools.status, names: [.result.tools.items[].name]}'
```

`ok` is false when the process exits non-zero. Incomplete collection still includes `result` lists plus `incomplete: true`. Connection and protocol failures write `ok: false` and a diagnostic line on stderr.

## Timeouts and cancel

`--connect-timeout` bounds handshake (including `server/discover`). `--timeout` bounds catalog RPCs. Interrupt cancels the context; stdio children are shut down (stdin close, then interrupt, then kill). Exit class `10` on connect timeout, `60` if listing is canceled mid-walk.

## Failure modes

| Symptom | Exit | What you see |
| --- | --- | --- |
| `--command` and `--url` together | `2` | usage on stderr |
| Child missing or dies before handshake | `10` | connection error; no fake empty catalog |
| Handshake / JSON-RPC mismatch | `20` | protocol error |
| Pagination stopped or list failed | `60` | labeled incomplete catalog |

Do not pass server-returned names to a shell. `--insecure` stays opt-in.
