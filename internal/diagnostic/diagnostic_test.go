package diagnostic

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

func TestFromErrorAndFields(t *testing.T) {
	cause := errclass.Protocol{Err: errors.New("bad rpc")}
	r := FromError(cause, Record{
		Operation: "initialize",
		Transport: "stdio",
		Era:       model.Era20251125,
		Elapsed:   12 * time.Millisecond,
	})
	if r.Code != CodeProtocol || r.Severity != SeverityError {
		t.Fatalf("code=%s severity=%s", r.Code, r.Severity)
	}
	if !r.ProtocolFailure() {
		t.Fatal("expected protocol failure")
	}
	if r.ExitClass() != errclass.ExitProtocol {
		t.Fatalf("exit %d", r.ExitClass())
	}
	if !errors.Is(r, cause.Err) {
		t.Fatal("unwrap")
	}
}

func TestChildStderrIsNotProtocolFailure(t *testing.T) {
	stderr := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"looks like rpc"}}`
	r := AttachChildStderr(Record{}, stderr)
	if r.ProtocolFailure() {
		t.Fatal("child stderr treated as protocol failure")
	}
	if r.Code != CodeChildStderr {
		t.Fatalf("code %s", r.Code)
	}
	if r.ExitClass() != errclass.ExitOK {
		t.Fatalf("exit %d", r.ExitClass())
	}
	conn := AttachChildStderr(FromError(errclass.Connection{Err: errors.New("spawn")}, Record{Operation: "connect"}), stderr)
	if conn.Code != CodeConnection {
		t.Fatalf("overwrote connection: %s", conn.Code)
	}
	if conn.ProtocolFailure() {
		t.Fatal("stderr promoted connection to protocol")
	}
}

func TestRenderersRedact(t *testing.T) {
	r := Record{
		Code:      CodeConnection,
		Severity:  SeverityError,
		Message:   "https://user:secret@example.com/mcp",
		Operation: "connect",
		Transport: "http",
		Era:       model.Era20260728,
		Details: map[string]any{
			"headers": map[string]any{"Authorization": "Bearer abc", "Accept": "application/json"},
			"command": []string{"mcpdev", "--header", "Authorization: Bearer abc", "API_KEY=k"},
			"_meta":   map[string]any{"token": "t", "era": "2026-07-28"},
		},
		ChildStderr: "Authorization: Bearer abc",
	}
	b, err := r.MarshalJSONRedacted(redact.Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, leak := range []string{"user:secret", "Bearer abc", "API_KEY=k"} {
		if strings.Contains(s, leak) {
			t.Fatalf("json leak %q in %s", leak, s)
		}
	}
	var env map[string]any
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteHuman(&buf, r, redact.Options{}); err != nil {
		t.Fatal(err)
	}
	h := buf.String()
	if strings.Contains(h, "secret") || strings.Contains(h, "Bearer abc") {
		t.Fatalf("human leak: %s", h)
	}
}

func TestJSONTruncationStableAndValid(t *testing.T) {
	long := strings.Repeat("a", 80) + strings.Repeat("b", 80)
	r := Record{
		Code:         CodeInternal,
		Severity:     SeverityError,
		Message:      long,
		ChildStderr:  long,
		PreviewLimit: 32,
		Details:      map[string]any{"body": long, "ok": true},
	}
	b1, err := r.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	b2, err := r.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("unstable\n%s\n%s", b1, b2)
	}
	var env envelope
	if err := json.Unmarshal(b1, &env); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, b1)
	}
	if !env.Truncated {
		t.Fatal("expected truncated")
	}
	if env.PreviewLimit != 32 {
		t.Fatalf("preview_limit %d", env.PreviewLimit)
	}
	if len(env.Message) != 32 || len(env.ChildStderr) != 32 {
		t.Fatalf("preview lengths message=%d stderr=%d", len(env.Message), len(env.ChildStderr))
	}
	details, _ := env.Details["body"].(string)
	if len(details) != 32 {
		t.Fatalf("details body %q", details)
	}
}
