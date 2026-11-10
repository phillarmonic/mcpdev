package watch_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	appwatch "github.com/phillarmonic/mcpdev/internal/app/watch"
	"github.com/phillarmonic/mcpdev/internal/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/supervisor"
	"github.com/phillarmonic/mcpdev/internal/watch/cycle"
)

type seqStore struct {
	snaps []model.Snapshot
	n     atomic.Int32
}

func (s *seqStore) Capture(context.Context, port.Session, []model.PrimitiveKind) (model.Snapshot, error) {
	i := int(s.n.Add(1) - 1)
	if i >= len(s.snaps) {
		return model.Snapshot{}, errors.New("no snapshot")
	}
	return s.snaps[i], nil
}
func (s *seqStore) Write(context.Context, string, model.Snapshot) error { return nil }
func (s *seqStore) Read(context.Context, string) (model.Snapshot, error) {
	return model.Snapshot{}, errors.New("unused")
}

type smokeGate struct {
	fail atomic.Bool
}

func (s *smokeGate) Run(context.Context, port.Session, []string) error {
	if s.fail.Load() {
		return errclass.Assertion{Err: errors.New("named smoke failed")}
	}
	return nil
}

func toolSnap(names ...string) model.Snapshot {
	items := make([]model.Tool, 0, len(names))
	for _, n := range names {
		items = append(items, model.Tool{Name: n, InputSchema: json.RawMessage(`{"type":"object"}`)})
	}
	return model.Snapshot{
		SchemaVersion: 1,
		Tools: model.CatalogSection{
			Kind: model.KindTools, Status: model.SectionComplete, Tools: items, ItemCount: len(items),
		},
	}
}

func TestSupervisorPromotesOnlyReadyGenerations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		breaking := toolSnap("echo")
		breaking.Tools.Tools[0].InputSchema = json.RawMessage(`{"type":"object","required":["q"]}`)
		recovery := toolSnap("echo", "sum")
		recovery.Tools.Tools[0].InputSchema = breaking.Tools.Tools[0].InputSchema
		store := &seqStore{snaps: []model.Snapshot{
			toolSnap("echo"),
			toolSnap("echo"),
			breaking,
			breaking,
			recovery,
		}}
		smoke := &smokeGate{}
		var reports []cycle.Result
		eng := appwatch.Engine(appwatch.Deps{Store: store, Differ: diff.Engine{}, Smoke: smoke}, appwatch.Request{
			Config: cycle.Config{Smoke: []string{"ci"}, HistoryBound: 8},
		}, func(r cycle.Result) { reports = append(reports, r) })
		hs := func(ctx context.Context, proc port.Proc) (port.Session, error) {
			return probeOK(l)(ctx, proc)
		}
		c := supervisor.New(supervisor.Options{Starter: &fakeStarter{lab: l}, Probe: supervisor.ProbeFunc(appwatch.Probe(eng, hs))})
		t.Cleanup(func() { _ = c.Close(context.Background()) })

		first := mustSwap(t, c)
		if first.ID != 1 || c.Snapshot().Current.ID != 1 {
			t.Fatalf("first %+v", first)
		}

		second := mustSwap(t, c)
		if second.ID != 2 || len(reports) != 1 {
			t.Fatalf("unchanged flood reports=%d gen=%d", len(reports), second.ID)
		}

		third := mustSwap(t, c)
		if third.ID != 3 || reports[len(reports)-1].Outcome != cycle.OutcomeBreaking {
			t.Fatalf("breaking %+v reports %+v", third, reports)
		}

		smoke.fail.Store(true)
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Swap(context.Background(), g, supervisor.SwapSpec{})
		if errclass.ExitCode(err) != errclass.ExitAssertion {
			t.Fatalf("smoke exit %d err=%v", errclass.ExitCode(err), err)
		}
		if c.Snapshot().Current.ID != 3 || l.liveCount() != 1 {
			t.Fatalf("scenario failure replaced current %+v live %d", c.Snapshot().Current, l.liveCount())
		}
		if !eng.HasBaseline() {
			t.Fatal("baseline lost")
		}

		hsFail := func(context.Context, port.Proc) (port.Session, error) {
			return nil, errclass.Protocol{Err: errors.New("initialize rejected")}
		}
		c2 := supervisor.New(supervisor.Options{Starter: &fakeStarter{lab: l}, Probe: supervisor.ProbeFunc(appwatch.Probe(eng, hsFail))})
		t.Cleanup(func() { _ = c2.Close(context.Background()) })
		g2, err := c2.Begin()
		if err != nil {
			t.Fatal(err)
		}
		_, err = c2.Swap(context.Background(), g2, supervisor.SwapSpec{})
		if errclass.ExitCode(err) != errclass.ExitProtocol {
			t.Fatalf("protocol exit %d", errclass.ExitCode(err))
		}
		if !eng.HasBaseline() {
			t.Fatal("protocol failure cleared baseline")
		}

		smoke.fail.Store(false)
		c3 := supervisor.New(supervisor.Options{Starter: &fakeStarter{lab: l}, Probe: supervisor.ProbeFunc(appwatch.Probe(eng, hs))})
		t.Cleanup(func() { _ = c3.Close(context.Background()) })
		rec := mustSwap(t, c3)
		if rec.ID != 1 {
			t.Fatalf("new controller gen %d", rec.ID)
		}
		h := eng.History()
		if !h[len(h)-1].Ready || h[len(h)-1].Outcome != cycle.OutcomeAdditive {
			t.Fatalf("recovery history %+v", h[len(h)-1])
		}
	})
}

