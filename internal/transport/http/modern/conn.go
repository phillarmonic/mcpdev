package modern

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

// Exchange is one POST response after JSON or request-scoped SSE decoding.
type Exchange struct {
	Events      []Event
	ContentType string
	Body        []byte
	Status      int
}

// Conn is a stateless 2026-07-28 Streamable HTTP client.
// It never stores or sends Mcp-Session-Id.
type Conn struct {
	client   *http.Client
	headers  map[string]string
	tracer   *trace.Recorder
	endpoint string
	origin   string
	mu       sync.Mutex
	closed   bool
}

// Dial validates the endpoint and builds a client. It does not handshake or create a session.
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
		client:   client,
		endpoint: u.String(),
		origin:   originFor(cfg, u),
		headers:  cfg.Headers,
		tracer:   cfg.Tracer,
	}, nil
}

// Post sends one self-contained JSON-RPC message.
func (c *Conn) Post(ctx context.Context, payload []byte) (Exchange, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return Exchange{}, wrapRoundTrip(io.ErrClosedPipe)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return Exchange{}, wrapRoundTrip(err)
	}
	if headerErr := c.applyHeaders(req, payload); headerErr != nil {
		return Exchange{}, headerErr
	}
	c.traceRequest(req, payload)
	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			c.tracer.Cancel("http", ctx.Err().Error())
		}
		return Exchange{}, wrapRoundTrip(err)
	}
	defer func() { _ = resp.Body.Close() }()
	ct := resp.Header.Get("Content-Type")
	ex := Exchange{Status: resp.StatusCode, ContentType: ct}
	if resp.StatusCode == http.StatusAccepted {
		c.traceExchange(ex, false)
		return ex, nil
	}
	if statusErr := classifyStatus(resp.StatusCode, nil, ct); statusErr != nil {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil && ctx.Err() != nil {
			c.tracer.Cancel("http", ctx.Err().Error())
			return Exchange{}, wrapRoundTrip(ctx.Err())
		}
		classified := classifyStatus(resp.StatusCode, raw, ct)
		ex.Body = raw
		c.traceExchange(ex, false)
		return ex, classified
	}
	body, events, err := decodeResponse(ct, resp.Body)
	if err != nil {
		if ctx.Err() != nil {
			c.tracer.Cancel("http", ctx.Err().Error())
			return Exchange{}, wrapRoundTrip(ctx.Err())
		}
		return Exchange{}, err
	}
	ex.Body = body
	ex.Events = events
	c.traceExchange(ex, strings.Contains(ct, "text/event-stream"))
	return ex, nil
}

// Close releases the client. It does not DELETE or assume a protocol session.
func (c *Conn) Close(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *Conn) applyHeaders(req *http.Request, payload []byte) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Origin", c.origin)
	req.Header.Set("MCP-Protocol-Version", versionFromPayload(payload))
	method, name := routingFromPayload(payload)
	if method != "" {
		req.Header.Set("Mcp-Method", method)
	}
	if methodNeedsName(method) {
		if name == "" {
			return errclass.Protocol{Err: errors.New("http: Mcp-Name required for " + method)}
		}
		req.Header.Set("Mcp-Name", EncodeHeaderValue(name))
	}
	return nil
}

func (c *Conn) traceRequest(req *http.Request, payload []byte) {
	c.tracer.HTTPRequest(req.Method, "http", trace.FlattenHeader(req.Header), payload)
}

func (c *Conn) traceExchange(ex Exchange, sse bool) {
	c.tracer.HTTPResponse(ex.Status, ex.ContentType, ex.Body, sse)
	for _, ev := range ex.Events {
		c.tracer.SSE(ev.Name, ev.ID, ev.Data)
	}
}
