// Package fake is an in-process custom connector for registry tests.
// It does not import internal/protocol or the official Go SDK.
package fake

import (
	"context"
	"io"
	"maps"
	"sync/atomic"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/extension"
)

// Connector is a named Adapter that opens an in-memory JSON-RPC pipe pair.
type Connector struct {
	Sem   extension.Semantics
	Label string
}

// Name is the registry key.
func (c Connector) Name() string {
	if c.Label == "" {
		return "fake"
	}
	return c.Label
}

// Semantics implements extension.Adapter.
func (c Connector) Semantics() extension.Semantics { return c.Sem }

// Open constructs a pipe pair. Close shuts both sides. Diagnostics never include secrets.
func (c Connector) Open(ctx context.Context, cfg extension.Config) (extension.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, errclass.Connection{Err: ctx.Err()}
	default:
	}
	rClient, wServer := io.Pipe()
	rServer, wClient := io.Pipe()
	conn := &conn{
		reader: rClient,
		writer: wClient,
		drop: func() {
			_ = rServer.Close()
			_ = wServer.Close()
		},
		diag: map[string]string{
			"connector": c.Name(),
			"standard":  "false",
			"endpoint":  cfg.Endpoint,
		},
	}
	return conn, nil
}

type conn struct {
	reader io.ReadCloser
	writer io.WriteCloser
	drop   func()
	diag   map[string]string
	closed atomic.Bool
}

func (c *conn) Reader() io.ReadCloser  { return c.reader }
func (c *conn) Writer() io.WriteCloser { return c.writer }

func (c *conn) Diagnostics() map[string]string {
	out := make(map[string]string, len(c.diag)+1)
	maps.Copy(out, c.diag)
	out["closed"] = formatClosed(c.closed.Load())
	return out
}

func (c *conn) Close(ctx context.Context) error {
	_ = ctx
	if c.closed.Swap(true) {
		return nil
	}
	err1 := c.reader.Close()
	err2 := c.writer.Close()
	if c.drop != nil {
		c.drop()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func formatClosed(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
