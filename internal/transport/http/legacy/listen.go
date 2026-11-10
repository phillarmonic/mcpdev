package legacy

import (
	"context"
	"io"
	"net/http"

	"github.com/phillarmonic/mcpdev/internal/trace"
)

// Listen follows the GET SSE stream and reconnects with Last-Event-ID.
// Reconnect is only for this 2025-11-25 sessionful era. emit runs in this
// goroutine; Listen returns when ctx or Close cancels, or retries are exhausted.
func (c *Conn) Listen(ctx context.Context, emit func(Event) error) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errclassClosed()
	}
	if c.listenDone == nil {
		c.listenDone = make(chan struct{})
	}
	done := c.listenDone
	c.mu.Unlock()
	defer close(done)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.mu.Lock()
	c.listenCancel = cancel
	c.mu.Unlock()

	limit := c.cfg.maxReconnects()
	retries := 0
	for {
		if err := ctx.Err(); err != nil {
			return wrapRoundTrip(err)
		}
		err := c.getStream(ctx, emit)
		if ctx.Err() != nil {
			return wrapRoundTrip(ctx.Err())
		}
		if retries >= limit {
			return err
		}
		retries++
		c.tracer.Retry("http", "sse_reconnect", retries)
		if d := c.cfg.ReconnectDelay; d > 0 {
			select {
			case <-ctx.Done():
				return wrapRoundTrip(ctx.Err())
			case <-c.cfg.clock().After(d):
			}
		}
	}
}

func (c *Conn) getStream(ctx context.Context, emit func(Event) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return wrapRoundTrip(err)
	}
	c.applyHeaders(req, false)
	c.tracer.HTTPRequest(req.Method, "http", trace.FlattenHeader(req.Header), nil)
	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			c.tracer.Cancel("http", ctx.Err().Error())
		}
		return wrapRoundTrip(err)
	}
	defer func() { _ = resp.Body.Close() }()
	ct := resp.Header.Get("Content-Type")
	c.tracer.HTTPResponse(resp.StatusCode, ct, nil, true)
	if err := classifyStatus(resp.StatusCode, nil, ct); err != nil {
		return err
	}
	return readSSE(resp.Body, func(ev Event) error {
		if ev.ID != "" {
			c.mu.Lock()
			c.lastEventID = ev.ID
			c.mu.Unlock()
		}
		c.tracer.SSE(ev.Name, ev.ID, ev.Data)
		return emit(ev)
	})
}

func errclassClosed() error {
	return wrapRoundTrip(io.ErrClosedPipe)
}
