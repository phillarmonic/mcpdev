package stdio

import (
	"context"
	"io"

	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/process"
	"github.com/phillarmonic/mcpdev/internal/protocol"
)

// Conn is a stdio child with exclusive JSON-RPC pipes.
type Conn struct {
	proc   port.Proc
	stdout io.ReadCloser
}

// Open starts a new child for this invocation. Callers pass a newly built
// binary path without restarting mcpdev.
func Open(ctx context.Context, spec port.ProcSpec) (*Conn, error) {
	return OpenWith(ctx, process.Supervisor{}, spec)
}

// OpenWith starts a child using sup.
func OpenWith(ctx context.Context, sup port.ProcessSupervisor, spec port.ProcSpec) (*Conn, error) {
	p, err := sup.Start(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &Conn{proc: p, stdout: &firstJSON{r: p.Stdout()}}, nil
}

// Stream is stdin/stdout for protocol.Connect. stderr is not included.
func (c *Conn) Stream() protocol.Stream {
	return protocol.Stream{Reader: c.stdout, Writer: c.proc.Stdin()}
}

// Proc returns the supervised child.
func (c *Conn) Proc() port.Proc {
	return c.proc
}

// StderrText is the bounded child stderr prefix when the proc is a process.Proc.
func (c *Conn) StderrText() string {
	if p, ok := c.proc.(*process.Proc); ok {
		return p.StderrText()
	}
	return ""
}

// Close escalates shutdown of the child and joins copy/wait goroutines.
func (c *Conn) Close(ctx context.Context) error {
	return c.proc.Close(ctx)
}
