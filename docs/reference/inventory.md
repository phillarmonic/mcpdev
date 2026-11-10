---
icon: lucide/list
---

# Server inventory

`internal/inventory` walks `tools/list`, `prompts/list`, `resources/list`, and `resources/templates/list` after handshake. The result is one deterministic document: server identity, negotiated era and version, advertised capabilities, and every collected primitive (or an explicit reason the family was skipped, failed, or stopped).

Inspect, snapshot, and diff consume this document. They never treat a stopped walk as an empty catalog.

## What is collected

| Field | Source |
| --- | --- |
| Server name, title, version | `initialize` / `server/discover` `serverInfo` |
| Era, protocol version, handshake, discover state | Negotiation recorded on the session |
| Capabilities | Presence of `tools`, `prompts`, and `resources` objects |
| Tools | Name, title, description, input/output schema, annotations, icons, `_meta` |
| Prompts | Name, title, description, arguments, icons, `_meta` |
| Resources and templates | URI or URI template, name, title, description, MIME type, size, annotations, icons, `_meta` |

Annotations are untrusted hints. Snapshots keep them so later diffs can see them; they are not used as authorization or safety checks.

## Capability gating

A family is listed only when the matching capability object was present. Resource templates use the `resources` capability.

| Capability object | List RPCs |
| --- | --- |
| `tools` | `tools/list` |
| `prompts` | `prompts/list` |
| `resources` | `resources/list` and `resources/templates/list` |

If the object is absent, the section status is `unsupported` with reason `capability_absent`. mcpdev does **not** call the list method. That is different from a completed list that returned zero items (`complete`, empty, no reasons).

## Pagination

Cursors are opaque. Each list call sends the previous `nextCursor` (or omits it on the first page) until:

- the server omits `nextCursor` (collection finished),
- the same cursor repeats (`cursor_loop`),
- `MaxPages` is reached (`pagination_stopped`),
- `MaxItems` is reached (`item_limit`),
- a page is marked incomplete by the protocol adapter (`page_incomplete`),
- the list RPC fails (`list_failed`),
- the context is done (`collection_canceled`).

Defaults: 256 pages and 10,000 items per family. Zero limits in code mean those defaults, not unbounded collection. The stopped cursor (when collection did not finish) is stored on the section so operators can see where the walk halted.

Duplicate names (tools/prompts) or URIs/templates keep the first item and record `duplicate_item` on the section. That is not a dropped page; the inventory stays complete unless another stop reason applies.

## Status vs empty

| Section status | Meaning | Overall `incomplete` |
| --- | --- | --- |
| `complete` | Every page was read (the list may be empty) | no |
| `unsupported` | Capability object missing; no RPC | no |
| `failed` | A list RPC returned an error (prior items are kept) | yes |
| `incomplete` | Loop, limit, canceled context, or adapter-incomplete page | yes |

Overall `incomplete` is true when any advertised family is `failed` or `incomplete`. `unsupported` does not set it. Exit class `60` applies when inspect/snapshot surfaces an incomplete inventory; see [CLI](cli.md).

Stable reason tokens:

- `capability_absent`
- `list_failed`
- `pagination_stopped`
- `cursor_loop`
- `item_limit`
- `page_incomplete`
- `duplicate_item`
- `collection_canceled`

## Determinism

Repeated collection against a stable server produces deeply equal inventories: items are sorted by name (tools/prompts) or URI / URI template (resources/templates), and reason lists are sorted. There are no timestamps on the document.

## Packages

| Package | Role |
| --- | --- |
| `internal/inventory` | Cursor walking, limits, gating, ordering |
| `internal/protocol` | One-page `List` and SDK conversion |
| `internal/model` | `Inventory`, `CatalogSection`, descriptors |
| `internal/snapshot` | Versioned on-disk contract; see [Snapshot format](snapshot-format.md) |
