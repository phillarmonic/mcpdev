package protocol

import (
	"context"
	"errors"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	var coded interface{ ExitClass() int }
	if errors.As(err, &coded) {
		return err
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errclass.Connection{Err: err}
	case errors.Is(err, mcp.ErrConnectionClosed), errors.Is(err, io.EOF), errors.Is(err, io.ErrClosedPipe):
		return errclass.Connection{Err: err}
	default:
		return errclass.Protocol{Err: err}
	}
}
