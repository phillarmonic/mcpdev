package supervisor

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/port"
)

type slot struct {
	ctx         context.Context
	proc        port.Proc
	sess        port.Session
	stop        context.CancelFunc
	gen         Generation
	swapStarted bool
	promoting   bool
}

func (c *Controller) stopSlot(ctx context.Context, sl *slot, reason string) error {
	if sl == nil {
		return nil
	}
	if sl.stop != nil {
		sl.stop()
	}
	stopCtx, cancel := context.WithTimeout(ctx, c.stopTimeout)
	defer cancel()
	var first error
	if sl.sess != nil {
		first = sl.sess.Close(stopCtx)
	}
	if sl.proc != nil {
		if err := sl.proc.Close(stopCtx); first == nil {
			first = err
		}
		sl.gen.Stderr = stderrOf(sl.proc)
	}
	sl.gen.State = StateClosed
	if sl.gen.TerminateReason == "" {
		sl.gen.TerminateReason = reason
	}
	return first
}

func armAbort(swapCtx context.Context, sl *slot) func() {
	done := make(chan struct{})
	go func() {
		// swap abort: owned by Swap; exits when Swap returns or the generation ends.
		select {
		case <-swapCtx.Done():
			sl.stop()
		case <-done:
		case <-sl.ctx.Done():
		}
	}()
	return func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}
}

type stderrSource interface {
	StderrText() string
}

func stderrOf(p port.Proc) string {
	if p == nil {
		return ""
	}
	s, ok := p.(stderrSource)
	if !ok {
		return ""
	}
	return s.StderrText()
}
