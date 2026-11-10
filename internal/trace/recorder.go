package trace

import (
	"io"
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Recorder appends redacted events with monotonic sequence and elapsed time.
// A nil receiver is disabled.
type Recorder struct {
	start     time.Time
	clock     port.Clock
	out       io.Writer
	format    Format
	extraKeys []string
	events    []Event
	preview   int
	seq       int
	mu        sync.Mutex
	enabled   bool
}

// Options construct an enabled recorder. Tracing is off when New is not called.
type Options struct {
	Clock     port.Clock
	Writer    io.Writer
	Format    Format
	ExtraKeys []string
	Preview   int
	Full      bool
}

// New returns an enabled recorder. Writer may be nil to keep events in memory only.
func New(opts Options) *Recorder {
	cl := opts.Clock
	if cl == nil {
		cl = clock.Wall{}
	}
	preview := opts.Preview
	switch {
	case opts.Full:
		preview = unlimitedPreview
	case preview <= 0:
		preview = DefaultPreviewLimit
	}
	format := opts.Format
	if format == "" {
		format = FormatHuman
	}
	return &Recorder{
		clock:     cl,
		out:       opts.Writer,
		extraKeys: append([]string(nil), opts.ExtraKeys...),
		start:     cl.Now(),
		format:    format,
		preview:   preview,
		enabled:   true,
	}
}

// Enabled reports that payloads may be copied and events will be stored.
func (r *Recorder) Enabled() bool {
	return r != nil && r.enabled
}

// Events returns a copy of recorded events in sequence order.
func (r *Recorder) Events() []Event {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}

// Emit stores one event. Payloads must already be redacted; prefer typed helpers.
func (r *Recorder) Emit(ev Event) {
	if !r.Enabled() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ev.Seq = r.seq
	r.seq++
	ev.Elapsed = r.clock.Now().Sub(r.start)
	ev.Elapsed = max(ev.Elapsed, 0)
	r.events = append(r.events, ev)
	if r.out != nil {
		_ = writeOne(r.out, ev, r.format)
	}
}

func (r *Recorder) frame(dir Direction, transport string, payload []byte) {
	if !r.Enabled() {
		return
	}
	method, id := inspectRPC(payload)
	text, trunc := preview(payload, r.extraKeys, r.preview)
	r.Emit(Event{
		Kind:         KindFrame,
		Direction:    dir,
		Transport:    transport,
		Method:       method,
		RequestID:    id,
		Payload:      text,
		PayloadBytes: len(payload),
		Truncated:    trunc,
	})
}

// HTTPRequest records one client HTTP round-trip start.
func (r *Recorder) HTTPRequest(httpMethod, transport string, headers map[string]string, payload []byte) {
	if !r.Enabled() {
		return
	}
	method, id := inspectRPC(payload)
	if method == "" {
		method = httpMethod
	}
	text, trunc := preview(payload, r.extraKeys, r.preview)
	r.Emit(Event{
		Kind:         KindHTTPReq,
		Direction:    DirClient,
		Transport:    transport,
		Method:       method,
		RequestID:    id,
		Headers:      redactHeaders(headers, r.extraKeys),
		Payload:      text,
		PayloadBytes: len(payload),
		Truncated:    trunc,
	})
}

// HTTPResponse records status and a JSON body preview. SSE bodies stay on SSE events.
func (r *Recorder) HTTPResponse(status int, contentType string, payload []byte, sse bool) {
	if !r.Enabled() {
		return
	}
	ev := Event{
		Kind:       KindHTTPResp,
		Direction:  DirServer,
		Transport:  "http",
		HTTPStatus: status,
		Note:       contentType,
	}
	if !sse && len(payload) > 0 {
		method, id := inspectRPC(payload)
		ev.Method = method
		ev.RequestID = id
		text, trunc := preview(payload, r.extraKeys, r.preview)
		ev.Payload = text
		ev.PayloadBytes = len(payload)
		ev.Truncated = trunc
	}
	r.Emit(ev)
}

// SSE records one server-sent event. It is not also a frame or HTTP body.
func (r *Recorder) SSE(name, id string, data []byte) {
	if !r.Enabled() {
		return
	}
	method, rpcID := inspectRPC(data)
	text, trunc := preview(data, r.extraKeys, r.preview)
	note := name
	if id != "" {
		if note != "" {
			note += " "
		}
		note += "id=" + id
	}
	r.Emit(Event{
		Kind:         KindSSE,
		Direction:    DirServer,
		Transport:    "http",
		Method:       method,
		RequestID:    rpcID,
		Note:         note,
		Payload:      text,
		PayloadBytes: len(data),
		Truncated:    trunc,
	})
}

// Retry records a protocol or HTTP reconnect attempt.
func (r *Recorder) Retry(transport, note string, attempt int) {
	r.Emit(Event{Kind: KindRetry, Direction: DirInternal, Transport: transport, Note: note, Attempt: attempt})
}

// Fallback records initialize-after-discover (or equivalent) without hiding the branch.
func (r *Recorder) Fallback(transport, note string) {
	r.Emit(Event{Kind: KindFallback, Direction: DirInternal, Transport: transport, Note: note})
}

// Cancel records a context deadline or interrupt. It is not a JSON-RPC cancelled notification.
func (r *Recorder) Cancel(transport, note string) {
	r.Emit(Event{Kind: KindCancel, Direction: DirInternal, Transport: transport, Note: note})
}

// Stderr records bounded child stderr. The text is never parsed as a frame.
func (r *Recorder) Stderr(transport, text string) {
	if !r.Enabled() || text == "" {
		return
	}
	previewText, trunc := boundString(text, r.preview)
	r.Emit(Event{
		Kind:         KindStderr,
		Direction:    DirInternal,
		Transport:    transport,
		Payload:      previewText,
		PayloadBytes: len(text),
		Truncated:    trunc,
	})
}
