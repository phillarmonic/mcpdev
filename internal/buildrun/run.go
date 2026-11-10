package buildrun

import (
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// Class is why a rebuild finished.
type Class string

const (
	ClassOK       Class = "ok"
	ClassNotFound Class = "command_not_found"
	ClassNonzero  Class = "nonzero"
	ClassTimeout  Class = "timeout"
	ClassCanceled Class = "canceled"
	ClassUsage    Class = "usage"
)

// Spec launches one argv rebuild. Argv is never passed through a shell.
type Spec struct {
	LookPath        func(string) (string, error)
	Run             func(ctx context.Context, argv []string, dir string, env []string, stdout, stderr io.Writer) error
	Env             map[string]string
	Dir             string
	ArtifactStaging string
	ArtifactFinal   string
	PreviousGood    string
	Argv            []string
	Timeout         time.Duration
	OutputLimit     int
	ClearEnv        bool
}

// Result is one classified rebuild attempt.
type Result struct {
	Err           error
	Stdout        string
	Stderr        string
	GoodPath      string
	Class         Class
	Duration      time.Duration
	ExitCode      int
	StdoutDropped int
	StderrDropped int
	Replaced      bool
}

// Run executes spec without shell interpolation.
func Run(ctx context.Context, spec Spec) Result {
	started := time.Now()
	out := Result{GoodPath: spec.PreviousGood}
	if len(spec.Argv) == 0 || strings.TrimSpace(spec.Argv[0]) == "" {
		out.Class = ClassUsage
		out.Err = errors.New("buildrun: argv is required")
		return finish(out, started)
	}
	lookPath := spec.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	argv := slices.Clone(spec.Argv)
	if !strings.ContainsRune(argv[0], os.PathSeparator) {
		resolved, err := lookPath(argv[0])
		if err != nil {
			out.Class = ClassNotFound
			out.Err = err
			return finish(out, started)
		}
		argv[0] = resolved
	}
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}
	stdout := newBoundWriter(spec.OutputLimit)
	stderr := newBoundWriter(spec.OutputLimit)
	run := spec.Run
	if run == nil {
		run = defaultRun
	}
	err := run(ctx, argv, spec.Dir, mergeEnv(spec), stdout, stderr)
	out.Stdout = stdout.String()
	out.Stderr = stderr.String()
	out.StdoutDropped = stdout.Dropped()
	out.StderrDropped = stderr.Dropped()
	out.Err = err
	out.Class = classify(ctx, err)
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		out.ExitCode = ee.ExitCode()
	}
	switch out.Class {
	case ClassOK:
		if spec.ArtifactStaging != "" && spec.ArtifactFinal != "" {
			if err := promote(spec.ArtifactStaging, spec.ArtifactFinal); err != nil {
				out.Class = ClassNonzero
				out.Err = err
				discard(spec.ArtifactStaging)
				return finish(out, started)
			}
			out.GoodPath = spec.ArtifactFinal
			out.Replaced = true
		}
	default:
		discard(spec.ArtifactStaging)
	}
	return finish(out, started)
}

func finish(out Result, started time.Time) Result {
	out.Duration = time.Since(started)
	return out
}

func classify(ctx context.Context, err error) Class {
	if err == nil {
		return ClassOK
	}
	if errors.Is(err, context.Canceled) {
		return ClassCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) || (ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)) {
		return ClassTimeout
	}
	if _, ok := errors.AsType[*exec.Error](err); ok {
		return ClassNotFound
	}
	if errors.Is(err, exec.ErrNotFound) {
		return ClassNotFound
	}
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		return ClassNonzero
	}
	return ClassNonzero
}

func defaultRun(ctx context.Context, argv []string, dir string, env []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // G204: argv is operator configuration, never shell-joined
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func mergeEnv(spec Spec) []string {
	values := make(map[string]string)
	if !spec.ClearEnv {
		for _, pair := range os.Environ() {
			key, val, ok := strings.Cut(pair, "=")
			if !ok {
				continue
			}
			values[key] = val
		}
	}
	maps.Copy(values, spec.Env)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}
