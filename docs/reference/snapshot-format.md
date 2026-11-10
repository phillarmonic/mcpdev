---
icon: lucide/camera
---

# Snapshot format

`internal/snapshot` writes a versioned JSON document of one server's public interface: identity, negotiated protocol, capabilities, and collected tools, prompts, resources, and templates. Inspect still prints a live catalog. This file is the portable contract [interface diffs](diff-policy.md) compare.

Collection rules (capability gating, pagination, incomplete reasons) stay those in [Inventory](inventory.md). stdout vs stderr and exit classes stay those in [CLI](cli.md).

## Document shape

The file is one compact JSON object plus a trailing newline, encoded with `encoding/json/v2` and `json.Deterministic(true)` so equivalent maps have sorted keys.

| Field | Role |
| --- | --- |
| `schema_version` | Integer document schema. This mcpdev writes `1`. |
| `generator` | `{name, version}` of the mcpdev that wrote the file. |
| `captured_at` | UTC timestamp (RFC 3339). |
| `provenance` | Connection context: `kind`, `profile`, `url`, `command`, `headers`, `session_id`. |
| `protocol` | `era`, negotiated `version`, `handshake`, `discover`. Eras are never collapsed. |
| `server` | `name`, `title`, `version`, `instructions`. |
| `capabilities` | Presence flags plus `experimental` / `extensions` name lists. |
| `incomplete` | True when an advertised family failed or stopped. |
| `incomplete_reasons` | Always an array. Empty when collection finished. |
| `tools` / `prompts` / `resources` / `templates` | One section per collected family. Omitted when that family was not requested. |

Each section has `status`, `reasons` (always an array), `items`, `count`, and optionally `stopped_cursor` and `pages`.

### Semantic vs diagnostic fields

Semantic comparison (byte-identical for equivalent inventories) includes `schema_version`, `protocol`, `server`, `capabilities`, `incomplete`, `incomplete_reasons`, and each section's `status`, `reasons`, `count`, and `items` (names, titles, descriptions, schemas, annotations, icons, `_meta`, URIs / URI templates).

These fields are stored and **omitted** from semantic bytes:

- `generator`
- `captured_at`
- `provenance` (including session id, URL, argv, headers)
- section `stopped_cursor` and `pages`

A later capture of the same interface against a different URL or mcpdev version therefore compares equal. An incomplete walk does **not** compare equal to a complete walk of the same names.

## Nested JSON

Tool `input_schema` / `output_schema` run through `internal/canonical` (sorted keys, sorted `required` / `enum` / `type` unions, empty `required` omitted). Composition arrays, tuples, annotations, unknown keywords, and `additionalProperties` stay as authored. Item `_meta` is key-sorted generic JSON plus redaction. Resource `uri` and template `uri_template` strings are stored verbatim (no `url.Parse` rewrite, including spaces in `file:` paths). JSON numbers in schemas are encoded by `encoding/json/v2` after decode.

`_meta` and provenance headers/URLs/argv run through `internal/redact` before persist. `Authorization` and `token` values become `[REDACTED]`. URL userinfo is replaced the same way (the standard library may percent-encode the replacement token).

## Incomplete documents

`incomplete: true` is required when any section `status` is `failed` or `incomplete`. Encode and decode reject a file that claims `incomplete: false` in that case. Items already collected stay in `items`; they are not turned into an empty success.

Unsupported families (`status: unsupported`, `reasons: ["capability_absent"]`) do not set `incomplete`.

## Compatibility and migration

| On-disk `schema_version` | Reader behavior |
| --- | --- |
| `1` | Current. Extra properties are ignored. Missing optional properties default to empty. |
| greater than `1` | Rejected with a usage error that names both versions. mcpdev does not guess a conversion. |
| missing, `0`, or other older integers | Rejected as unsupported or corrupt. There is no v0 on-disk format. |

To introduce a breaking snapshot change, increment `schema_version` and add an explicit reader branch (or a documented conversion command). Do not silently rewrite a newer file into v1. Additive v1 fields may appear later with `omitzero`; readers of this version skip unknown keys. `mcpdev --format json version` reports the schema this binary writes as `result.snapshot_schema`. Record schema bumps in `CHANGELOG.md`.

Corrupt JSON, a missing `schema_version`, or an invalid `captured_at` is a usage error (exit class `2` when the CLI surfaces it). Write I/O failures are internal (class `70`). Incomplete collection remains class `60`. `mcpdev snapshot --out` writes through a temporary file in the same directory and renames it into place. A **new** `--out` path may still receive an incomplete document. An **existing** baseline file is left untouched when collection did not finish.

## Golden files

`testdata/snapshots/` holds compact v1 documents for both first-class eras, a metadata-rich tool schema (`$defs`, numbers), all four primitive families, and an incomplete tools walk (`pagination_stopped`). Refresh them with `UPDATE_GOLDEN=1 go test ./internal/snapshot -run TestGoldens`.

## Packages

| Package | Role |
| --- | --- |
| `internal/snapshot` | Document encode/decode, `_meta` JSON, validation, `SnapshotStore`, semantic digests |
| `internal/canonical` | JSON Schema canonicalization and schema digests |
| `internal/model` | `Snapshot` document types |
| `internal/inventory` | Live collection that `Capture` walks |
| `internal/redact` | Credentials in provenance and `_meta` |
