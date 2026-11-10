package legacy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/trace"
)

// Exchange is one POST response after JSON or SSE decoding.
type Exchange struct {
	Events      []Event
	SessionID   string
	ContentType string
	Body        []byte
	Status      int
}

// Conn is a stateful 2025-11-25 Streamable HTTP client.
type Conn struct {
	client       *http.Client
	listenCancel context.CancelFunc
	listenDone   chan struct{}
	headers      map[string]string
	tracer       *trace.Recorder
	endpoint     string
	origin       string
	sessionID    string
	lastEventID  string
	cfg          Config
	mu           sync.Mutex
	closed       bool
}

// Dial validates the endpoint and builds a client. It does not handshake.
func Dial(cfg Config) (*Conn, error) {
	u, err := parseEndpoint(cfg.Endpoint, cfg.AllowInsecure)
	if err != nil {
		return nil, err
	}
	client, err := buildClient(cfg, u)
	if err != nil {
		return nil, err
	}
	return &Conn{
		cfg:      cfg,
		client:   client,
		endpoint: u.String(),
		origin:   originFor(cfg, u),
		headers:  cfg.Headers,
		tracer:   cfg.Tracer,
	}, nil
}

// SessionID is the Mcp-Session-Id captured from initialize, if any.
func (c *Conn) SessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// Post sends one JSON-RPC message. JSON and SSE responses both yield the RPC body.
func (c *Conn) Post(ctx context.Context, payload []byte) (Exchange, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return Exchange{}, wrapRoundTrip(err)
	}
	c.applyHeaders(req, true)
	c.tracer.HTTPRequest(req.Method, "http", trace.FlattenHeader(req.Header), payload)
	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			c.tracer.Cancel("http", ctx.Err().Error())
		}
		return Exchange{}, wrapRoundTrip(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Exchange{}, wrapRoundTrip(err)
	}
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.mu.Lock()
		c.sessionID = sid
		c.mu.Unlock()
	}
	ct := resp.Header.Get("Content-Type")
	statusErr := classifyStatus(resp.StatusCode, raw, ct)
	if statusErr != nil {
		ex := Exchange{Status: resp.StatusCode, ContentType: ct, Body: raw, SessionID: c.SessionID()}
		c.traceExchange(ex, false)
		return ex, statusErr
	}
	body, events, err := decodePOSTBody(ct, raw)
	if err != nil {
		return Exchange{}, err
	}
	ex := Exchange{
		Status:      resp.StatusCode,
		ContentType: ct,
		Body:        body,
		SessionID:   c.SessionID(),
		Events:      events,
	}
	c.traceExchange(ex, strings.Contains(ct, "text/event-stream"))
	return ex, nil
}

func (c *Conn) traceExchange(ex Exchange, sse bool) {
	c.tracer.HTTPResponse(ex.Status, ex.ContentType, ex.Body, sse)
	for _, ev := range ex.Events {
		c.tracer.SSE(ev.Name, ev.ID, ev.Data)
	}
}

// Close cancels GET listen and DELETE-tears down a minted session.
func (c *Conn) Close(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	done := c.listenDone
	cancelListen := c.listenCancel
	sid := c.sessionID
	c.mu.Unlock()
	if cancelListen != nil {
		cancelListen()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
	if sid == "" {
		return nil
	}
	return c.deleteSession(ctx, sid)
}

func (c *Conn) deleteSession(ctx context.Context, sid string) error {
	delCtx := context.WithoutCancel(ctx)
	req, err := http.NewRequestWithContext(delCtx, http.MethodDelete, c.endpoint, nil)
	if err != nil {
		return wrapRoundTrip(err)
	}
	c.applyHeaders(req, false)
	req.Header.Set("Mcp-Session-Id", sid)
	resp, err := c.client.Do(req)
	if err != nil {
		return wrapRoundTrip(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	err = classifyStatus(resp.StatusCode, body, resp.Header.Get("Content-Type"))
	if errclass.ExitCode(err) == errclass.ExitConnection && errors.Is(err, errSessionNotFound) {
		return nil
	}
	return err
}

func (c *Conn) applyHeaders(req *http.Request, post bool) {
	if post {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	} else {
		req.Header.Set("Accept", "text/event-stream")
	}
	req.Header.Set("MCP-Protocol-Version", ProtocolVersion)
	req.Header.Set("Origin", c.origin)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	c.mu.Lock()
	sid := c.sessionID
	last := c.lastEventID
	c.mu.Unlock()
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
	if !post && last != "" {
		req.Header.Set("Last-Event-ID", last)
	}
}
