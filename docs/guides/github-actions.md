---
icon: lucide/rocket
---

# GitHub Actions

[`phillarmonic/setup-mcpdev`](https://github.com/phillarmonic/setup-mcpdev) installs the published mcpdev release binary on a runner, checks the archive against the release `SHA256SUMS`, and adds it to `PATH`. The job then runs the same binary the release published, without a Go toolchain.

## Install the binary

```yaml
- name: Checkout
  uses: actions/checkout@v7

- name: Setup mcpdev
  uses: phillarmonic/setup-mcpdev@v1

- name: Check identity
  run: mcpdev version
```

`@v1` tracks the newest 1.x action release. Pin the exact tag (`@v1.0.0`) when a workflow needs a fixed action revision.

The action tag and the mcpdev version are separate. `with.version` selects the CLI release:

| `version` | Result |
| --- | --- |
| `latest` (default) | Newest stable release |
| `v0` or `v0.1` | Newest release on that line |
| `v0.1.1` | Exact tag |

```yaml
- uses: phillarmonic/setup-mcpdev@v1
  with:
    version: 'v0.1.1'
```

Supported targets are Linux, macOS, and Windows on amd64 and arm64: `mcpdev_<version>_<os>_<arch>.tar.gz`, or `.zip` on Windows. Archive contents and build-time policy are in [Installation](../reference/installation.md).

## Outputs

| Output | Meaning |
| --- | --- |
| `version` | Installed binary version, without the `v` prefix (`0.1.1`) |
| `path` | Absolute path to the installed binary |
| `cache-hit` | Whether the binary came from the action cache |

The install step fails when the extracted binary reports a version other than the requested tag, so a mislabelled archive cannot pass silently.

## Interface gate

Commit a baseline snapshot next to the server, then diff the rebuilt binary in the job. Exit `40` means the interface changed at or above `--fail-on`; exit `60` means a catalog did not finish collecting. Both fail the step by default.

```yaml
name: MCP interface

on: [pull_request]

jobs:
  interface:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - uses: phillarmonic/setup-mcpdev@v1

      - name: Rebuild the server
        run: go build -o ./bin/server ./cmd/server

      - name: Compare against the committed baseline
        run: mcpdev diff testdata/mcpdev/baseline.json --command ./bin/server --fail-on breaking
```

Refresh the baseline locally when a change is intentional:

```bash
mcpdev snapshot --command ./bin/server --out testdata/mcpdev/baseline.json
```

Classification rules and the full exit table are in [Snapshots and diffs](snapshots-and-diffs.md).

## Declarative suites and JUnit

```yaml
- name: Run the suite
  run: mcpdev --command ./bin/server --junit ./tmp/mcpdev-junit.xml test testdata/scenarios

- name: Upload the JUnit report
  if: always()
  uses: actions/upload-artifact@v6
  with:
    name: mcpdev-junit
    path: ./tmp/mcpdev-junit.xml
```

Assertion failures become JUnit `failure` records; connection, protocol, and cleanup problems become `error` records. Suite shape and failure classes are in [Declarative tests](testing.md).

## Caching

The action caches the installed binary under a key built from the mcpdev version and the runner platform. Set `cache: 'false'` to download on every job. A restored binary still goes through the version check, and a fresh download is always checked against `SHA256SUMS`.

## Failure modes

| Symptom | Cause |
| --- | --- |
| `mcpdev: command not found` | The setup step did not run, or the runner platform is unsupported |
| Checksum mismatch | The archive and `SHA256SUMS` disagree; report the release instead of retrying |
| Version mismatch | The tag and the archive disagree; pin a different release |

Hosted runners include the `curl`, `jq`, `tar`, and checksum tools the action uses.
