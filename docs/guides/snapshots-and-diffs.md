---
icon: lucide/git-compare
---

# Snapshots and diffs

`mcpdev snapshot` writes a canonical public-interface document from one live stdio binary or Streamable HTTP endpoint. `mcpdev diff` compares two of those documents, or captures one or both sides live in the same process. You do not restart an Inspector UI or an external AI harness.

On-disk shape is in [Snapshot format](../reference/snapshot-format.md). Classification rules are in [Interface diffs](../reference/diff-policy.md). Command, stdout, and exit classes are in [CLI](../reference/cli.md).

## Capture a baseline

```bash
go build -o ./bin/server ./cmd/server
mcpdev snapshot --command ./bin/server --out testdata/mcpdev/baseline.json
```

The file is replaced atomically (temp file in the same directory, then rename). stdout is the snapshot JSON document even when `--out` is set. Human mode also prints `wrote PATH` on stderr.

Incomplete collection (stopped pagination, failed list) still writes a **new** `--out` path so you can inspect what arrived, and exits `60`. If `--out` already exists, mcpdev **does not replace** that baseline. Add `--allow-incomplete` only when a partial document should exit `0`.

## Compare two files

```bash
mcpdev diff testdata/mcpdev/baseline.json testdata/mcpdev/after.json
mcpdev --format json diff --from testdata/mcpdev/baseline.json --to testdata/mcpdev/after.json
```

Human output labels both sides, then a unified-style list (`+` addition, `-` removal, `~` change) with severity and rationale. JSON is one object: `ok`, `from`, `to`, `verdict`, `outcome`, `changed`, `result.changes`, and `ignored`.

## Live-to-file and live-to-live

After a rebuild, compare the new binary to the committed baseline in one command:

```bash
go build -o ./bin/server ./cmd/server
mcpdev diff testdata/mcpdev/baseline.json --command ./bin/server
```

Compare two binaries or two HTTP endpoints without writing files:

```bash
mcpdev diff --from-command ./bin/server-old --to-command ./bin/server-new
mcpdev diff --from-url http://127.0.0.1:8080/mcp --to-url http://127.0.0.1:8081/mcp
```

`--from-command` / `--to-command` are repeatable argv slots, the same rule as `-c`.

## Exit codes for CI

| Outcome | Meaning | Default exit |
| --- | --- | --- |
| `no-change` | Complete inputs, nothing classified after ignore rules | `0` |
| `below-threshold` | Only metadata (or other severities under `--fail-on`) remain | `0` |
| `threshold-failure` | At least one change at or above `--fail-on` (unknown always counts) | `40` |
| `indeterminate` | Either snapshot is incomplete | `60` |

`--fail-on` is `additive` by default: metadata-only diffs are below threshold. `--fail-on breaking` fails only on breaking and unknown changes. `--fail-on metadata` fails on any remaining change. `--ignore /tools/echo/title` drops **metadata** at that JSON Pointer (and descendants). `--rename tools:old=new` maps identities before classification. `--no-exit-on-change` keeps exit `0` on `threshold-failure` (the JSON `outcome` still says so). `--allow-incomplete` maps `indeterminate` to `0` without calling it `equal`.

## CI example

Commit a baseline next to the server, rebuild in CI, then fail the job only when the public interface moved past the threshold:

```yaml
- name: Interface regression
  run: |
    go build -o ./bin/server ./cmd/server
    mcpdev --format json diff testdata/mcpdev/baseline.json --command ./bin/server
```

Refresh the committed file locally when a change is intentional:

```bash
mcpdev snapshot --command ./bin/server --out testdata/mcpdev/baseline.json
```

Match exit `40` or `60` in the job. Do not scrape stderr. Incomplete collection must not overwrite the committed baseline.

## Failure modes

| Symptom | Exit |
| --- | --- |
| Missing live target on `snapshot`, or only one diff side | `2` |
| Corrupt or newer `schema_version` snapshot file | `2` |
| Child or HTTP connect failure on a live side | `10` / `20` |
| Interface changed at or above `--fail-on` | `40` |
| Incomplete snapshot input, or live capture did not finish | `60` |
| `--out` I/O failure | `70` |
