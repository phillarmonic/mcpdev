package modern

import (
	"context"
	"encoding/json/v2"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/testkit"
	"github.com/phillarmonic/mcpdev/internal/trace"
)

func TestDiscoverJSONHeadersAndNoSession(t *testing.T) {
	lab := startModernHTTP(t)
	conn := dialLab(t, lab, Config{})
	ex, err := conn.Post(t.Context(), modernRPC(1, "server/discover", nil))
	if err != nil {
		t.Fatal(err)
	}
	if ex.Status != http.StatusOK {
		t.Fatalf("status %d", ex.Status)
	}
	h := lab.LastRequestHeaders()
	if got := h.Get("Accept"); !strings.Contains(got, "application/json") || !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Accept %q", got)
	}
	if h.Get("MCP-Protocol-Version") != ProtocolVersion {
		t.Fatalf("version %q", h.Get("MCP-Protocol-Version"))
	}
	if h.Get("Mcp-Method") != "server/discover" {
		t.Fatalf("method %q", h.Get("Mcp-Method"))
	}
	if h.Get("Mcp-Session-Id") != "" {
		t.Fatalf("session header %q", h.Get("Mcp-Session-Id"))
	}
	if !jsonHas(ex.Body, "mcpdev-fixture") {
		t.Fatalf("body %s", ex.Body)
	}
}

func TestMcpNameEncodingAndRouting(t *testing.T) {
	lab := startModernHTTP(t)
	conn := dialLab(t, lab, Config{})
	name := "Hello, 世界"
	_, err := conn.Post(t.Context(), modernRPC(2, "tools/call", map[string]any{
		"name":      name,
		"arguments": map[string]any{"text": "x"},
	}))
	if err != nil && errclass.ExitCode(err) == errclass.ExitConnection {
		t.Fatal(err)
	}
	got := lab.LastRequestHeaders().Get("Mcp-Name")
	want := EncodeHeaderValue(name)
	if got != want {
		t.Fatalf("Mcp-Name %q want %q", got, want)
	}
	if !strings.HasPrefix(got, "=?base64?") {
		t.Fatalf("expected encoded name, got %q", got)
	}
	if lab.LastRequestHeaders().Get("Mcp-Method") != "tools/call" {
		t.Fatal("missing Mcp-Method")
	}
}

