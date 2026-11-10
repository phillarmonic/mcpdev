# mcpdev

mcpdev is a command-line client for MCP servers. It connects over stdio or
Streamable HTTP, inspects tools, prompts, resources, and templates, calls them,
snapshots a public interface and diffs it against another snapshot, runs
declarative scenario files, and watches a rebuild without restarting your AI
harness.

It treats MCP **2025-11-25** (stateful `initialize`) and **2026-07-28**
(stateless per-request `_meta`) as distinct wire eras. Protocol eras and
fallback rules are defined in [Compatibility](reference/compatibility.md).

## Install

Requires Go 1.27. Linux, macOS, and Windows are supported on amd64 and arm64.

```bash
go install github.com/phillarmonic/mcpdev/cmd/mcpdev@latest
mcpdev version
```

Checksummed archives, `build_time_policy`, and Windows notes are in
[Installation](reference/installation.md).

## Quick start

Build the in-repo stdio fixture and inspect it:

```bash
mkdir -p ./bin
go build -o ./bin/stdio-lab ./testdata/servers/stdio-lab
mcpdev inspect --command ./bin/stdio-lab
```

Run the declarative suite against the same server:

```bash
mcpdev --command ./bin/stdio-lab test testdata/scenarios/valid/stdio.json
```

Lock a baseline, then diff a later build against it:

```bash
mcpdev snapshot --command ./bin/stdio-lab --out testdata/mcpdev/baseline.json
mcpdev diff testdata/mcpdev/baseline.json --command ./bin/stdio-lab
```

`diff` exits `40` when the public interface changed and `60` when a catalog
could not be captured completely, so a pipeline can gate on the exit code
instead of parsing stderr.

## Where to go next

| Task | Page |
| --- | --- |
| Launch a local stdio server | [Stdio servers](guides/stdio.md) |
| Reach a Streamable HTTP endpoint | [Streamable HTTP](guides/http.md) |
| Inspect a rebuilt binary or endpoint | [Inspect a server](guides/inspect.md) |
| Call a tool, render a prompt, read a resource | [Call, prompt, and read](guides/operations.md) |
| Test a target declaratively | [Declarative tests](guides/testing.md) |
| Snapshot and diff interfaces | [Snapshots and diffs](guides/snapshots-and-diffs.md) |
| Record and replay fixtures | [Replay fixtures](guides/replay.md) |
| Watch a rebuild and reconnect | [Watch and rebuild](guides/watch.md) |
| See the wire exchange on stderr | [Protocol tracing](guides/tracing.md) |
| Debug framing, auth, or timeouts | [Troubleshooting](guides/troubleshooting.md) |

## Reference

| Topic | Page |
| --- | --- |
| Commands, streams, exit classes | [CLI](reference/cli.md) |
| Protocol era matrix | [Compatibility](reference/compatibility.md) |
| `mcpdev.toml`, profiles, interpolation | [Configuration](reference/configuration.md) |
| Diagnostic records and redaction | [Diagnostics and redaction](reference/diagnostics.md) |
| Pagination and capability gating | [Server inventory](reference/inventory.md) |
| On-disk interface document | [Snapshot format](reference/snapshot-format.md) |
| Change classification and severity | [Interface diff policy](reference/diff-policy.md) |
| YAML/JSON suite document | [Scenario format](reference/scenario-format.md) |
| Releases, archives, identity | [Installation](reference/installation.md) |

## Contributing

- Fixture catalog and how to add one: [Test fixtures](contributing/testing.md).
- Official requirement sets, pinned per revision: [Official MCP conformance](contributing/conformance.md).
- End-to-end stdio loop: [Stdio developer loop](tutorials/stdio-loop.md).

## Development

```bash
xdrun --list
xdrun ci
```

`xdrun ci` is the quality gate: formatting, vet, lint, tests, build, security,
vulnerability scanning, and a strict build of this site. GitHub Actions runs it
on Ubuntu for every ready pull request, plus the focused process, transport,
and watch set on Ubuntu, macOS, and Windows.
