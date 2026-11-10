---
icon: lucide/git-compare
---

# Interface diff policy

`internal/diff` compares two [snapshot](snapshot-format.md) documents and returns structured changes. It does not connect to a server. `mcpdev diff` loads files or live targets, then applies `--fail-on`, `--ignore`, and `--allow-incomplete` on top of this result. Operator steps are in [Snapshots and diffs](../guides/snapshots-and-diffs.md).

JSON Schema representation noise is already removed by `internal/canonical` before comparison: it never collapses composition arrays, tuples, annotations, unknown keywords, or `additionalProperties`.

## Result shape

Every run produces a `verdict`, a boolean `changed`, and a `changes` array (never `null`).

| `verdict` | `changed` | When |
| --- | --- | --- |
| `equal` | `false` | Both snapshots are complete and semantically identical. `changes` is empty. |
| `changed` | `true` | Both snapshots are complete and at least one classified difference exists. |
| `indeterminate` | `false` | Either snapshot is incomplete (`incomplete: true`, or a family `status` of `failed` / `incomplete`). |

Incomplete inputs never yield `equal`, even when collected items match. An `/incomplete` change with severity `unknown` is always present in that case. Observed item differences may still appear beside it so a reviewer can see what was collected. CI should treat `indeterminate` as incomplete (exit class `60`), not as a clean no-change and not as a definite interface change (`40`).

Capture provenance, generator version, timestamps, stopped cursors, and page counts are ignored, matching semantic snapshot bytes.

## Change records

Each change has:

| Field | Role |
| --- | --- |
| `path` | JSON Pointer (RFC 6901). `/` in identities is escaped as `~1`. |
| `category` | `addition`, `removal`, or `change`. |
| `severity` | `breaking`, `additive`, `metadata`, or `unknown`. |
| `rationale` | One-sentence explanation. Never empty. |
| `before` / `after` | Canonical JSON values. Omitted on pure additions or removals respectively. |

Identity keys:

| Family | Pointer prefix | Key |
| --- | --- | --- |
| Tools | `/tools/{name}` | Tool `name` |
| Prompts | `/prompts/{name}` | Prompt `name` |
| Resources | `/resources/{uri}` | Resource `uri` |
| Templates | `/templates/{uri_template}` | Template `uri_template` |

Examples: `/server/name`, `/capabilities/tools`, `/tools/echo/input_schema/properties/text/type`, `/resources/fixture:~1~1note`.

Changes are sorted by `path`, then `category`, then `severity`, then `rationale`. The same inputs produce the same JSON on every platform.

## Severity defaults

These rules are the default policy. They are not user-configurable in this task.

**Breaking**

- Removal of a tool, prompt, resource, or template
- Removal of a capability flag (`true` → `false`)
- Removal of an experimental name or extension name
- Protocol era, protocol version, or handshake kind change
- Server `name` change
- Required input added (new required property or prompt argument, or an existing argument/property becoming required)
- Input type narrowing (including dropping `null`, `number` → `integer`, adding `type` to an unconstrained schema)
- `enum` narrowing, or introducing `enum` where none existed
- Input property removal
- Output property removal, output `enum`/`type` narrowing, or removal of a required output name
- Mapped rename of a catalog identity (the public name or URI still changed)
- `additionalProperties` `true` → `false`
- Tightened numeric or length bounds

**Additive**

- New tool, prompt, resource, or template
- New optional input property or optional prompt argument
- Capability flag `false` → `true`
- New experimental or extension name
- Type or `enum` widening, or removing an `enum` constraint
- Required input removed
- Required output name added
- Relaxed numeric or length bounds
- `additionalProperties` `false` → `true`

**Metadata**

- Titles, descriptions, icons, annotations, `_meta`
- Server title, version string, and instructions
- Resource/template MIME type
- Schema annotations (`title`, `description`, `$comment`, `default`, `examples`)
- Protocol discover state

**Unknown**

- Incomplete snapshots (`/incomplete`)
- `additionalProperties` missing versus present (`true`, `false`, or a schema)
- Unwalkable or invalid schema JSON
- `oneOf` membership changes, `$ref` / `pattern` / `format` / unknown keywords
- Boolean schema versus object schema, and other forms that are not classified above

## Renames

A renamed item without an explicit mapping is a removal plus an addition. Supply `Engine.Renames` (`kind` plus `from` / `to` identity keys) to compare the pair as one item. Remaining field diffs use the **after** identity in the path. The identity field itself is still recorded as a breaking `change`.

## Tests

`internal/diff` goldens live in `testdata/diffs/`. Refresh them with `UPDATE_GOLDEN=1 go test ./internal/diff -run TestRulesAndGoldens`.
