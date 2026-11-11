# mcpdev

Documentation: [phillarmonic.github.io/mcpdev](https://phillarmonic.github.io/mcpdev/)

AI agent skill: [`mcpdev`](https://github.com/phillarmonic/ai-skills/tree/master/skills/mcpdev) — install with [Repertoire](https://github.com/phillarmonic/repertoire-ai)

Connect to a changing MCP binary or HTTP endpoint, inspect it, call it, snapshot and semantically diff its public interface, and watch rebuilds without restarting an AI harness.

mcpdev is a Go command-line client. It treats MCP **2025-11-25** (stateful initialize) and **2026-07-28** (stateless per-request `_meta`) as distinct wire eras. Built-in transports are **stdio** and **Streamable HTTP**. gRPC is not a standard MCP transport. There is no browser UI, hosted tunnel, or background daemon.

## Install

Requires **Go 1.27**. Linux, macOS, and Windows (amd64 and arm64) are supported.

```bash
go install github.com/phillarmonic/mcpdev/cmd/mcpdev@latest
mcpdev version
```

From a checkout:

```bash
xdrun install
# or
go install ./cmd/mcpdev
```

Checksummed release archives, `mcpdev version` JSON fields, and Windows `.exe` notes are in [Installation](docs/reference/installation.md).

In GitHub Actions, [`phillarmonic/setup-mcpdev`](https://github.com/phillarmonic/setup-mcpdev) installs the same archive and verifies it against `SHA256SUMS`. See [GitHub Actions](docs/guides/github-actions.md).

## Use it from an AI agent

mcpdev is built to be driven by an agent: one command is one connection, results
go to stdout, diagnostics go to stderr, and every outcome is an exit class rather
than a message to parse.

The [`mcpdev` skill](https://github.com/phillarmonic/ai-skills/tree/master/skills/mcpdev)
carries the CLI contract, copyable `mcpdev.toml` profiles, smoke and error suites,
and the protocol/failure reference. Install it with
[Repertoire](https://github.com/phillarmonic/repertoire-ai), the package manager
for portable agent skills:

```bash
repertoire add mcpdev --catalog phillarmonic
```

The skill also writes an `AGENTS.md` pointer, so a repository that declares it
tells its agent when to load `mcpdev`. Targets, updates, and the package
contents: [AI agents](docs/guides/ai-agents.md).

## Quick start

The motivating loop is: rebuild a local stdio server, inspect it, lock a baseline, then let watch keep the last good binary while you edit.

Build the in-repo stdio fixture:

```bash
mkdir -p ./bin
go build -o ./bin/stdio-lab ./testdata/servers/stdio-lab
mcpdev inspect --command ./bin/stdio-lab
```

`inspect` starts a **new** child every time. You do not reconfigure Cursor, Claude, or another host. Pin an era with `--protocol 2025-11-25` or `--protocol 2026-07-28`; default `--protocol auto` probes `server/discover` then may fall back to `initialize`.

### Test

```bash
mcpdev --command ./bin/stdio-lab test testdata/scenarios/valid/stdio.json
```

YAML/JSON suite shape: [Scenario format](docs/reference/scenario-format.md). Guide: [Declarative tests](docs/guides/testing.md).

### Snapshot and diff

```bash
mcpdev snapshot --command ./bin/stdio-lab --out testdata/mcpdev/baseline.json
mcpdev diff testdata/mcpdev/baseline.json --command ./bin/stdio-lab
```

Incomplete collection still writes a **new** `--out` path and exits `60`. It does not replace an existing baseline. CI should match exit `40` (interface change) and `60` (indeterminate), not scrape stderr. Guide: [Snapshots and diffs](docs/guides/snapshots-and-diffs.md).

### Watch (rebuild, keep last good, reconnect)

`--rebuild` is an argv array (no `sh -c`). Successful builds rename `--staging` onto `--artifact`. Failed builds leave the last good file in place. `--command` is the argv that runs after a successful promote.

```bash
mcpdev --command ./bin/weather watch \
  --rebuild go --rebuild build --rebuild -o --rebuild ./bin/weather.partial --rebuild ./cmd/weather \
  --staging ./bin/weather.partial --artifact ./bin/weather --debounce 200ms
```

Ctrl-C cancels in-flight rebuilds and shuts the child down (stdin close, then interrupt, then kill). Full flags and generation rules: [Watch and rebuild](docs/guides/watch.md).

Streamable HTTP uses `--url` instead of `--command` (never both):

```bash
mcpdev --url http://127.0.0.1:8080/mcp inspect
```

## Documentation

The Zensical site is built from `docs/` (`xdrun ci` includes a strict docs build).

| Topic | Page |
| --- | --- |
| Install and `version` | [Installation](docs/reference/installation.md) |
| Install the release binary in CI | [GitHub Actions](docs/guides/github-actions.md) |
| Drive the CLI from an AI agent | [AI agents](docs/guides/ai-agents.md) |
| Commands, stdout/stderr, exit classes | [CLI](docs/reference/cli.md) |
| Protocol eras and fallback | [Compatibility](docs/reference/compatibility.md) |
| Inspect | [Inspect a server](docs/guides/inspect.md) |
| Tests | [Declarative tests](docs/guides/testing.md) |
| Snapshot / diff | [Snapshots and diffs](docs/guides/snapshots-and-diffs.md) |
| Watch | [Watch and rebuild](docs/guides/watch.md) |
| Framing, negotiation, auth, timeouts, leftover processes | [Troubleshooting](docs/guides/troubleshooting.md) |

The site index groups every page by task: [docs/index.md](docs/index.md).

## Development

Repository automation is defined with [drun](https://github.com/phillarmonic/drun), which provides the `xdrun` interpreter. The spec lives in `.drun/spec.drun`; the drun skill is part of the same [Repertoire](https://github.com/phillarmonic/repertoire-ai) catalog as `mcpdev`.

```bash
xdrun --list
xdrun ci
```

`xdrun ci` is the required quality gate (format, vet, lint, tests, build, security, vulnerabilities, documentation). Do not tag, push, or publish from this workflow unless the repository owner asks for that specific action.

## License

MIT. See [LICENSE](LICENSE).
