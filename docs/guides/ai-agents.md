---
icon: lucide/bot
---

# AI agents

mcpdev is designed to be driven by an agent. One command opens one connection,
the result goes to stdout, diagnostics stay on stderr, and every outcome is an
exit class. Nothing needs a browser, a daemon, or a host restart.

The rules that make that work are packaging, not guesswork. The `mcpdev` skill
teaches an agent which target flags exist, which exit number means a truncated
catalog, and which assertion paths fail quietly.

## Install the skill

The skill lives in the [`phillarmonic/ai-skills`](https://github.com/phillarmonic/ai-skills)
catalog and installs with [Repertoire](https://github.com/phillarmonic/repertoire-ai),
a package manager for portable agent skills:

```bash
repertoire add mcpdev --catalog phillarmonic
```

The official `phillarmonic` catalog is built in. Add `--catalog phillarmonic`
when another visible catalog also defines `mcpdev`, or when you want this one
explicitly. Without `--target`, Repertoire detects existing agent configuration;
`--target codex`, `--target claude`, `--target opencode`, or `--target all`
choose explicitly. `--project` installs into the current Git worktree instead of
the home directory.

Check what landed and keep it fresh:

```bash
repertoire show mcpdev
repertoire update mcpdev
```

## What the package contains

| Path | Contents |
| --- | --- |
| `SKILL.md` | Target selection, the inspect-then-exercise loop, snapshot and diff, scenarios, watch, and the exit table |
| `references/scenarios.md` | Suite document, assertion data model, interpolation, hooks, and report classes |
| `references/protocol-and-failures.md` | Wire eras, negotiation mistakes, incomplete catalogs, auth and timeout symptoms |
| `assets/scenarios/smoke.yaml` | Copyable happy-path suite with `$.result.*` paths, secrets, and a latency assertion |
| `assets/scenarios/errors.yaml` | `expect_failure` for a tool error versus a protocol failure |
| `assets/mcpdev.toml` | stdio and HTTP profiles, including a memory-safe fixture target |
| `pointers/agents.md` | The managed `AGENTS.md` section |

Repertoire installs `instructions:` artifacts on every `add`, `install`, and
`update`, so the `AGENTS.md` pointer arrives without a flag. The section tells an
agent to load the skill before hand-rolling a client, which is the difference
between a two-command check and a half-written JSON-RPC implementation.

## Traps the skill encodes

| Trap | Correct behavior |
| --- | --- |
| Parsing human output | Use `--format json`, then read fields; stdout is only the result |
| Treating a stopped page walk as empty | Exit `60`; `--allow-incomplete` is an explicit choice |
| Scraping stderr for failures | Branch on exit `0/2/10/20/30/40/60/70` |
| Reusing a child between checks | Each command starts a new stdio child |
| Writing `$.isError` in a scenario | Use `$.result.isError`; the bare path skips with exit `60` |
| Forgetting `isError` semantics | A tool error is RPC success unless `--fail-on-tool-error` |
| Assuming exit `10` for a missing command | On Windows an absolute `--command` path that does not exist is exit `2` |

## The same binary in CI

The skill and the action come from the same release.
[`setup-mcpdev`](https://github.com/phillarmonic/setup-mcpdev) installs the
archive, checks it against `SHA256SUMS`, and puts `mcpdev` on `PATH`:

```yaml
- uses: phillarmonic/setup-mcpdev@v1
  with:
    version: 'v0.1.1'
- run: mcpdev diff testdata/mcpdev/baseline.json --command ./bin/server
```

An agent that makes a change and a pipeline that gates it then run the same
command. Details: [GitHub Actions](github-actions.md).

## Reference

- [CLI](../reference/cli.md): flags, streams, and the exit table.
- [Declarative tests](testing.md): suite behavior and JUnit mapping.
- [Installation](../reference/installation.md): archives and build identity.
- [Skill source](https://github.com/phillarmonic/ai-skills/tree/master/skills/mcpdev).
