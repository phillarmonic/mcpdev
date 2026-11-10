package process

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Proc is a live stdio child. Stdin and Stdout are exclusive JSON-RPC pipes.
type Proc struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     *boundWriter
	clock      port.Clock
	waitDone   chan struct{}
	stderrDone chan struct{}
	closing    chan struct{}
	waitErr    error
	closeErr   error
	grace      time.Duration
	closeOnce  sync.Once
}

var _ port.Proc = (*Proc)(nil)

func (p *Proc) Stdin() io.WriteCloser { return p.stdin }
func (p *Proc) Stdout() io.ReadCloser { return p.stdout }

// PID is the child pid, or -1 if the process has not started.
func (p *Proc) PID() int {
	if p.cmd == nil || p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

// StderrText returns the bounded stderr prefix captured so far.
func (p *Proc) StderrText() string {
	return p.stderr.String()
}

// StderrDropped is the number of stderr bytes discarded after the bound.
func (p *Proc) StderrDropped() int {
	return p.stderr.Dropped()
}

// Wait blocks until the child exits.
func (p *Proc) Wait() error {
	<-p.waitDone
	return p.wrapWait(p.waitErr)
}

// Signal delivers sig to the child process group when possible.
func (p *Proc) Signal(sig os.Signal) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return errclass.Connection{Err: errors.New("process: not started")}
	}
	if sig == os.Kill {
		return killProcess(p.cmd.Process)
	}
	return interruptProcess(p.cmd.Process, sig)
}

// Close closes stdin, waits for a graceful exit, then interrupts and kills.
func (p *Proc) Close(ctx context.Context) error {
	p.closeOnce.Do(func() {
		close(p.closing)
		_ = p.stdin.Close()
		p.closeErr = p.escalate(ctx)
		<-p.stderrDone
	})
	return p.closeErr
}

func (p *Proc) escalate(ctx context.Context) error {
	if err := p.waitWithGrace(ctx); err == nil {
		return p.exitErr()
	} else if errors.Is(err, ctx.Err()) && ctx.Err() != nil {
		_ = p.kill()
		<-p.waitDone
		<-p.stderrDone
		return connectionErr("process_shutdown", ctx.Err(), p.StderrText())
	}
	_ = p.Signal(os.Interrupt)
	if err := p.waitWithGrace(ctx); err == nil {
		return p.exitErr()
	} else if errors.Is(err, ctx.Err()) && ctx.Err() != nil {
		_ = p.kill()
		<-p.waitDone
		<-p.stderrDone
		return connectionErr("process_shutdown", ctx.Err(), p.StderrText())
	}
	if err := p.kill(); err != nil && !alreadyDead(err) {
		<-p.waitDone
		<-p.stderrDone
		return connectionErr("process_kill", err, p.StderrText())
	}
	select {
	case <-p.waitDone:
		return p.exitErr()
	case <-ctx.Done():
		<-p.waitDone
		<-p.stderrDone
		return connectionErr("process_shutdown", ctx.Err(), p.StderrText())
	}
}

// exitErr joins the stderr copy before classifying an exit so the diagnostic
// carries the child's last words instead of an empty capture.
func (p *Proc) exitErr() error {
	<-p.stderrDone
	return p.wrapWait(p.waitErr)
}

func (p *Proc) waitWithGrace(ctx context.Context) error {
	select {
	case <-p.waitDone:
		return nil
	case <-p.clock.After(p.grace):
		return errclass.Connection{Err: errors.New("process: shutdown grace elapsed")}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Proc) kill() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return killProcess(p.cmd.Process)
}

func (p *Proc) wrapWait(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		return connectionErr("process_exit", err, p.StderrText())
	}
	return connectionErr("process_wait", err, p.StderrText())
}

func alreadyDead(err error) bool {
	return err == nil || errors.Is(err, os.ErrProcessDone)
}
