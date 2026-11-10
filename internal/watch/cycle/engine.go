package cycle

import (
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Options construct an Engine.
type Options struct {
	Store    port.SnapshotStore
	Differ   port.Differ
	Clock    port.Clock
	Smoke    SmokeRunner
	OnReport func(Result) // new outcomes only; identical consecutive fingerprints are skipped
	Config   Config
}

// Engine holds the last ready snapshot and a bounded generation history.
type Engine struct {
	store           port.SnapshotStore
	differ          port.Differ
	clock           port.Clock
	smoke           SmokeRunner
	onReport        func(Result)
	lastFingerprint string
	hist            []Record
	cfg             Config
	lastReady       model.Snapshot
	mu              sync.Mutex
	hasReady        bool
}

// New returns an engine with no comparison baseline.
func New(opts Options) *Engine {
	clk := opts.Clock
	if clk == nil {
		clk = clock.Wall{}
	}
	return &Engine{
		store:    opts.Store,
		differ:   opts.Differ,
		clock:    clk,
		smoke:    opts.Smoke,
		onReport: opts.OnReport,
		cfg:      opts.Config.resolved(),
		hist:     []Record{},
	}
}

// History copies the bounded in-memory log, oldest first.
func (e *Engine) History() []Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Record, len(e.hist))
	copy(out, e.hist)
	return out
}

// HasBaseline reports whether a ready generation has supplied a comparison snapshot.
func (e *Engine) HasBaseline() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.hasReady
}

func (e *Engine) now() time.Time {
	return e.clock.Now()
}

func (e *Engine) commitReady(res Result, fp string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastReady = res.Snapshot
	e.hasReady = true
	e.appendLocked(Record{
		At:       e.now(),
		Duration: res.Duration,
		Outcome:  res.Outcome,
		Message:  res.Message,
		Ready:    true,
	})
	e.lastFingerprint = fp
}

func (e *Engine) commitFail(reason string, duration time.Duration, outcome Outcome, msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.appendLocked(Record{
		At:       e.now(),
		Duration: duration,
		Outcome:  outcome,
		Message:  msg,
		Reason:   reason,
		Ready:    false,
	})
}

func (e *Engine) appendLocked(rec Record) {
	e.hist = append(e.hist, rec)
	if len(e.hist) > e.cfg.HistoryBound {
		e.hist = e.hist[len(e.hist)-e.cfg.HistoryBound:]
	}
}

func (e *Engine) snapshotLocked() (model.Snapshot, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastReady, e.hasReady
}