func TestHeaderMismatchIsProtocol(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32020,"message":"HeaderMismatch"}}`))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	conn, err := Dial(Config{Endpoint: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(t.Context()) })
	_, err = conn.Post(t.Context(), modernRPC(1, "server/discover", nil))
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("class %d err %v", errclass.ExitCode(err), err)
	}
}

func TestNotificationAccepted(t *testing.T) {
	lab := startModernHTTP(t)
	conn := dialLab(t, lab, Config{})
	ex, err := conn.Post(t.Context(), rpcNotify("notifications/progress"))
	if err != nil {
		t.Fatal(err)
	}
	if ex.Status != http.StatusAccepted {
		t.Fatalf("status %d body %s", ex.Status, ex.Body)
	}
	if len(ex.Body) != 0 {
		t.Fatalf("202 must have empty body, got %s", ex.Body)
	}
}

func TestTraceOrderWithoutDoubleCount(t *testing.T) {
	lab := startModernHTTP(t)
	rec := trace.New(trace.Options{})
	conn := dialLab(t, lab, Config{Tracer: rec})
	_, err := conn.Post(t.Context(), modernRPC(1, "server/discover", nil))
	if err != nil {
		t.Fatal(err)
	}
	ev := rec.Events()
	if len(ev) < 2 {
		t.Fatalf("events %d", len(ev))
	}
	if ev[0].Kind != trace.KindHTTPReq || ev[1].Kind != trace.KindHTTPResp {
		t.Fatalf("order %s %s", ev[0].Kind, ev[1].Kind)
	}
	frames := 0
	for _, e := range ev {
		if e.Kind == trace.KindFrame {
			frames++
		}
	}
	if frames != 0 {
		t.Fatalf("http traced as frames: %+v", ev)
	}
}

func TestRequestScopedSSE(t *testing.T) {
	d, err := testkit.Named("modern-sse")
	if err != nil {
		t.Fatal(err)
	}
	lab := testkit.StartHTTP(t, d)
	rec := trace.New(trace.Options{})
	conn := dialLab(t, lab, Config{Tracer: rec})
	ex, err := conn.Post(t.Context(), modernRPC(3, "tools/call", map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"text": "sse"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ex.ContentType, "text/event-stream") {
		t.Fatalf("content-type %q", ex.ContentType)
	}
	if !jsonHas(ex.Body, "sse") {
		t.Fatalf("decoded body %s", ex.Body)
	}
	if len(ex.Events) == 0 {
		t.Fatal("expected SSE events")
	}
	var kinds []trace.Kind
	for _, e := range rec.Events() {
		kinds = append(kinds, e.Kind)
		if e.Kind == trace.KindHTTPResp && e.Payload != "" {
			t.Fatalf("SSE HTTP response also stored body: %+v", e)
		}
		if e.Kind == trace.KindFrame {
			t.Fatalf("SSE counted as frame: %+v", e)
		}
	}
	if !slices.Contains(kinds, trace.KindHTTPReq) || !slices.Contains(kinds, trace.KindHTTPResp) || !slices.Contains(kinds, trace.KindSSE) {
		t.Fatalf("kinds %v", kinds)
	}
}

func TestCancelDoesNotSendCancelled(t *testing.T) {
	var mu sync.Mutex
	var methods []string
	started := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var env struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &env)
		mu.Lock()
		methods = append(methods, env.Method)
		mu.Unlock()
		select {
		case <-started:
		default:
			close(started)
		}
		<-r.Context().Done()
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	conn, err := Dial(Config{Endpoint: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(t.Context()) })
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, postErr := conn.Post(ctx, modernRPC(1, "server/discover", nil))
		done <- postErr
	}()
	<-started
	cancel()
	err = <-done
	if err == nil {
		t.Fatal("expected cancel error")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, method := range methods {
		if method == "notifications/cancelled" {
			t.Fatal("must not POST notifications/cancelled")
		}
	}
}

func TestStatelessLoadBalancedPosts(t *testing.T) {
	a := startModernHTTP(t)
	b := startModernHTTP(t)
	ca := dialLab(t, a, Config{})
	cb := dialLab(t, b, Config{})
	if _, err := ca.Post(t.Context(), modernRPC(1, "server/discover", nil)); err != nil {
		t.Fatal(err)
	}
	if _, err := cb.Post(t.Context(), modernRPC(2, "tools/list", nil)); err != nil {
		t.Fatal(err)
	}
	if a.LastRequestHeaders().Get("Mcp-Session-Id") != "" || b.LastRequestHeaders().Get("Mcp-Session-Id") != "" {
		t.Fatal("load-balanced posts must not send a session header")
	}
	if err := ca.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if a.GETCount() != 0 {
		t.Fatalf("GET count %d", a.GETCount())
	}
}

func startModernHTTP(t *testing.T) *testkit.HTTPLab {
	t.Helper()
	d, err := testkit.Named("modern-http")
	if err != nil {
		t.Fatal(err)
	}
	return testkit.StartHTTP(t, d)
}

func dialLab(t *testing.T, lab *testkit.HTTPLab, cfg Config) *Conn {
	t.Helper()
	cfg.Endpoint = lab.Server.URL
	conn, err := Dial(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(t.Context()) })
	return conn
}

func modernRPC(id any, method string, extra map[string]any) []byte {
	meta := map[string]any{
		"io.modelcontextprotocol/protocolVersion": ProtocolVersion,
		"io.modelcontextprotocol/clientInfo": map[string]any{
			"name":    "mcpdev-test",
			"version": "0",
		},
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}
	params := map[string]any{"_meta": meta}
	maps.Copy(params, extra)
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}, json.Deterministic(true))
	if err != nil {
		panic(err)
	}
	return body
}

func rpcNotify(method string) []byte {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}, json.Deterministic(true))
	if err != nil {
		panic(err)
	}
	return body
}

func jsonHas(body []byte, needle string) bool {
	return strings.Contains(string(body), needle)
}
