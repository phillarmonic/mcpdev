package watch

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/diagnostic"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
	fswatch "github.com/phillarmonic/mcpdev/internal/watch"
)

// OnSuccess is what happens after a classified-ok rebuild.
type OnSuccess string

const (
	OnSuccessReconnect OnSuccess = "reconnect"
	OnSuccessNone      OnSuccess = "none"
)

// RebuildReport is one classified rebuild from the CLI adapter.
type RebuildReport struct {
	Err      error
	Log      string
	Class    string
	Duration time.Duration
}

// Ports are injectable rebuild and generation operations. CLI owns adapters.
type Ports struct {
	Rebuild   func(ctx context.Context, batch []fswatch.Event) RebuildReport
	Begin     func() (int64, error)
	Promote   func(ctx context.Context, gen int64, log string) error
	FailBuild func(gen int64, log, reason string) error
	Shutdown  func(ctx context.Context) error
}

// Session coalesces file events, rebuilds, and reconnects until ctx is done.
type Session struct {
	Ports        Ports
	Source       fswatch.Source
	Clock        port.Clock
	Emit         func(fswatch.Note)
	OnSuccess    OnSuccess
	Matcher      fswatch.Matcher
	Debounce     time.Duration
	seq          atomic.Uint64
	FailFast     bool
	KeepLastGood bool
	ExitOnChange bool
}

// Run watches until cancel, fail-fast, or --exit-on-change.
func (s *Session) Run(ctx context.Context) error {
	clk := s.Clock
	if clk == nil {
		clk = clock.Wall{}
	}
	onSuccess := s.OnSuccess
	if onSuccess == "" {
		onSuccess = OnSuccessReconnect
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindFS, State: "watching", Message: "watch started"})
	if err := s.boot(ctx, clk, onSuccess); err != nil {
		shutErr := s.shutdown(ctx)
		return firstErr(err, shutErr)
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var terminal error
	loop := fswatch.Loop{
		Source:   s.Source,
		Clock:    clk,
		Matcher:  s.Matcher,
		Debounce: s.Debounce,
		Build: func(bctx context.Context, batch []fswatch.Event) error {
			err := s.cycle(bctx, clk, onSuccess, batch)
			if err != nil && (s.FailFast || isChangeExit(err)) {
				terminal = err
				cancel()
			}
			return nil
		},
	}
	runErr := loop.Run(runCtx)
	shutErr := s.shutdown(ctx)
	if terminal != nil {
		return firstErr(terminal, shutErr)
	}
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		return shutErr
	}
	return firstErr(runErr, shutErr)
}

func (s *Session) boot(ctx context.Context, clk port.Clock, onSuccess OnSuccess) error {
	if onSuccess != OnSuccessReconnect || s.Ports.Begin == nil || s.Ports.Promote == nil {
		return nil
	}
	gen, err := s.Ports.Begin()
	if err != nil {
		return err
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindRestart, State: "starting", Generation: gen})
	s.emit(clk, fswatch.Note{Kind: fswatch.KindNegotiate, State: "handshake", Generation: gen})
	if err := s.Ports.Promote(ctx, gen, ""); err != nil {
		s.emit(clk, fswatch.Note{Kind: fswatch.KindRecovery, State: "failed", Generation: gen, Message: errString(err), Incomplete: true})
		if s.FailFast {
			return err
		}
		return nil
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindProbe, State: "ready", Generation: gen})
	return nil
}

func (s *Session) cycle(ctx context.Context, clk port.Clock, onSuccess OnSuccess, batch []fswatch.Event) error {
	path := ""
	if len(batch) > 0 {
		path = batch[0].Path
		s.emit(clk, fswatch.Note{Kind: fswatch.KindFS, State: batch[0].Op.String(), Path: path})
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindBuild, State: "running", Path: path})
	var report RebuildReport
	if s.Ports.Rebuild != nil {
		report = s.Ports.Rebuild(ctx, batch)
	}
	state := "ok"
	incomplete := report.Err != nil
	if incomplete {
		state = "failed"
	}
	s.emit(clk, fswatch.Note{
		Kind:       fswatch.KindBuild,
		State:      state,
		Class:      report.Class,
		Message:    errString(report.Err),
		Path:       path,
		Incomplete: incomplete,
	})
	if report.Err != nil {
		if s.Ports.Begin != nil && s.Ports.FailBuild != nil {
			gen, beginErr := s.Ports.Begin()
			if beginErr == nil {
				_ = s.Ports.FailBuild(gen, report.Log, report.Class)
			}
		}
		s.emit(clk, fswatch.Note{Kind: fswatch.KindRecovery, State: "keep_last_good", Class: report.Class, Message: errString(report.Err), Incomplete: true})
		if s.FailFast {
			return report.Err
		}
		return nil
	}
	if onSuccess != OnSuccessReconnect || s.Ports.Begin == nil || s.Ports.Promote == nil {
		return nil
	}
	gen, err := s.Ports.Begin()
	if err != nil {
		return err
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindRestart, State: "starting", Generation: gen})
	s.emit(clk, fswatch.Note{Kind: fswatch.KindNegotiate, State: "handshake", Generation: gen})
	if err := s.Ports.Promote(ctx, gen, report.Log); err != nil {
		s.emit(clk, fswatch.Note{Kind: fswatch.KindRecovery, State: "failed", Generation: gen, Message: errString(err), Incomplete: true})
		if s.FailFast || isChangeExit(err) {
			return err
		}
		return nil
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindProbe, State: "ready", Generation: gen})
	return nil
}

func (s *Session) shutdown(ctx context.Context) error {
	clk := s.Clock
	if clk == nil {
		clk = clock.Wall{}
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindShutdown, State: "stopping"})
	if s.Ports.Shutdown == nil {
		s.emit(clk, fswatch.Note{Kind: fswatch.KindShutdown, State: "stopped"})
		return nil
	}
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	err := s.Ports.Shutdown(stopCtx)
	state := "stopped"
	if err != nil {
		state = "failed"
	}
	s.emit(clk, fswatch.Note{Kind: fswatch.KindShutdown, State: state, Message: errString(err), Incomplete: err != nil})
	return err
}

func (s *Session) emit(clk port.Clock, n fswatch.Note) {
	if s.Emit == nil {
		return
	}
	n.ID = "e" + strconv.FormatUint(s.seq.Add(1), 10)
	n.At = clk.Now().UTC()
	s.Emit(n)
}

// errString is the note message for err. A diagnostic that carries child stderr
// adds the child's last stderr line, because the note is the only place a watch
// user learns why a generation died.
func errString(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if rec, ok := errors.AsType[diagnostic.Record](err); ok {
		if tail := stderrTail(rec.ChildStderr); tail != "" {
			msg += ": child stderr: " + tail
		}
	}
	return msg
}

// stderrTail is the last non-empty line of child stderr, bounded for a note.
func stderrTail(s string) string {
	const limit = 240
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for _, raw := range slices.Backward(lines) {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if len(line) > limit {
			return line[:limit] + "..."
		}
		return line
	}
	return ""
}

func firstErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

func isChangeExit(err error) bool {
	return errclass.ExitCode(err) == errclass.ExitInterfaceChange
}
