package test

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/phillarmonic/mcpdev/internal/config"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/scenario/runner"
	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

// Request is one mcpdev test invocation after flag parse.
type Request struct {
	Env             map[string]string
	Paths           []string
	WorkDir         string
	Overlay         port.Target
	Jobs            int
	AllowIncomplete bool
	HasOverlay      bool
}

// Result is the aggregated suite reports for renderers.
type Result struct {
	Suites     []runner.Report `json:"suites"`
	Incomplete bool            `json:"incomplete"`
}

// Run loads suites (usage on invalid files, before connect) then executes them.
func Run(ctx context.Context, deps runner.Deps, req Request) (Result, error) {
	files, err := collectFiles(req.Paths)
	if err != nil {
		return Result{}, err
	}
	workDir := req.WorkDir
	if workDir == "" {
		workDir, err = os.Getwd()
		if err != nil {
			return Result{}, errclass.Internal{Err: err}
		}
	}
	env := req.Env
	if env == nil {
		env = map[string]string{}
	}
	out := Result{}
	for _, file := range files {
		suite, loadErr := schema.LoadFile(file)
		if loadErr != nil {
			return out, loadErr
		}
		target, tErr := resolveTarget(workDir, env, suite, req)
		if tErr != nil {
			return out, tErr
		}
		rep := runner.Run(ctx, deps, suite, runner.Request{
			File: file, WorkDir: workDir, Target: target, Env: env, Jobs: req.Jobs,
		})
		out.Suites = append(out.Suites, rep)
	}
	return out, mapExit(&out, req.AllowIncomplete)
}

func resolveTarget(cwd string, env map[string]string, suite schema.Suite, req Request) (port.Target, error) {
	if req.HasOverlay {
		t := req.Overlay
		mergeDefaults(&t, suite)
		return t, nil
	}
	settings, err := config.Load(cwd, env)
	if err != nil {
		return port.Target{}, err
	}
	t := port.Target{
		Protocol:       suite.Protocol,
		Timeout:        settings.Timeout,
		ConnectTimeout: settings.ConnectTimeout,
	}
	if t.Protocol == "" {
		t.Protocol = settings.Protocol
	}
	switch {
	case suite.Target.Profile != "":
		resolved, rerr := settings.ResolveProfile(suite.Target.Profile, env)
		if rerr != nil {
			return port.Target{}, rerr
		}
		t = resolved
		if suite.Protocol != "" {
			t.Protocol = suite.Protocol
		}
	case len(suite.Target.Command) > 0:
		t.Command = suite.Target.Command
	case suite.Target.URL != "":
		t.URL = suite.Target.URL
	default:
		return port.Target{}, errclass.Usage{Msg: "suite target is missing"}
	}
	switch {
	case t.Connector != "":
		t.Kind = t.Connector
	case t.URL != "":
		t.Kind = "http"
	default:
		t.Kind = "stdio"
	}
	mergeDefaults(&t, suite)
	return t, nil
}

func mergeDefaults(t *port.Target, suite schema.Suite) {
	if suite.Protocol != "" && suite.Protocol != "auto" {
		t.Protocol = suite.Protocol
	}
	if t.Timeout == 0 && suite.Defaults.Timeout != "" {
		if d, err := time.ParseDuration(suite.Defaults.Timeout); err == nil {
			t.Timeout = d
		}
	}
	if t.ConnectTimeout == 0 && suite.Defaults.ConnectTimeout != "" {
		if d, err := time.ParseDuration(suite.Defaults.ConnectTimeout); err == nil {
			t.ConnectTimeout = d
		}
	}
}

func mapExit(res *Result, allowIncomplete bool) error {
	var firstAssert, firstIncomplete, firstOther error
	for _, s := range res.Suites {
		switch s.Category {
		case "incomplete":
			res.Incomplete = true
			if firstIncomplete == nil {
				firstIncomplete = errclass.Incomplete{Err: errorsNew(s.Message)}
			}
		case "assertion", "tool_error", "interface_change":
			if firstAssert == nil {
				firstAssert = errclass.Assertion{Err: errorsNew(s.Message)}
			}
		case "connection":
			if firstOther == nil {
				firstOther = errclass.Connection{Err: errorsNew(s.Message)}
			}
		case "protocol":
			if firstOther == nil {
				firstOther = errclass.Protocol{Err: errorsNew(s.Message)}
			}
		case "cleanup", "internal":
			if firstOther == nil {
				firstOther = errclass.Cleanup{Err: errorsNew(s.Message)}
			}
		case "usage":
			if firstOther == nil {
				firstOther = errclass.Usage{Msg: s.Message}
			}
		default:
			if s.Status != "pass" && firstOther == nil {
				firstOther = errclass.Internal{Err: errorsNew(s.Message)}
			}
		}
	}
	if firstOther != nil {
		return firstOther
	}
	if firstIncomplete != nil {
		if allowIncomplete && firstAssert == nil {
			return nil
		}
		return firstIncomplete
	}
	return firstAssert
}

func errorsNew(msg string) error {
	if msg == "" {
		msg = "scenario failed"
	}
	return errors.New(msg)
}
