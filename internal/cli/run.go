package cli

import (
	"context"
	"io"
	"maps"
	"os"
	"os/signal"
	"strings"

	diffapp "github.com/phillarmonic/mcpdev/internal/app/diff"
	"github.com/phillarmonic/mcpdev/internal/app/inspect"
	"github.com/phillarmonic/mcpdev/internal/app/invoke"
	apptest "github.com/phillarmonic/mcpdev/internal/app/test"
	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/config"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/render"
	"github.com/phillarmonic/mcpdev/internal/replay"
	"github.com/phillarmonic/mcpdev/internal/trace"
	"github.com/phillarmonic/mcpdev/internal/transport/registry"
	"github.com/phillarmonic/mcpdev/internal/version"
)

// Run parses argv, executes one command, and returns an exit class.
func Run(args []string, stdout, stderr io.Writer) int {
	in, err := parseArgs(args)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	if in.Help {
		_, _ = io.WriteString(stdout, helpText)
		return errclass.ExitOK
	}
	if in.Command == "" {
		_, _ = io.WriteString(stderr, helpText)
		return errclass.ExitUsage
	}
	switch in.Command {
	case "version":
		return runVersion(in, stdout, stderr)
	case "inspect":
		return runInspect(in, stdout, stderr)
	case "call", "prompt", "read":
		return runInvoke(in, stdout, stderr)
	case "snapshot":
		return runSnapshot(in, stdout, stderr)
	case "diff":
		return runDiff(in, stdout, stderr)
	case "test":
		return runTest(in, stdout, stderr)
	case "watch":
		return runWatch(in, stdout, stderr)
	default:
		return fail(stdout, stderr, in, errclass.Usage{Msg: "unknown command " + in.Command})
	}
}

func runVersion(in Invocation, stdout, stderr io.Writer) int {
	info := version.Current()
	if in.Format == model.FormatJSON {
		if err := render.VersionJSON(stdout, info); err != nil {
			_ = stderr
			return errclass.ExitInternal
		}
		return errclass.ExitOK
	}
	if err := render.VersionHuman(stdout, info); err != nil {
		return errclass.ExitInternal
	}
	_ = stderr
	return errclass.ExitOK
}

func runInspect(in Invocation, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	target, format, err := resolveTarget(in)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	kinds, err := parseSection(in.Section)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	res, runErr := inspect.Run(ctx, newConnector(in, stderr, target.Sensitive), inspect.Request{
		Target:          target,
		Names:           in.Names,
		Kinds:           kinds,
		Expand:          in.Expand,
		AllowIncomplete: in.AllowIncomplete,
	})
	res.Target = target
	if format == model.FormatJSON {
		if err := render.InspectJSON(stdout, res, runErr); err != nil {
			_ = writeln(stderr, err.Error())
			return errclass.ExitInternal
		}
	} else if runErr == nil || res.Inventory.Tools.Kind != "" || res.Inventory.Prompts.Kind != "" || res.Inventory.Resources.Kind != "" || res.Inventory.Templates.Kind != "" {
		if err := render.InspectHuman(stdout, res); err != nil {
			return errclass.ExitInternal
		}
	}
	if runErr != nil {
		_ = writeln(stderr, runErr.Error())
		return errclass.ExitCode(runErr)
	}
	return errclass.ExitOK
}

func runInvoke(in Invocation, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if in.FailOnToolError && in.Command != "call" {
		return fail(stdout, stderr, in, errclass.Usage{Msg: "--fail-on-tool-error is only valid with call"})
	}
	if in.Template != "" && in.Command != "read" {
		return fail(stdout, stderr, in, errclass.Usage{Msg: "--template is only valid with read"})
	}
	target, format, err := resolveTarget(in)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	req := invoke.Request{
		Stdin:           os.Stdin,
		ArgPairs:        in.ArgPairs,
		Kind:            invoke.Kind(in.Command),
		Name:            in.Section,
		URI:             in.Section,
		Template:        in.Template,
		ArgsJSON:        in.ArgsJSON,
		ArgsFile:        in.ArgsFile,
		SavePath:        in.SavePath,
		Target:          target,
		MaxBytes:        in.MaxBytes,
		FailOnToolError: in.FailOnToolError,
	}
	if in.Command != "read" {
		req.URI = ""
	} else {
		req.Name = ""
	}
	res, runErr := invoke.Run(ctx, newConnector(in, stderr, target.Sensitive), req)
	res.Target = target
	res.Command = in.Command
	if format == model.FormatJSON {
		if err := render.InvokeJSON(stdout, res, runErr); err != nil {
			_ = writeln(stderr, err.Error())
			return errclass.ExitInternal
		}
	} else if runErr == nil || res.Tool != nil || res.Prompt != nil || res.Resource != nil {
		if err := render.InvokeHuman(stdout, res); err != nil {
			return errclass.ExitInternal
		}
	}
	if runErr != nil {
		_ = writeln(stderr, runErr.Error())
		return errclass.ExitCode(runErr)
	}
	return errclass.ExitOK
}

