package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	appwatch "github.com/phillarmonic/mcpdev/internal/app/watch"
	"github.com/phillarmonic/mcpdev/internal/buildrun"
	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/process"
	"github.com/phillarmonic/mcpdev/internal/protocol"
	watchrender "github.com/phillarmonic/mcpdev/internal/render/watch"
	"github.com/phillarmonic/mcpdev/internal/snapshot"
	"github.com/phillarmonic/mcpdev/internal/supervisor"
	"github.com/phillarmonic/mcpdev/internal/version"
	fswatch "github.com/phillarmonic/mcpdev/internal/watch"
	"github.com/phillarmonic/mcpdev/internal/watch/cycle"
)

// testWatchSource and testWatchPorts are nil in production. Tests inject them.
var (
	testWatchSource fswatch.Source
	testWatchPorts  *appwatch.Ports
	testWatchClock  port.Clock
)

func runWatch(in Invocation, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runWatchCtx(ctx, in, stdout, stderr)
}

func runWatchCtx(ctx context.Context, in Invocation, stdout, stderr io.Writer) int {
	target, format, err := resolveTarget(in)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	if len(target.Command) == 0 {
		return fail(stdout, stderr, in, errclass.Usage{Msg: "watch requires --command (stdio child)"})
	}
	if !in.FormatSet {
		loaded, fmtErr := loadFormat(in)
		if fmtErr != nil {
			return fail(stdout, stderr, in, fmtErr)
		}
		format = loaded
	}
	exitOnChange := in.ExitOnChangeSet && in.ExitOnChange
	onSuccess := appwatch.OnSuccessReconnect
	switch in.OnSuccess {
	case "", "reconnect":
	case "none":
		onSuccess = appwatch.OnSuccessNone
	default:
		return fail(stdout, stderr, in, errclass.Usage{Msg: "unknown --on-success " + in.OnSuccess})
	}
	keepLast := true
	if in.KeepLastGoodSet {
		keepLast = in.KeepLastGood
	}
	debounce := 200 * time.Millisecond
	if in.DebounceSet {
		debounce = in.Debounce
	}
	roots := in.WatchRoots
	if len(roots) == 0 {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fail(stdout, stderr, in, errclass.Internal{Err: cwdErr})
		}
		roots = []string{cwd}
	}
	matcher := fswatch.NewMatcher(in.Include, in.Exclude)
	var source fswatch.Source
	var sourceClose func() error
	if testWatchSource != nil {
		source = testWatchSource
	} else {
		fs, openErr := fswatch.OpenFS(roots, matcher)
		if openErr != nil {
			return fail(stdout, stderr, in, errclass.Internal{Err: openErr})
		}
		source = fs
		sourceClose = fs.Close
	}
	if sourceClose != nil {
		defer func() { _ = sourceClose() }()
	}

	var human *watchrender.Writer
	emit := func(n fswatch.Note) {
		if format == model.FormatJSON {
			_ = watchrender.JSONL(stdout, n)
			return
		}
		if human == nil {
			human = &watchrender.Writer{Out: stderr, Mode: watchrender.Detect(stderr, os.Environ())}
		}
		_ = human.Write(n)
	}
	defer func() {
		if human != nil {
			_ = human.Flush()
		}
	}()

	clk := testWatchClock
	if clk == nil {
		clk = clock.Wall{}
	}
	store := snapshot.Store{Version: version.Version, Target: target, Clock: clk}
	cfg := cycle.Config{
		Ignore:          append([]string{}, in.Ignore...),
		AllowIncomplete: in.AllowIncomplete,
	}
	switch in.Readiness {
	case "", string(cycle.ReadinessInventory):
		cfg.Readiness = cycle.ReadinessInventory
		cfg.Smoke = append([]string{}, in.Smoke...)
	case string(cycle.ReadinessScenario):
		cfg.Readiness = cycle.ReadinessScenario
		cfg.ReadinessScenarios = append([]string{}, in.Smoke...)
	default:
		return fail(stdout, stderr, in, errclass.Usage{Msg: "unknown --readiness " + in.Readiness})
	}
	var last cycle.Result
	eng := appwatch.Engine(appwatch.Deps{
		Store:  store,
		Differ: diff.Engine{},
		Clock:  clk,
		Smoke:  toolSmoke{emit: emit, clock: clk},
	}, appwatch.Request{Config: cfg}, func(res cycle.Result) {
		last = res
		emit(fswatch.Note{
			Kind:       fswatch.KindDiff,
			State:      string(res.Outcome),
			Outcome:    string(res.Outcome),
			Message:    res.Message,
			Incomplete: res.Outcome == cycle.OutcomeIndeterminate,
		})
	})
	if in.Baseline != "" {
		snap, readErr := store.Read(ctx, in.Baseline)
		if readErr != nil {
			return fail(stdout, stderr, in, readErr)
		}
		eng.SetBaseline(snap)
	}
	handshake := appwatch.Probe(eng, func(pctx context.Context, proc port.Proc) (port.Session, error) {
		return protocol.Connect(pctx, protocol.Stream{Reader: proc.Stdout(), Writer: proc.Stdin()}, protocol.Options{
			Identity:     model.Identity{Name: "mcpdev", Version: version.Version},
			PreferEra:    preferEra(target.Protocol),
			SkipDiscover: target.SkipDiscover,
			Transport:    "stdio",
			Tracer:       startTracer(in, stderr, target.Sensitive),
		})
	})

	ports := testWatchPorts
	if ports == nil {
		ctrl := supervisor.New(supervisor.Options{
			Starter: process.Supervisor{},
			Probe: func(pctx context.Context, proc port.Proc) (port.Session, error) {
				return handshake(pctx, proc)
			},
			Clock:          clk,
			ReplaceInPlace: !keepLast,
		})
		built := portsFrom(ctrl, target, in)
		ports = &built
	}

	sess := &appwatch.Session{
		Source:       source,
		Clock:        clk,
		Matcher:      matcher,
		Ports:        *ports,
		Emit:         emit,
		OnSuccess:    onSuccess,
		Debounce:     debounce,
		FailFast:     in.FailFast,
		KeepLastGood: keepLast,
		ExitOnChange: exitOnChange,
	}
	runErr := sess.Run(ctx)
	if runErr == nil && exitOnChange && last.Ready {
		switch last.Outcome {
		case cycle.OutcomeAdditive, cycle.OutcomeBreaking, cycle.OutcomeMetadata:
			runErr = errclass.InterfaceChange{Err: errors.New(string(last.Outcome))}
		case cycle.OutcomeIndeterminate:
			runErr = errclass.Incomplete{Err: errors.New("watch interface indeterminate")}
		}
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		_ = writeln(stderr, runErr.Error())
		return errclass.ExitCode(runErr)
	}
	return errclass.ExitOK
}

