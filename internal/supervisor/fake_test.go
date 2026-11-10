package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

type lab struct {
	live map[int]*fakeProc
	sess map[int]*fakeSession
	next int
	mu   sync.Mutex
}

func newLab() *lab {
	return &lab{live: make(map[int]*fakeProc), sess: make(map[int]*fakeSession)}
}

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

func (l *lab) spawn(stderr string) *fakeProc {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.next++
	id := l.next
	r, w := io.Pipe()
	p := &fakeProc{
		id:     id,
		lab:    l,
		stderr: stderr,
		out:    r,
		in:     w,
		done:   make(chan struct{}),
	}
	l.live[id] = p
	return p
}

func (l *lab) rememberSession(id int, s *fakeSession) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sess[id] = s
}

type fakeProc struct {
	lab    *lab
	out    *io.PipeReader
	in     *io.PipeWriter
	done   chan struct{}
	stderr string
	id     int
	once   sync.Once
	closed atomic.Bool
}

func (p *fakeProc) Stdin() io.WriteCloser { return p.in }
func (p *fakeProc) Stdout() io.ReadCloser { return p.out }
func (p *fakeProc) PID() int              { return p.id }
func (p *fakeProc) StderrText() string    { return p.stderr }
func (p *fakeProc) Wait() error {
	<-p.done
	return nil
}
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

type fakeStarter struct {
	fail        error
	hangStart   <-chan struct{}
	liveAtStart *atomic.Int32
	lab         *lab
	stderr      string
}

func (s *fakeStarter) Start(ctx context.Context, _ port.ProcSpec) (port.Proc, error) {
	if s.hangStart != nil {
		select {
		case <-s.hangStart:
		case <-ctx.Done():
			return nil, errclass.Connection{Err: ctx.Err()}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, errclass.Connection{Err: err}
	}
	if s.fail != nil {
		return nil, s.fail
	}
	if s.liveAtStart != nil {
		s.liveAtStart.Store(int32(s.lab.liveCount()))
	}
	return s.lab.spawn(s.stderr), nil
}

type fakeSession struct {
	lab    *lab
	info   model.SessionInfo
	id     int
	once   sync.Once
	closed atomic.Bool
}

func (s *fakeSession) Era() model.ProtocolEra             { return s.info.Era }
func (s *fakeSession) DiscoverState() model.DiscoverState { return s.info.DiscoverState }
func (s *fakeSession) Info() model.SessionInfo            { return s.info }
func (s *fakeSession) Inventory() port.InventoryReader    { return unusedInv{} }
func (s *fakeSession) CallTool(context.Context, string, json.RawMessage) (model.ToolResult, error) {
	return model.ToolResult{}, unused()
}

func (s *fakeSession) GetPrompt(context.Context, string, json.RawMessage) (model.PromptResult, error) {
	return model.PromptResult{}, unused()
}

func (s *fakeSession) ReadResource(context.Context, string) (model.ResourceBody, error) {
	return model.ResourceBody{}, unused()
}

func (s *fakeSession) Close(context.Context) error {
	s.once.Do(func() { s.closed.Store(true) })
	return nil
}

type unusedInv struct{}

func (unusedInv) List(context.Context, model.PrimitiveKind, model.Page) (model.CatalogPage, error) {
	return model.CatalogPage{}, unused()
}

func unused() error {
	return errclass.Internal{Err: errors.New("supervisor test: unused")}
}

func probeOK(l *lab) ProbeFunc {
	return func(_ context.Context, proc port.Proc) (port.Session, error) {
		fp := proc.(*fakeProc)
		s := &fakeSession{
			id:  fp.id,
			lab: l,
			info: model.SessionInfo{
				Era:             model.Era20251125,
				ProtocolVersion: "2025-11-25",
				ServerName:      "lab",
				Handshake:       model.HandshakeInitialize,
			},
		}
		l.rememberSession(fp.id, s)
		return s, nil
	}
}