func newConnector(in Invocation, stderr io.Writer, extra []string) port.Connector {
	if in.Replay != "" {
		return replay.File{Path: in.Replay, Clock: clock.Wall{}}
	}
	keys := append(recordExtraKeys(in), extra...)
	live := registryConnector{reg: registry.Builtins(), tracer: startTracer(in, stderr, keys)}
	if in.Record != "" {
		return replay.Recording{Inner: live, Path: in.Record, ExtraKeys: keys}
	}
	return live
}

func recordExtraKeys(in Invocation) []string {
	extra := make([]string, 0, len(in.Env)+len(in.Headers))
	for _, p := range in.Env {
		k, _, ok := strings.Cut(p, "=")
		if ok {
			extra = append(extra, k)
		}
	}
	for _, h := range in.Headers {
		name, _, ok := strings.Cut(h, ":")
		if ok {
			extra = append(extra, strings.TrimSpace(name))
		}
	}
	return extra
}

func startTracer(in Invocation, stderr io.Writer, extra []string) *trace.Recorder {
	enabled := in.Trace || in.Verbose
	full := in.Verbose
	if v := os.Getenv("MCPDEV_TRACE"); v != "" {
		b, err := config.ParseBoolEnv(v)
		if err == nil && b {
			enabled = true
		}
	}
	if v := os.Getenv("MCPDEV_VERBOSE"); v != "" {
		b, err := config.ParseBoolEnv(v)
		if err == nil && b {
			enabled = true
			full = true
		}
	}
	if !enabled {
		return nil
	}
	format := trace.FormatHuman
	switch {
	case in.TraceFormat == model.FormatJSON:
		format = trace.FormatJSON
	case os.Getenv("MCPDEV_TRACE_FORMAT") == "json":
		format = trace.FormatJSON
	}
	return trace.New(trace.Options{Writer: stderr, Format: format, ExtraKeys: extra, Full: full})
}

func resolveTarget(in Invocation) (port.Target, model.Format, error) {
	return resolveTargetSpec(in, in.CommandArgv, in.URL, in.Profile, true, true)
}

func loadFormat(in Invocation) (model.Format, error) {
	if in.FormatSet {
		return in.Format, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return in.Format, errclass.Internal{Err: err}
	}
	settings, err := config.Load(cwd, envMap(os.Environ()))
	if err != nil {
		return in.Format, err
	}
	return settings.Format, nil
}

