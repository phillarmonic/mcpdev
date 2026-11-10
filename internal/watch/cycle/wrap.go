package cycle

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Wrap runs handshake then Evaluate. Handshake or readiness failure keeps the baseline.
func (e *Engine) Wrap(handshake Handshake) Handshake {
	return func(ctx context.Context, proc port.Proc) (port.Session, error) {
		start := e.now()
		if handshake == nil {
			_, err := e.fail(start, "handshake", errclass.Usage{Msg: "watch cycle: handshake required"})
			return nil, err
		}
		sess, err := handshake(ctx, proc)
		if err != nil {
			_, _ = e.fail(start, "protocol", err)
			return nil, err
		}
		if _, evalErr := e.Evaluate(ctx, sess); evalErr != nil {
			_ = sess.Close(context.WithoutCancel(ctx))
			return nil, evalErr
		}
		return sess, nil
	}
}
