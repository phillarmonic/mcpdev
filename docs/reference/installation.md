---
icon: lucide/download
---

# Installation

mcpdev is a single Go binary. Supported operating systems are **Linux**, **macOS**, and **Windows** (amd64 and arm64). The module requires **Go 1.27** (`go 1.27.3` in `go.mod`).

## go install

```bash
go install github.com/phillarmonic/mcpdev/cmd/mcpdev@latest
```

Pin a release tag when you need a known snapshot schema and protocol matrix:

```bash
go install github.com/phillarmonic/mcpdev/cmd/mcpdev@v0.1.0
```

`go install` does not apply the release `-ldflags`. Identity then comes from `runtime/debug.ReadBuildInfo`: module version, `vcs.revision`, and `vcs.time` (policy `vcs_time`). `go run` reports version `0.1.0` with policy `unknown`. `xdrun build` runs `go build`, so in a Git checkout Go's VCS stamping fills those keys: the version becomes a pseudo-version such as `0.0.0-20261110195728-cb5bf007d015`, `commit` is the full revision, `build_time` is the commit time, and the policy is `vcs_time`.

From a checkout:

```bash
xdrun install
# or
go install ./cmd/mcpdev
```

## Release archives

GitHub Releases attach checksummed archives named

`mcpdev_<version>_<os>_<arch>.tar.gz` (Linux and macOS) or `.zip` (Windows).

Each archive contains one binary (`mcpdev` or `mcpdev.exe`). `SHA256SUMS` sits next to the archives (GNU `sha256sum` two-space format).

Pack locally:

```bash
xdrun pack
```

`xdrun pack` cross-compiles with `CGO_ENABLED=0` and `-trimpath`, stamps version/commit/build time, writes `dist/`, then unpacks the **host** triple and runs `mcpdev --format json version` plus a stdio `inspect tools` against `testdata/servers/stdio-lab`. Cross-compiled archives are not executed on the pack host.

Build-time policy in `mcpdev version`:

| `build_time_policy` | When |
| --- | --- |
| `source_date_epoch` | `SOURCE_DATE_EPOCH` was set while packing |
| `git_commit` | pack used the HEAD committer unix time |
| `vcs_time` | Go embedded `vcs.time` (`go install` of a module version) |
| `unknown` | unstamped local build |

Reproducible timestamps use that unix epoch for archive member mtimes as well as `build_time`.

## Check identity

```bash
mcpdev version
mcpdev --format json version
```

JSON `result` includes `version`, `commit`, `build_time`, `build_time_policy`, `snapshot_schema` (currently `1`), `goos`, and `goarch`. A snapshot file from an older mcpdev that writes a greater `schema_version` is rejected rather than guessed.

## GitHub Actions

[`phillarmonic/setup-mcpdev`](https://github.com/phillarmonic/setup-mcpdev) installs the release archive on a runner, verifies it against `SHA256SUMS`, and adds it to `PATH`:

```yaml
- uses: phillarmonic/setup-mcpdev@v1
  with:
    version: 'v0.1.0'
```

Inputs, outputs, and CI examples are in [GitHub Actions](../guides/github-actions.md).

## Platform notes

Stdio children are launched as argv (no shell). Paths with spaces are one argv slot. On Unix, the child is a new process group (`Setpgid`). On Windows, `CREATE_NEW_PROCESS_GROUP` is set and `os.Process.Signal` / `Kill` are used; shutdown still closes stdin first. Helper and fixture binaries built in tests use a `.exe` suffix on Windows so `go build -o` and `exec` refer to the same path.

Watch mode maps fsnotify create/write/remove/rename/chmod, recursively adds new directories, and ignores `bin/`, `dist/`, and other generated trees. macOS may emit extra chmod events; they coalesce with the debounce window.

CI runs `xdrun ci` on Ubuntu and the focused process/transport/watch package set on Ubuntu, macOS, and Windows (`xdrun matrix`). Those tests do not skip core process or watch behavior per OS.