func mustSwap(t *testing.T, c *supervisor.Controller) *supervisor.Generation {
	t.Helper()
	g, err := c.Begin()
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Swap(context.Background(), g, supervisor.SwapSpec{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// lab/process fakes mirror internal/supervisor tests so Probe can run Swap.

type lab struct {
	live map[int]*fakeProc
	next int
	mu   sync.Mutex
}

func newLab() *lab { return &lab{live: map[int]*fakeProc{}} }

func (l *lab) liveCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, p := range l.live {
		if !p.closed.Load() {
			n++
		}
	}
	return n
}

func (l *lab) spawn() *fakeProc {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.next++
	id := l.next
	r, w := io.Pipe()
	p := &fakeProc{id: id, lab: l, out: r, in: w, done: make(chan struct{})}
	l.live[id] = p
	return p
}

type fakeProc struct {
	lab    *lab
	out    *io.PipeReader
	in     *io.PipeWriter
	done   chan struct{}
	id     int
	once   sync.Once
	closed atomic.Bool
}

func (p *fakeProc) Stdin() io.WriteCloser  { return p.in }
func (p *fakeProc) Stdout() io.ReadCloser  { return p.out }
func (p *fakeProc) PID() int               { return p.id }
func (p *fakeProc) StderrText() string     { return "" }
func (p *fakeProc) Wait() error            { <-p.done; return nil }
func (p *fakeProc) Signal(os.Signal) error { return nil }
func (p *fakeProc) Close(context.Context) error {
	p.once.Do(func() {
		p.closed.Store(true)
		close(p.done)
		_ = p.in.Close()
		_ = p.out.Close()
	})
	return nil
}

type fakeStarter struct{ lab *lab }

func (s *fakeStarter) Start(ctx context.Context, _ port.ProcSpec) (port.Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, errclass.Connection{Err: err}
	}
	return s.lab.spawn(), nil
}

type fakeSession struct {
	closed atomic.Bool
}

func (s *fakeSession) Era() model.ProtocolEra             { return model.Era20251125 }
func (s *fakeSession) DiscoverState() model.DiscoverState { return model.DiscoverNotApplicable }
func (s *fakeSession) Info() model.SessionInfo {
	return model.SessionInfo{Era: model.Era20251125, ProtocolVersion: "2025-11-25", ServerName: "lab", Handshake: model.HandshakeInitialize}
}

func (s *fakeSession) Inventory() port.InventoryReader { return s }

func (s *fakeSession) CallTool(context.Context, string, json.RawMessage) (model.ToolResult, error) {
	return model.ToolResult{}, errors.New("unused")
}

func (s *fakeSession) GetPrompt(context.Context, string, json.RawMessage) (model.PromptResult, error) {
	return model.PromptResult{}, errors.New("unused")
}

func (s *fakeSession) ReadResource(context.Context, string) (model.ResourceBody, error) {
	return model.ResourceBody{}, errors.New("unused")
}

func (s *fakeSession) Close(context.Context) error { s.closed.Store(true); return nil }

func (s *fakeSession) List(context.Context, model.PrimitiveKind, model.Page) (model.CatalogPage, error) {
	return model.CatalogPage{}, nil
}

func probeOK(l *lab) supervisor.ProbeFunc {
	return func(_ context.Context, proc port.Proc) (port.Session, error) {
		_ = l
		_ = proc
		return &fakeSession{}, nil
	}
}