func portsFrom(ctrl *supervisor.Controller, target port.Target, in Invocation) appwatch.Ports {
	return appwatch.Ports{
		Rebuild: func(ctx context.Context, _ []fswatch.Event) appwatch.RebuildReport {
			if len(in.RebuildArgv) == 0 {
				return appwatch.RebuildReport{Class: string(buildrun.ClassOK)}
			}
			res := buildrun.Run(ctx, buildrun.Spec{
				Argv:            append([]string{}, in.RebuildArgv...),
				Dir:             target.Dir,
				Env:             target.Env,
				ArtifactStaging: in.Staging,
				ArtifactFinal:   in.Artifact,
				PreviousGood:    in.Artifact,
			})
			return appwatch.RebuildReport{Err: res.Err, Log: res.Stdout + res.Stderr, Class: string(res.Class), Duration: res.Duration}
		},
		Begin: ctrl.Begin,
		Promote: func(ctx context.Context, gen int64, log string) error {
			_, err := ctrl.Swap(ctx, gen, supervisor.SwapSpec{
				BuildLog: log,
				Proc: port.ProcSpec{
					Env:  target.Env,
					Dir:  target.Dir,
					Argv: append([]string{}, target.Command...),
				},
			})
			return err
		},
		FailBuild: ctrl.FailBuild,
		Shutdown:  ctrl.Close,
	}
}

type toolSmoke struct {
	emit  func(fswatch.Note)
	clock port.Clock
}

func (s toolSmoke) Run(ctx context.Context, sess port.Session, names []string) error {
	for _, name := range names {
		if s.emit != nil {
			n := fswatch.Note{Kind: fswatch.KindScenario, State: "running", Message: name}
			if s.clock != nil {
				n.At = s.clock.Now().UTC()
			}
			s.emit(n)
		}
		if _, err := sess.CallTool(ctx, name, json.RawMessage(`{}`)); err != nil {
			return err
		}
	}
	return nil
}
