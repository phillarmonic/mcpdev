---
icon: lucide/list-checks
---

# Declarative tests

`mcpdev test` loads YAML or JSON scenario files, connects with the same stdio and Streamable HTTP path as `inspect` / `call`, and reports per-case results. Invalid files fail at load (exit class `2`) before any MCP session is opened. The file contract is [Scenario format](../reference/scenario-format.md). Exit classes are in [CLI](../reference/cli.md).

Each case opens its own session, runs steps in order, then closes the session. Remaining steps in that case are skipped after the first unexpected failure. Cleanup hooks still run. Other cases continue, up to `--jobs` (default 4) in parallel.

## Run a suite

CLI connection flags override the suite `target` when you pass `--command`, `--url`, or `--profile`:

```bash
mcpdev --command ./bin/server test testdata/scenarios/valid/stdio.json
mcpdev --profile local test ./testdata/scenarios
mcpdev --format json --jobs 2 --junit ./tmp/junit.xml test ./suite.yaml
```

Human stdout is one line per suite and case, including elapsed time. A non-passing case adds one indented line per failed or skipped assertion naming the kind, path, and diff (or reason), so the mismatch is visible without `--format json`. `--format json` writes a single envelope (`ok`, `command`, `incomplete`, `result.suites`). Suite, case, and step durations are integer `duration_ms` values (JSON v2 has no default for `time.Duration`). `--junit PATH` writes JUnit XML beside that. Assertion failures are JUnit `failure`; connection, protocol, and cleanup problems are JUnit `error`. In a workflow, [GitHub Actions](github-actions.md) installs the binary and the JUnit file uploads as an artifact.

## Failure classes

| Outcome | Exit | Report category |
| --- | --- | --- |
| All required assertions passed | `0` | `pass` |
| Assertion failed or `expect_failure` missed | `30` | `assertion` |
| Assertion path or schema could not be evaluated | `60` | `incomplete` |
| Connect or child shutdown | `10` | `connection` |
| Handshake or unsupported primitive | `20` | `protocol` |
| Always-run cleanup hook or `Close` failed | `70` | `cleanup` |
| Invalid document | `2` | (no connect) |

Unavailable assertions are `skip` records, never a pass. Cancel and recovered panics fail remaining steps without marking them passed. Child processes from `exec` hooks and MCP sessions are closed on every outcome.

## Security

`exec` hooks are argv arrays only. Do not interpolate server-returned captures into hook argv. Secret values come from `from_env` and are not stored on the loaded suite.

For protocol regressions without a live server, use [Replay fixtures](replay.md) (`--replay` / `--record`).
Official MCP requirement sets are a separate signal; they are not `mcpdev test` and they are not part of `xdrun ci`. See [Official MCP conformance](../contributing/conformance.md).

Writing suites with an agent? The [`mcpdev` skill](ai-agents.md) carries the assertion envelope, the failure classes, and copyable smoke and error templates.
