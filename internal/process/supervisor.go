package process

import (
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/diagnostic"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

const defaultGrace = 2 * time.Second

// Supervisor implements port.ProcessSupervisor with os/exec.
type Supervisor struct {
	Clock    port.Clock
	LookPath func(string) (string, error)
}

var _ port.ProcessSupervisor = Supervisor{}

// Start launches argv as a process group without a shell.
func (s Supervisor) Start(ctx context.Context, spec port.ProcSpec) (port.Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, connectionErr("process_start", err, "")
	}
	lookPath := s.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	argv, err := resolveArgv(spec, lookPath)
	if err != nil {
		return nil, errclass.Usage{Err: err, Msg: "process"}
	}
	clk := spec.Clock
	if clk == nil {
		clk = s.Clock
	}
	if clk == nil {
		clk = clock.Wall{}
	}
	grace := spec.GracePeriod
	if grace <= 0 {
		grace = defaultGrace
	}
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // G204: argv is operator configuration, never shell-joined
	cmd.Dir = spec.Dir
	cmd.Env = mergeEnv(spec)
	prepareSys(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, connectionErr("process_start", err, "")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, connectionErr("process_start", err, "")
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, connectionErr("process_start", err, "")
	}
	stderr := newBoundWriter(spec.StderrLimit)
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderrPipe.Close()
		return nil, connectionErr("process_start", err, "")
	}
	p := &Proc{
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		clock:      clk,
		grace:      grace,
		waitDone:   make(chan struct{}),
		stderrDone: make(chan struct{}),
		closing:    make(chan struct{}),
	}
	go func() {
		// stderr copy: owned by Proc; joined in Close.
		_, _ = io.Copy(stderr, stderrPipe)
		_ = stderrPipe.Close()
		close(p.stderrDone)
	}()
	go func() {
		// waitpid: owned by Proc; joined in Close.
		p.waitErr = cmd.Wait()
		close(p.waitDone)
	}()
	go func() {
		// start-context cancel: owned by Proc; Close is idempotent.
		select {
		case <-ctx.Done():
			_ = p.Close(context.WithoutCancel(ctx))
		case <-p.closing:
		}
	}()
	return p, nil
}

func connectionErr(operation string, err error, stderr string) error {
	rec := diagnostic.FromError(errclass.Connection{Err: err}, diagnostic.Record{
		Operation: operation,
		Transport: "stdio",
	})
	return diagnostic.AttachChildStderr(rec, stderr)
}