func resolveTargetSpec(in Invocation, cmd []string, url, profile string, envOverlay bool, required bool) (port.Target, model.Format, error) {
	if in.AllowHTTPSse {
		return port.Target{}, in.Format, errclass.Usage{Msg: "--allow-http-sse is not implemented"}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return port.Target{}, in.Format, errclass.Internal{Err: err}
	}
	env := envMap(os.Environ())
	settings, err := config.Load(cwd, env)
	if err != nil {
		return port.Target{}, in.Format, err
	}
	if settings.AllowHTTPSSE {
		return port.Target{}, in.Format, errclass.Usage{Msg: "--allow-http-sse is not implemented"}
	}
	format := settings.Format
	if in.FormatSet {
		format = in.Format
	}
	protocol := settings.Protocol
	if in.ProtocolSet {
		protocol = in.Protocol
	}
	switch protocol {
	case "", "auto", string(model.Era20251125), string(model.Era20260728):
	default:
		return port.Target{}, format, errclass.Usage{Msg: "unknown --protocol " + protocol}
	}
	if envOverlay {
		if profile == "" {
			profile = env["MCPDEV_PROFILE"]
		}
		if len(cmd) == 0 {
			envCmd, cmdErr := config.ParseCommandEnv(env["MCPDEV_COMMAND"])
			if cmdErr != nil {
				return port.Target{}, format, cmdErr
			}
			cmd = envCmd
		}
		if url == "" {
			url = env["MCPDEV_URL"]
		}
	}
	var fromProfile port.Target
	if profile != "" {
		if len(cmd) > 0 || url != "" {
			return port.Target{}, format, errclass.Usage{Msg: "--profile cannot be combined with --command or --url"}
		}
		fromProfile, err = settings.ResolveProfile(profile, env)
		if err != nil {
			return port.Target{}, format, err
		}
		cmd = fromProfile.Command
		url = fromProfile.URL
	}
	if in.Record != "" && in.Replay != "" {
		return port.Target{}, format, errclass.Usage{Msg: "--record cannot be combined with --replay"}
	}
	if in.Replay != "" {
		if len(cmd) > 0 || url != "" || profile != "" {
			return port.Target{}, format, errclass.Usage{Msg: "--replay cannot be combined with --command, --url, or --profile"}
		}
		if in.Record != "" {
			return port.Target{}, format, errclass.Usage{Msg: "--record cannot be combined with --replay"}
		}
		return port.Target{Kind: "replay"}, format, nil
	}
	sources := 0
	if len(cmd) > 0 {
		sources++
	}
	if url != "" {
		sources++
	}
	if fromProfile.Connector != "" {
		sources++
	}
	switch {
	case sources == 0 && !required:
		return port.Target{}, format, nil
	case sources != 1:
		return port.Target{}, format, errclass.Usage{Msg: "exactly one of --command, --url, --profile, or --replay is required"}
	}
	if in.Record == "-" {
		return port.Target{}, format, errclass.Usage{Msg: "--record requires a file path"}
	}
	insecure := settings.Insecure || fromProfile.AllowInsecure || in.Insecure
	skip := in.SkipDiscover
	if v := env["MCPDEV_NO_DISCOVER"]; v != "" && !in.SkipDiscover {
		b, boolErr := config.ParseBoolEnv(v)
		if boolErr != nil {
			return port.Target{}, format, boolErr
		}
		skip = b
	}
	headers, err := parseHeaders(in.Headers, env["MCPDEV_HEADER"])
	if err != nil {
		return port.Target{}, format, err
	}
	childEnv, err := parseEnvPairs(in.Env)
	if err != nil {
		return port.Target{}, format, err
	}
	timeout := settings.Timeout
	if in.TimeoutSet {
		timeout = in.Timeout
	}
	connect := settings.ConnectTimeout
	if in.ConnectSet {
		connect = in.ConnectTimeout
	}
	dir := in.CWD
	if dir == "" {
		dir = env["MCPDEV_CWD"]
	}
	if dir == "" {
		dir = fromProfile.Dir
	}
	if len(fromProfile.Headers) > 0 {
		merged := maps.Clone(fromProfile.Headers)
		maps.Copy(merged, headers)
		headers = merged
	}
	if len(fromProfile.Env) > 0 {
		merged := maps.Clone(fromProfile.Env)
		maps.Copy(merged, childEnv)
		childEnv = merged
	}
	kind := "stdio"
	switch {
	case fromProfile.Connector != "":
		kind = fromProfile.Connector
	case url != "":
		kind = "http"
	}
	if fromProfile.Protocol != "" && !in.ProtocolSet {
		protocol = fromProfile.Protocol
	}
	sensitive := config.ExtraKeys(append(append([]string{}, settings.Sensitive...), fromProfile.Sensitive...), headers, childEnv)
	return port.Target{
		Headers:        headers,
		Env:            childEnv,
		Kind:           kind,
		Profile:        profile,
		URL:            url,
		Connector:      fromProfile.Connector,
		Origin:         fromProfile.Origin,
		Protocol:       protocol,
		Dir:            dir,
		CAFile:         fromProfile.CAFile,
		CertFile:       fromProfile.CertFile,
		KeyFile:        fromProfile.KeyFile,
		Command:        cmd,
		Sensitive:      sensitive,
		Timeout:        timeout,
		ConnectTimeout: connect,
		AllowInsecure:  insecure,
		SkipDiscover:   skip,
	}, format, nil
}

func fail(stdout, stderr io.Writer, in Invocation, err error) int {
	if in.Format == model.FormatJSON || in.FormatSet && in.Format == model.FormatJSON {
		target := port.Target{Kind: "", Command: in.CommandArgv, URL: in.URL, Profile: in.Profile}
		switch in.Command {
		case "call", "prompt", "read":
			_ = render.InvokeJSON(stdout, invoke.Result{Command: in.Command, Name: in.Section, URI: in.Section, Target: target}, err)
		case "snapshot":
			_ = render.CommandErrorJSON(stdout, "snapshot", err)
		case "diff":
			_ = render.DiffJSON(stdout, diffapp.Result{From: diffapp.Source{Path: in.From, Label: in.From}, To: diffapp.Source{Path: in.To, Label: in.To}}, err)
		case "test":
			_ = render.TestJSON(stdout, apptest.Result{}, err)
		default:
			_ = render.InspectJSON(stdout, inspect.Result{Target: target}, err)
		}
	}
	_ = writeln(stderr, err.Error())
	return errclass.ExitCode(err)
}

func writeln(w io.Writer, s string) error {
	_, err := io.WriteString(w, s+"\n")
	return err
}

func envMap(pairs []string) map[string]string {
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		out[k] = v
	}
	return out
}

func parseHeaders(flags []string, env string) (map[string]string, error) {
	out := map[string]string{}
	add := func(raw string) error {
		name, val, ok := strings.Cut(raw, ":")
		if !ok {
			return errclass.Usage{Msg: "header must be Name: value"}
		}
		out[strings.TrimSpace(name)] = strings.TrimSpace(val)
		return nil
	}
	for _, h := range flags {
		if err := add(h); err != nil {
			return nil, err
		}
	}
	if env != "" {
		for part := range strings.SplitSeq(env, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if err := add(part); err != nil {
				return nil, err
			}
		}
	}
	if len(out) == 0 {
		return map[string]string{}, nil
	}
	return out, nil
}

func parseEnvPairs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, errclass.Usage{Msg: "--env must be KEY=VALUE"}
		}
		out[k] = v
	}
	return out, nil
}
