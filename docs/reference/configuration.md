---
icon: lucide/settings
---

# Configuration

mcpdev loads a versioned TOML file so you can reuse named stdio, HTTP, and custom connector targets without putting secrets on the command line. Flags and the [CLI](cli.md) still win for a single process. Redaction of credential-shaped fields is in [Diagnostics](diagnostics.md).

## File locations

| Layer | Path |
| --- | --- |
| Project | `mcpdev.toml` (preferred) or `.mcpdev.toml`, walking from the current working directory toward the filesystem root. The nearest file wins. |
| User | `$MCPDEV_CONFIG` if set (error if the file is missing). Otherwise `$XDG_CONFIG_HOME/mcpdev/config.toml`. On Windows, `%APPDATA%\mcpdev\config.toml` when `XDG_CONFIG_HOME` is unset. Otherwise `~/.config/mcpdev/config.toml`. |

A missing default user file is skipped. A missing project file is skipped. `MCPDEV_CONFIG` pointing at a missing path is a usage error and names that path.

## Precedence (highest wins)

1. Command-line flags for the current process
2. Process environment (`MCPDEV_*`)
3. Project config file (nearest)
4. User config file
5. Built-in defaults (`protocol=auto`, `format=human`, insecure and HTTP+SSE off)

A flag that is present with an empty value is still a flag. Unset environment variables do not override config files. Boolean env values: `1`, `true`, `TRUE`, `yes` are true; `0`, `false`, `FALSE`, `no`, empty are false. Any other string is a usage error.

Project profiles replace a user profile of the same name entirely (they are not field-merged).

## Document shape

`version = 1` is required when `version` is present. Other versions are usage errors that name the `version` field. A document without `version` is treated as version 1.

```toml
version = 1
protocol = "auto"
timeout = "10s"
connect_timeout = "15s"
format = "human"
sensitive = ["X-Lab-Token"]

[profiles.local]
command = ["go", "run", "."]
cwd = "./server"

[profiles.http]
url = "https://mcp.example.com/mcp"

[profiles.http.header_env]
Authorization = "MCPDEV_TOKEN"

[profiles.http.tls]
ca_file = "/etc/ssl/certs/ca.pem"

[profiles.custom]
connector = "fake"
```

`command` is an argv array only. A shell string is a usage error on that profile's `command` field.

Exactly one of `command`, `url`, or `connector` is required on each profile. Combining them is a usage error that names `profiles.<name>`. `connector = "grpc"` is refused (gRPC is not a standard MCP transport).

## Environment interpolation

String fields may contain `${NAME}`, `${NAME:?}`, or `${NAME:-default}`. `$$` is a literal `$`.

| Form | Missing or empty value |
| --- | --- |
| `${NAME}` or `${NAME:?}` | Usage error naming the profile field and the variable |
| `${NAME:-default}` | Substitutes `default` |

Interpolation runs after files and `MCPDEV_*` scalars are merged, using the process environment. `header_env` values are **variable names**, not interpolated strings.

## Auth and TLS references

Do not put bearer tokens, passwords, or PEM blobs in the file.

| Mechanism | Role |
| --- | --- |
| `[profiles.name.header_env]` | Header name to environment variable. The variable must be set and non-empty when the profile is used. |
| `[profiles.name.headers]` with `${NAME:?}` | Same idea inline. Literal values on credential-shaped names (`Authorization`, `token`, `password`, names in `sensitive`, …) are refused. |
| `[profiles.name.tls]` `ca_file` / `cert_file` / `key_file` | Paths (interpolated). `cert_file` and `key_file` must be set together. Inline PEM is refused. |
| `url` userinfo (`https://user:pass@host`) | Refused. Use `header_env`. |

Flags `--header` / `--env` overlay profile maps (the flag wins per key).

## Insecure combinations

Loopback `http` is allowed. Non-loopback `http` and skipped TLS verification need an explicit opt-in:

- `--insecure`
- `MCPDEV_INSECURE` (true values above)
- `insecure = true` in a config file (file or profile or `tls.insecure`)

File and environment opt-ins append a warning on the loaded settings. They never turn on from `--url` alone. `allow_http_sse = true` is accepted as an explicit key but mcpdev still reports that HTTP+SSE is not implemented.

## Errors

Usage errors (exit class `2`) name the **file path**, **profile**, and **field** when those are known:

```text
config: /path/mcpdev.toml: profiles.http.headers.Authorization: inline secret refused; use header_env or ${NAME:?}
```

Secret values are not copied into the message.

## Redaction

`sensitive` and credential-shaped header/env keys feed the same redaction pass used by inspect JSON, traces, snapshots, scenario reports, and replay fixtures. Renderers substitute `[REDACTED]`. See [Diagnostics](diagnostics.md).

```bash
mcpdev --profile http --format json inspect tools
```
