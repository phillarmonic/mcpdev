---
icon: lucide/play
---

# Call, prompt, and read

`mcpdev call`, `prompt`, and `read` exercise one advertised primitive on a stdio child or Streamable HTTP endpoint. You do not route the request through an AI host. Connection flags are the same as [Inspect a server](inspect.md). The command contract is in [CLI](../reference/cli.md).

Application-level tool failures (`isError: true`) are a successful RPC. Transport and JSON-RPC failures stay on exit classes `10` and `20`.

## Tool call

JSON arguments default to `{}`. Load an object from inline text, a file, or stdin. Repeatable `--arg KEY=JSON` merges on top (later keys win). Invalid JSON is usage (exit `2`).

```bash
mcpdev --command ./bin/weather call echo --args '{"text":"hi"}'
mcpdev --command ./bin/weather call echo --args-file ./args.json
mcpdev --command ./bin/weather call echo --args - < ./args.json
mcpdev --command ./bin/weather call echo --arg text='"hi"'
```

`--fail-on-tool-error` maps server `isError` to assertion exit `30`. Without it, stdout still includes `isError: true` and the process exits `0`.

A missing tools capability, or a JSON-RPC error from the server, is protocol exit `20`. It is not an empty successful result.

## Prompt get

Prompt arguments are strings. `--arg name=true` sends the string `true`. A `--args` object may only contain strings; numbers and booleans are usage errors so a flag is not silently JSON-coerced.

```bash
mcpdev --command ./bin/weather prompt greet --arg name=true
mcpdev --format json --url http://127.0.0.1:8080/mcp prompt greet --args '{"name":"lab"}'
```

## Resource read and templates

`read URI` calls `resources/read`. `--template` expands RFC 6570 `{var}` (percent-encode) and `{+var}` (reserved) using the same string `--arg` / `--args` rules as prompts. Unexpanded or unknown operators are usage errors.

```bash
mcpdev --command ./bin/weather read fixture://stable/note
mcpdev --command ./bin/weather read --template 'fixture://items/{id}' --arg 'id=a b'
```

The expanded URI for `id=a b` is `fixture://items/a%20b`.

## Output, binary, and size

Human mode prints text. Image, audio, and blob parts are labeled with MIME type and byte length. They are never dumped as raw bytes on stdout.

`--format json` keeps `kind` on every content block. Binary is `data_base64` or `blob_base64`. `--save PATH` writes the full body (including binary) to a file you named; stdout still follows the size bound.

`--max-bytes` (default `65536`) truncates stdout text and omits oversized binary from stdout (`truncated: true`). The saved file is not truncated.

```bash
mcpdev --format json --command ./bin/weather read fixture://stable/blob --save ./out.bin --max-bytes 1
```

`--timeout` bounds the RPC. Interrupt cancels the context. Connect failures stay class `10`.

Do not pass server-returned URIs or names to a shell. `--save` is a local path you supply, not a server string.
