package port

import (
	"context"
	"io"
	"os"
	"time"
)

// ProcSpec launches a stdio child. Argv is never passed through a shell.
type ProcSpec struct {
	Clock       Clock
	Env         map[string]string
	Dir         string
	Argv        []string
	StderrLimit int
	GracePeriod time.Duration
	ClearEnv    bool
}

// ProcessSupervisor starts a child process for the stdio transport.
type ProcessSupervisor interface {
	Start(ctx context.Context, spec ProcSpec) (Proc, error)
}

// Proc is a live child. Stdin and Stdout are the MCP JSON-RPC pipes.
type Proc interface {
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Wait() error
	Signal(os.Signal) error
	Close(ctx context.Context) error
}
