package replay

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// File opens a replay document as a Session. It never dials a live server.
type File struct {
	Clock port.Clock
	Path  string
}

var _ port.Connector = File{}

func (f File) Connect(ctx context.Context, _ port.Target) (port.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, errclass.Connection{Err: err}
	}
	doc, err := Load(f.Path)
	if err != nil {
		return nil, err
	}
	clk := f.Clock
	if clk == nil {
		clk = clock.Wall{}
	}
	if doc.Session.Era == "" {
		return nil, errclass.Usage{Msg: "replay: session.era is required for --replay"}
	}
	return newSession(doc, clk), nil
}
