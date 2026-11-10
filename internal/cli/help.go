package cli

const helpText = `mcpdev is a direct CLI for MCP servers (stdio and Streamable HTTP).

Usage:
  mcpdev [<global flags>] <command> [<command flags>] [args]

Commands:
  inspect [tools|prompts|resources|templates]  list primitives
  call NAME                                    tools/call
  prompt NAME                                  prompts/get
  read URI                                     resources/read
  snapshot                                     write a canonical interface document
  diff A B                                     compare two snapshots or live targets
  test PATH [PATH...]                          run declarative scenario files
  watch                                        rebuild, reconnect, and report events
  version                                      print version, commit, and snapshot schema

Connection (exactly one):
  -c, --command VALUE   stdio argv slot (repeatable)
      --url URL         Streamable HTTP endpoint
  -p, --profile NAME    named target from mcpdev.toml

      --header NAME: VALUE   HTTP header (repeatable)
      --env KEY=VALUE        stdio child env (repeatable)
      --cwd DIR              stdio working directory
      --timeout DURATION     per-RPC deadline (default 30s)
      --connect-timeout D    handshake deadline (default 15s)
      --protocol auto|2025-11-25|2026-07-28
  -o, --format human|json
      --insecure             opt in to insecure TLS / Origin
      --no-discover          skip server/discover (recorded as skipped)
      --trace                opt-in protocol trace on stderr (off by default)
  -v, --verbose              full redacted request/response bodies on stderr
      --trace-format human|json
      --replay PATH          play a sanitized fixture (no live server)
      --record PATH          write a sanitized fixture after the command (off by default)
      --allow-incomplete     exit 0 when a catalog did not finish
      --name NAME            inspect: keep matching names/URIs
      --expand               inspect: include schemas in human output
      --args JSON            call/prompt: object, @file, or - for stdin
      --args-file PATH       JSON object file (or - for stdin)
      --arg KEY=VALUE        call: KEY=JSON; prompt/read template: raw string
      --fail-on-tool-error   call: exit 30 when the tool sets isError
      --template URI         read: RFC 6570 template expanded with --arg
      --save PATH            write the full body (including binary) to a file
      --max-bytes N          bound stdout bodies (default 65536)
      --out PATH             snapshot: atomic file write
      --from PATH            diff: before snapshot file
      --to PATH              diff: after snapshot file
      --from-command VALUE   diff: live before argv slot (repeatable)
      --to-command VALUE     diff: live after argv slot (repeatable)
      --from-url URL         diff: live before HTTP endpoint
      --to-url URL           diff: live after HTTP endpoint
      --fail-on LEVEL        diff: metadata|additive|breaking (default additive)
      --ignore POINTER       diff: ignore metadata at a JSON Pointer (repeatable)
      --rename kind:from=to  diff: map a catalog identity (repeatable)
      --no-exit-on-change    diff: keep exit 0 when the interface changed
      --junit PATH           test: write JUnit XML
      --jobs N               test: max parallel cases (default 4)
      --rebuild VALUE        watch: rebuild argv slot (repeatable)
      --root DIR             watch: filesystem root (repeatable; default cwd)
      --include GLOB         watch: extra include glob (repeatable)
      --exclude GLOB         watch: extra exclude glob (repeatable)
      --debounce DURATION    watch: coalesce window (default 200ms)
      --fail-fast            watch: exit on first failed rebuild or probe
      --keep-last-good       watch: overlap until probe succeeds (default)
      --no-keep-last-good    watch: stop the current child before starting the next
      --smoke NAME           watch: call tool NAME after inventory (repeatable)
      --baseline PATH        watch: comparison snapshot before the first ready generation
      --staging PATH         watch: rebuild writes here, then rename onto --artifact
      --artifact PATH        watch: last-good binary path
      --on-success POLICY    watch: reconnect (default) or none after a good rebuild
      --readiness MODE       watch: inventory (default) or scenario
      --exit-on-change       watch: exit 40 when a ready generation changes the interface

Exit classes: 0 ok, 2 usage, 10 connection, 20 protocol, 30 assertion, 40 interface change, 60 incomplete, 70 internal.
stdout is the result only. Diagnostics go to stderr.
`
