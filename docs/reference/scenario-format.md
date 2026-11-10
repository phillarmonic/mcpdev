---
icon: lucide/list-checks
---

# Scenario format

`internal/scenario/schema` is the versioned YAML/JSON document for `mcpdev test`. This page is the file contract. `internal/scenario/runner` connects with the same protocol services as one-shot commands and evaluates assertions. Invalid files fail at load time (usage, exit class `2`) before any MCP session is opened.

The language is deliberately small: ordered cases, one MCP operation per step, and a closed assertion set. It is not a general workflow engine. There are no loops, branches, includes, or shell steps.

Command stdout/stderr and the `test` exit classes stay those in [CLI](cli.md). Snapshot files used by compare steps stay those in [Snapshot format](snapshot-format.md).

## Document shape

YAML or JSON. The first non-space byte `{` or `[` selects JSON; anything else is YAML. Quote YAML strings that contain `{` (URI templates, interpolations) so they are not parsed as flow mappings. `schema_version` is the integer `1`. Newer versions are rejected by name. Extra properties are errors, not ignored.

| Field | Role |
| --- | --- |
| `schema_version` | Must be `1`. |
| `name` | Suite identity for reports. |
| `description` | Optional prose. |
| `target` | Exactly one of `profile`, `command` (argv array), or `url`. |
| `protocol` | Optional `auto`, `2025-11-25`, or `2026-07-28`. |
| `defaults.timeout` / `defaults.connect_timeout` | Go duration strings (`10s`). Cases may override `timeout`. |
| `variables` | String map. Literals only (no nested interpolation). |
| `secrets` | Map of `{from_env: ENV_NAME}`. No `value` field. |
| `setup` / `cleanup` | Optional hooks (see below). |
| `cases` | Non-empty. Unique `name`. Each case has a non-empty `steps` list. |

The published JSON Schema is `internal/scenario/schema/suite.v1.json` (also `schema.FormalJSON`).

### Target

Same exclusivity as the CLI: one connection. `command` is an argv array, the same rule as `mcpdev.toml` profiles. A shell string is invalid.

```yaml
target:
  profile: lab
# or
target:
  command: ["go", "run", "./cmd/server"]
# or
target:
  url: http://127.0.0.1:8080/mcp
```

### Secrets

Reference process environment. The loaded suite never stores the resolved value, so reports and round-trips cannot serialize it. At run time a missing `from_env` is an explicit missing-value error, not an empty string.

Use `${secrets.name}` inside later string fields (tool arguments, URIs). Do not put the token in `variables`.

## Steps

Each step has exactly one operation:

| Key | Meaning |
| --- | --- |
| `inspect` | List `tools`, `prompts`, `resources`, `templates`, or `all` (or omit `kind` for all). |
| `call` | `tools/call`. `arguments` is an opaque JSON object (including keys such as `command`). |
| `prompt` | `prompts/get`. Argument values are strings. |
| `read` | `resources/read`. Exactly one of `uri` or `template` (plus string `arguments` for RFC 6570). |
| `snapshot` | Compare the live public interface to a baseline file (`fail_on`, `ignore`, `rename` match `mcpdev diff`). |

Optional on any step:

- `id` for stable reports (unique in the case).
- `expect_failure.category` when the step must fail. Categories: `usage`, `connection`, `protocol`, `assertion`, `interface_change`, `incomplete`, `internal`, `tool_error` (`isError` from the server).
- `capture` entries `{name, path}` bind a JSON Path or JSON Pointer from this step's structured result to a case-local name. A missing path at run time is an error, not a silent empty capture. Names must not collide with `variables` or `secrets`.
- `assert` (closed set below).

Interpolation in strings: `${variable}`, `${capture}`, or `${secrets.name}`. It applies to step arguments, prompt/read string arguments, and the value-bearing assertion fields: `equal.value`, `contains.value`, and the string values inside an inline `schema`. Paths and closed-vocabulary fields (`error_category.equals`, `latency.max`) stay literal. Unknown names fail at load time when they can be proven missing, and at run time when a capture path is absent.

## Assertions

Exactly one kind per `assert` item:

| Kind | Check |
| --- | --- |
| `equal` | JSON Path equals `value`, or equals a `variable` / prior capture. |
| `contains` | String or array at `path` contains `value`. |
| `json_path` | `exists` and/or `count` at a JSON Path. |
| `schema` | Value at `path` matches inline JSON Schema or `file`. |
| `content_type` | MIME `equals`. |
| `latency` | Step duration `max` (Go duration). |
| `error_category` | Classified failure, same vocabulary as `expect_failure`. |

The `equal.value` and `contains.value` fields accept `${...}` interpolation (see the note under Steps), so a prior capture can be asserted without duplicating its literal. Inside inline `schema`, string values interpolate too. Prefer `variable:` on `equal` when the whole value is a binding.

Unavailable assertions are later recorded as failed or skipped by the runner. They must not be encoded as a pass in this file format.

## Setup and cleanup

Not a shell. Allowed hooks:

- `snapshot`: `{path, as?}` names a baseline file (setup only).
- `exec`: `{argv, cwd?}`. Argv array, same as profile `command`. Use this only for a local helper the operator already trusts (for example `go build`). Never interpolate server-returned strings into `argv`.

Rejected keys on the **suite document** (not inside opaque JSON) include `shell`, `sh`, `bash`, `run`, `script`, and `cmd`. `command` is only an mcpdev field on `target`, and only as an argv array. The same names are allowed as keys inside `call.arguments`, `prompt.arguments`, `read.arguments`, and assertion `value` / inline `schema` objects, because those are MCP or JSON payloads rather than mcpdev spawn configuration.

A server tool that takes a command can look like this (the `command` key is the tool argument):

```yaml
call:
  tool: run_command
  arguments:
    command: testdata/mcpdev/fixtures/vendor/bin/phpunit
    cwd: "${secrets.workspace}"
```

mcpdev itself locks that envelope with `testdata/scenarios/valid/call-command-arg.yaml`: `echo` plus a `command` argument, then `fail` with `expect_failure.category: tool_error` and `$.result.isError`. Interpreter lookup on a merged PATH belongs to the server process, not this file format.

## Examples

Valid suites live in `testdata/scenarios/valid/` (YAML covering profile targets and the stdio lab primitives, JSON covering a stdio argv target). Invalid suites in `testdata/scenarios/invalid/` cover shell keys, mixed operations, string `target.command`, unknown fields, inline secret values, and missing interpolations. Load reports every useful issue in one usage error.

```bash
mcpdev --profile lab test testdata/scenarios/valid/stdio-profile.yaml
mcpdev --command ./bin/server test testdata/scenarios/valid/stdio.json
```

How cases run, reports, and cleanup is in [Declarative tests](../guides/testing.md).

## Compatibility

| `schema_version` | Reader |
| --- | --- |
| `1` | Current. Unknown fields are errors. |
| greater than `1` | Rejected; upgrade mcpdev. |
| missing or older | Rejected. |

Additive v1 fields would need a documentation update and `omitzero` on the Go types. Do not silently coerce a newer file into v1.

## Packages

| Package | Role |
| --- | --- |
| `internal/scenario/schema` | Load YAML/JSON, reject unknown/forbidden keys, semantic checks, redacted report JSON |
| `internal/scenario/runner` | Execute cases, assertions, cleanup |
| `internal/assert` | Typed assertion results |
| `internal/app/test` | Load files, resolve targets, map exit classes |
| `internal/errclass` | Load failures are `Usage`; run failures keep their class |
