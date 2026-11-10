# Changelog

Keep this file in reverse chronological order. Every release that changes the MCP protocol matrix or the on-disk snapshot schema must name those changes in the notes (eras `2025-11-25` and `2026-07-28`; snapshot `schema_version` is currently `1`). Do not describe a schema bump as a compatible no-op.

## Unreleased

### Added

### Fixed

### Changed

## 0.1.1 - 2026-11-10

### Fixed

- Stdio shutdown errors and watch notes include the child's stderr tail when the child dies during shutdown.

### Changed

- The macOS and Windows matrix jobs run on pushes to `master`, not only on pull requests.
- The [`setup-mcpdev`](https://github.com/phillarmonic/setup-mcpdev) GitHub Action is documented in [GitHub Actions](docs/guides/github-actions.md) and linked from the README.


