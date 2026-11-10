package trace

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/port"
)

type stepClock struct {
	n int
}

func (s *stepClock) Now() time.Time {
	s.n++
	return time.Unix(0, int64(s.n)*int64(time.Millisecond))
}

func (s *stepClock) After(time.Duration) <-chan time.Time {
	return nil
}

var _ port.Clock = (*stepClock)(nil)

func TestOrderRedactionTruncation(t *testing.T) {
	var out bytes.Buffer
	rec := New(Options{Clock: &stepClock{}, Writer: &out, Format: FormatJSON, Preview: 48})
	rec.frame(DirClient, "stdio", []byte(`{"authorization":"sekrit","id":1,"jsonrpc":"2.0","method":"initialize","padding":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`))
	rec.HTTPRequest("POST", "http", map[string]string{"Authorization": "Bearer sekrit", "Accept": "application/json"}, []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call"}`))
	rec.HTTPResponse(200, "application/json", []byte(`{"jsonrpc":"2.0","id":2,"result":{"ok":true}}`), false)
	rec.SSE("message", "3", []byte(`{"jsonrpc":"2.0","id":3,"result":{"password":"nope"}}`))
	ev := rec.Events()
	if len(ev) != 4 {
		t.Fatalf("len %d", len(ev))
	}
	for i, e := range ev {
		if e.Seq != i {
			t.Fatalf("seq %d want %d", e.Seq, i)
		}
		if e.Elapsed <= 0 {
			t.Fatalf("elapsed %s at seq %d", e.Elapsed, i)
		}
		if i > 0 && e.Elapsed < ev[i-1].Elapsed {
			t.Fatalf("elapsed went backwards")
		}
	}
	if ev[0].Kind != KindFrame || ev[0].Method != "initialize" || ev[0].RequestID != "1" {
		t.Fatalf("frame %+v", ev[0])
	}
	if strings.Contains(ev[0].Payload, "sekrit") {
		t.Fatalf("secret in payload %s", ev[0].Payload)
	}
	if !strings.Contains(ev[0].Payload, "[REDACTED]") {
		t.Fatalf("expected redaction %s", ev[0].Payload)
	}
	if !ev[0].Truncated {
		t.Fatal("expected truncation")
	}
	if ev[1].Headers["Authorization"] != "[REDACTED]" {
		t.Fatalf("headers %+v", ev[1].Headers)
	}
	if ev[1].Headers["Accept"] != "application/json" {
		t.Fatal("lost Accept")
	}
	if strings.Contains(out.String(), "sekrit") || strings.Contains(out.String(), "nope") {
		t.Fatalf("secret in jsonl %s", out.String())
	}
	if ev[2].Kind != KindHTTPResp || ev[2].HTTPStatus != 200 {
		t.Fatalf("resp %+v", ev[2])
	}
	if ev[3].Kind != KindSSE || ev[3].RequestID != "3" {
		t.Fatalf("sse %+v", ev[3])
	}
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte{'\n'})
	if len(lines) != 4 {
		t.Fatalf("jsonl lines %d", len(lines))
	}
}

func TestFullKeepsLongPayload(t *testing.T) {
	rec := New(Options{Clock: &stepClock{}, Format: FormatJSON, Full: true})
	padding := strings.Repeat("x", DefaultPreviewLimit+64)
	body := `{"id":1,"jsonrpc":"2.0","method":"initialize","padding":"` + padding + `"}`
	rec.frame(DirClient, "stdio", []byte(body))
	ev := rec.Events()
	if len(ev) != 1 {
		t.Fatalf("len %d", len(ev))
	}
	if ev[0].Truncated {
		t.Fatal("verbose payload truncated")
	}
	if !strings.Contains(ev[0].Payload, padding) {
		t.Fatalf("missing padding in %s", ev[0].Payload)
	}
}

func TestHumanWaterfallOmitsSecrets(t *testing.T) {
	var out bytes.Buffer
	rec := New(Options{Clock: &stepClock{}, Writer: &out, Format: FormatHuman})
	rec.Retry("http", "unsupported_protocol_version", 1)
	rec.Fallback("stdio", "legacy_initialize")
	rec.Cancel("http", "context canceled")
	rec.Stderr("stdio", `log {"jsonrpc":"2.0"} token=not-a-header`)
	s := out.String()
	if !strings.Contains(s, "retry") || !strings.Contains(s, "fallback") || !strings.Contains(s, "cancel") || !strings.Contains(s, "stderr") {
		t.Fatalf("waterfall %s", s)
	}
	for _, e := range rec.Events() {
		if e.Kind == KindFrame || e.Kind == KindHTTPReq || e.Kind == KindSSE {
			t.Fatalf("correlation treated as frame %+v", e)
		}
	}
}

func TestDisabledIsNoop(t *testing.T) {
	var rec *Recorder
	if rec.Enabled() {
		t.Fatal("nil enabled")
	}
	r := io.NopCloser(bytes.NewReader([]byte(`{"a":1}\n`)))
	if WrapReader(r, rec, "stdio") != r {
		t.Fatal("wrap reader")
	}
	w := nopWriteCloser{}
	if WrapWriter(w, rec, "stdio") != w {
		t.Fatal("wrap writer")
	}
	rec.frame(DirClient, "stdio", []byte(`{"authorization":"x"}`))
	rec.HTTPRequest("POST", "http", nil, []byte("x"))
	if rec.Events() != nil {
		t.Fatal("events")
	}
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }
