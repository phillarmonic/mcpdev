package supervisor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

const (
	defaultProbeTimeout = 10 * time.Second
	defaultStopTimeout  = 2 * time.Second
)

// ProbeFunc turns a live stdio child into a protocol session.
type ProbeFunc func(ctx context.Context, proc port.Proc) (port.Session, error)

// Options construct a Controller.
type Options struct {
	Starter        port.ProcessSupervisor
	Probe          ProbeFunc
	Clock          port.Clock
	ReplaceInPlace bool
	ProbeTimeout   time.Duration
	StopTimeout    time.Duration
}

// SwapSpec starts one generation after a successful rebuild.
type SwapSpec struct {
	BuildLog string
	Proc     port.ProcSpec
}

// Controller owns generation-based stdio restart.
type Controller struct {
	starter        port.ProcessSupervisor
	clock          port.Clock
	cond           *sync.Cond
	probe          ProbeFunc
	current        *slot
	incoming       *slot
	lastFailed     *Generation
	state          State
	probeTimeout   time.Duration
	stopTimeout    time.Duration
	seq            int64
	mu             sync.Mutex
	replaceInPlace bool
}

// New returns an idle controller. Starter and Probe are required for Swap.
func New(opts Options) *Controller {
	clk := opts.Clock
	if clk == nil {
		clk = clock.Wall{}
	}
	probeTO := opts.ProbeTimeout
	if probeTO <= 0 {
		probeTO = defaultProbeTimeout
	}
	stopTO := opts.StopTimeout
	if stopTO <= 0 {
		stopTO = defaultStopTimeout
	}
	c := &Controller{
		starter:        opts.Starter,
		probe:          opts.Probe,
		clock:          clk,
		replaceInPlace: opts.ReplaceInPlace,
		probeTimeout:   probeTO,
		stopTimeout:    stopTO,
		state:          StateIdle,
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Begin reserves a monotonic generation id and cancels an unpromoted attempt.
func (c *Controller) Begin() (int64, error) {
	c.mu.Lock()
	if c.state == StateClosed {
		c.mu.Unlock()
		return 0, closedErr()
	}
	for c.incoming != nil && c.incoming.promoting {
		c.cond.Wait()
		if c.state == StateClosed {
			c.mu.Unlock()
			return 0, closedErr()
		}
	}
	old := c.takeIncomingLocked("superseded")
	c.seq++
	life, cancel := context.WithCancel(context.Background())
	c.incoming = &slot{
		gen: Generation{
			ID:    c.seq,
			State: StateBuilding,
		},
		ctx:  life,
		stop: cancel,
	}
	c.state = StateBuilding
	id := c.seq
	c.mu.Unlock()
	_ = c.stopSlot(context.Background(), old, "superseded")
	return id, nil
}

// FailBuild records a failed rebuild for gen without starting a process.
func (c *Controller) FailBuild(gen int64, log, reason string) error {
	c.mu.Lock()
	if c.incoming == nil || c.incoming.gen.ID != gen {
		c.mu.Unlock()
		return stale(gen)
	}
	if reason == "" {
		reason = "build_failed"
	}
	c.incoming.gen.BuildLog = log
	c.incoming.gen.TerminateReason = reason
	c.incoming.gen.State = StateFailed
	snap := c.incoming.gen
	c.lastFailed = &snap
	sl := c.takeIncomingLocked(reason)
	c.restoreStateLocked()
	c.mu.Unlock()
	_ = c.stopSlot(context.Background(), sl, reason)
	return nil
}

// Swap starts gen, probes it, then promotes it as the only current generation.
func (c *Controller) Swap(ctx context.Context, gen int64, spec SwapSpec) (*Generation, error) {
	if err := ctx.Err(); err != nil {
		return nil, errclass.Connection{Err: err}
	}
	sl, err := c.markStarting(gen, spec.BuildLog)
	if err != nil {
		return nil, err
	}
	releaseAbort := armAbort(ctx, sl)
	defer releaseAbort()

	if c.replaceInPlace {
		if retireErr := c.retireCurrent(ctx); retireErr != nil {
			return c.failSwap(ctx, sl, retireErr)
		}
	}
	proc, err := c.startProc(ctx, sl, spec.Proc)
	if err != nil {
		return c.failSwap(ctx, sl, err)
	}
	sess, err := c.runProbe(ctx, sl, proc)
	if err != nil {
		return c.failSwap(ctx, sl, err)
	}
	return c.promote(ctx, sl, proc, sess)
}

// Close stops incoming and current generations and refuses further work.
func (c *Controller) Close(ctx context.Context) error {
	c.mu.Lock()
	if c.state == StateClosed {
		c.mu.Unlock()
		return nil
	}
	for c.incoming != nil && c.incoming.promoting {
		c.cond.Wait()
		if c.state == StateClosed {
			c.mu.Unlock()
			return nil
		}
	}
	c.state = StateStopping
	incoming := c.takeIncomingLocked("controller_closed")
	current := c.current
	c.current = nil
	c.mu.Unlock()

	var first error
	if err := c.stopSlot(ctx, incoming, "controller_closed"); err != nil {
		first = err
	}
	if err := c.stopSlot(ctx, current, "controller_closed"); err != nil && first == nil {
		first = err
	}

	c.mu.Lock()
	c.state = StateClosed
	c.cond.Broadcast()
	c.mu.Unlock()
	return first
}

// Snapshot copies controller state.
func (c *Controller) Snapshot() View {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := View{State: c.state}
	if c.current != nil {
		g := c.current.gen
		v.Current = &g
	}
	if c.incoming != nil {
		g := c.incoming.gen
		v.Incoming = &g
	}
	if c.lastFailed != nil {
		g := *c.lastFailed
		v.LastFailed = &g
	}
	return v
}

func (c *Controller) markStarting(gen int64, log string) (*slot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == StateClosed {
		return nil, closedErr()
	}
	if c.incoming == nil || c.incoming.gen.ID != gen {
		return nil, stale(gen)
	}
	if c.incoming.swapStarted {
		return nil, errclass.Usage{Err: errors.New("supervisor: swap already started"), Msg: "supervisor"}
	}
	if c.starter == nil || c.probe == nil {
		return nil, errclass.Usage{Err: errors.New("supervisor: starter and probe required"), Msg: "supervisor"}
	}
	c.incoming.swapStarted = true
	c.incoming.gen.BuildLog = log
	c.incoming.gen.State = StateStarting
	c.state = StateStarting
	return c.incoming, nil
}

func (c *Controller) startProc(ctx context.Context, sl *slot, spec port.ProcSpec) (port.Proc, error) {
	if spec.Clock == nil {
		spec.Clock = c.clock
	}
	// process.Start closes the child when startCtx ends; that lifetime is the generation.
	proc, err := c.starter.Start(sl.ctx, spec) //nolint:contextcheck // generation life, not Swap
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.incoming != sl {
		c.mu.Unlock()
		_ = proc.Close(context.WithoutCancel(ctx))
		c.mu.Lock()
		return nil, stale(sl.gen.ID)
	}
	sl.proc = proc
	sl.gen.Stderr = stderrOf(proc)
	return proc, nil
}

func (c *Controller) runProbe(ctx context.Context, sl *slot, proc port.Proc) (port.Session, error) {
	probeCtx, cancel := context.WithTimeout(ctx, c.probeTimeout)
	defer cancel()
	go func() {
		// generation cancel: owned by runProbe; exits when probeCtx or life ends.
		select {
		case <-sl.ctx.Done():
			cancel()
		case <-probeCtx.Done():
		}
	}()
	sess, err := c.probe(probeCtx, proc)
	sl.gen.Stderr = stderrOf(proc)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.incoming != sl {
		c.mu.Unlock()
		_ = sess.Close(context.WithoutCancel(ctx))
		_ = proc.Close(context.WithoutCancel(ctx))
		c.mu.Lock()
		return nil, stale(sl.gen.ID)
	}
	sl.sess = sess
	sl.gen.Protocol = sess.Info()
	sl.gen.HasProtocol = true
	return sess, nil
}

func (c *Controller) promote(ctx context.Context, sl *slot, proc port.Proc, sess port.Session) (*Generation, error) {
	c.mu.Lock()
	if c.incoming != sl || c.state == StateClosed {
		c.mu.Unlock()
		_ = sess.Close(context.WithoutCancel(ctx))
		_ = proc.Close(context.WithoutCancel(ctx))
		return nil, stale(sl.gen.ID)
	}
	sl.promoting = true
	old := c.current
	c.current = nil
	c.state = StateStopping
	c.mu.Unlock()

	_ = c.stopSlot(ctx, old, "replaced")

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == StateClosed {
		return nil, closedErr()
	}
	sl.promoting = false
	sl.gen.State = StateReady
	c.current = sl
	c.incoming = nil
	c.state = StateReady
	c.cond.Broadcast()
	out := sl.gen
	return &out, nil
}

func (c *Controller) failSwap(ctx context.Context, sl *slot, cause error) (*Generation, error) {
	reason := "start_failed"
	if cause != nil {
		reason = cause.Error()
	}
	_ = c.stopSlot(ctx, sl, reason)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == StateClosed {
		return nil, cause
	}
	if c.incoming == sl {
		sl.gen.State = StateFailed
		sl.gen.TerminateReason = reason
		snap := sl.gen
		c.lastFailed = &snap
		c.incoming = nil
		sl.stop()
	}
	c.restoreStateLocked()
	c.cond.Broadcast()
	return c.lastFailed, cause
}

func (c *Controller) retireCurrent(ctx context.Context) error {
	c.mu.Lock()
	old := c.current
	c.current = nil
	if old != nil {
		c.state = StateStopping
	}
	c.mu.Unlock()
	if old == nil {
		return nil
	}
	return c.stopSlot(ctx, old, "replaced_in_place")
}

func (c *Controller) takeIncomingLocked(reason string) *slot {
	sl := c.incoming
	c.incoming = nil
	if sl == nil {
		return nil
	}
	if sl.gen.TerminateReason == "" {
		sl.gen.TerminateReason = reason
	}
	sl.stop()
	return sl
}

func (c *Controller) restoreStateLocked() {
	switch {
	case c.current != nil:
		c.state = StateReady
	case c.lastFailed != nil:
		c.state = StateFailed
	default:
		c.state = StateIdle
	}
}

func closedErr() error {
	return errclass.Connection{Err: errors.New("supervisor: closed")}
}
