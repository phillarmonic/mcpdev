package testkit

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

// HTTPLab is a loopback Streamable HTTP fixture.
type HTTPLab struct {
	lastHeaders http.Header
	Server      *httptest.Server
	state       *SessionState
	rpcMethods  []string
	Desc        Descriptor
	mu          sync.Mutex
}

// StartHTTP serves the fixture on loopback with httptest.NewServer.
func StartHTTP(t testing.TB, d Descriptor) *HTTPLab {
	return startHTTP(t, d, false)
}

// StartHTTPS serves the fixture with httptest.NewTLSServer.
func StartHTTPS(t testing.TB, d Descriptor) *HTTPLab {
	return startHTTP(t, d, true)
}

func startHTTP(t testing.TB, d Descriptor, tls bool) *HTTPLab {
	t.Helper()
	if d.Transport != TransportHTTP {
		t.Fatalf("testkit: fixture %q is transport %s, not http", d.Name, d.Transport)
	}
	lab := &HTTPLab{Desc: d, state: &SessionState{}}
	if tls {
		lab.Server = httptest.NewTLSServer(lab)
	} else {
		lab.Server = httptest.NewServer(lab)
	}
	t.Cleanup(lab.Server.Close)
	return lab
}

// TornDown reports whether DELETE ended the protocol session.
func (l *HTTPLab) TornDown() bool {
	return l.state.isTornDown()
}

// GETCount is how many standalone GET SSE requests the fixture has seen.
func (l *HTTPLab) GETCount() int {
	return l.state.GETCount()
}

// SeenLastEventID is the Last-Event-ID header from the most recent GET.
func (l *HTTPLab) SeenLastEventID() string {
	return l.state.SeenLastEventID()
}

// RPCMethods is the JSON-RPC methods seen on POST, oldest first.
func (l *HTTPLab) RPCMethods() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.rpcMethods)
}

// LastRequestHeaders is the HTTP header set from the most recent POST.
func (l *HTTPLab) LastRequestHeaders() http.Header {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastHeaders.Clone()
}

func (l *HTTPLab) notePOST(r *http.Request, method string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rpcMethods = append(l.rpcMethods, method)
	l.lastHeaders = r.Header.Clone()
}

func (l *HTTPLab) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !l.authorize(w, r) {
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		if l.Desc.Era == Era20260728 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodDelete {
			l.serveLegacyDELETE(w, r)
			return
		}
		l.serveLegacyGET(w, r)
		return
	case http.MethodPost:
		l.servePOST(ctx, w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (l *HTTPLab) authorize(w http.ResponseWriter, r *http.Request) bool {
	if !l.Desc.AuthRequired {
		return true
	}
	want := "Bearer " + l.Desc.AuthToken
	if r.Header.Get("Authorization") != want {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (l *HTTPLab) serveLegacyGET(w http.ResponseWriter, r *http.Request) {
	if !l.legacySessionOK(w, r) {
		return
	}
	n := l.state.noteGET(r.Header.Get("Last-Event-ID"))
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	if fl, ok := w.(http.Flusher); ok {
		if l.Desc.NotifyListChanged {
			_, _ = fmt.Fprintf(w, "id: %d\nevent: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/tools/list_changed\"}\n\n", n)
		}
		fl.Flush()
	}
	if l.Desc.DropGETOnce && n == 1 {
		return
	}
	<-r.Context().Done()
}

func (l *HTTPLab) serveLegacyDELETE(w http.ResponseWriter, r *http.Request) {
	if !l.legacySessionOK(w, r) {
		return
	}
	l.state.tearDown()
	w.WriteHeader(http.StatusNoContent)
}

func (l *HTTPLab) legacySessionOK(w http.ResponseWriter, r *http.Request) bool {
	if l.state.isTornDown() {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	if !l.Desc.Session.MintID {
		return true
	}
	id := l.state.currentSessionID()
	if id == "" {
		return true
	}
	if r.Header.Get("Mcp-Session-Id") != id {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	return true
}

func (l *HTTPLab) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch l.Desc.Malformed {
	case MalformedEmptyHTTPBody:
		w.WriteHeader(http.StatusBadRequest)
		return
	case MalformedUnrecognizedError:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","error":{"code":-1,"message":"nope"}}`)
		return
	case MalformedJSON:
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-json\n")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = writeHTTPJSON(w, rpcError(nil, codeParse, "parse error", nil))
		return
	}
	l.notePOST(r, req.Method)
	if l.Desc.Era == Era20260728 && req.ID != nil {
		if resp, ok := checkModernHTTPHeaders(r, req); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = writeHTTPJSON(w, resp)
			return
		}
	}
	if req.ID == nil && l.Desc.Era == Era20260728 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if l.Desc.Era == Era20251125 && l.Desc.Session.MintID && req.Method != "initialize" {
		if l.state.isTornDown() || r.Header.Get("Mcp-Session-Id") != l.state.currentSessionID() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if ver := r.Header.Get("MCP-Protocol-Version"); ver != "" && ver != string(l.Desc.Era) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = writeHTTPJSON(w, rpcError(req.ID, codeHeaderMismatch, "HeaderMismatch", nil))
			return
		}
	}
	resp := handle(ctx, l.Desc, l.state, req)
	if req.Method == "initialize" && l.Desc.Session.MintID {
		w.Header().Set("Mcp-Session-Id", l.Desc.sessionID())
	}
	if l.Desc.SSEOnCall && req.Method == "tools/call" && wantsSSE(r) {
		writeSSE(w, resp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if resp.Error != nil && (resp.Error.Code == codeHeaderMismatch || resp.Error.Code == codeUnsupportedVer) {
		w.WriteHeader(http.StatusBadRequest)
	}
	_ = writeHTTPJSON(w, resp)
}

func checkModernHTTPHeaders(r *http.Request, req Request) (Response, bool) {
	proto := r.Header.Get("MCP-Protocol-Version")
	method := r.Header.Get("Mcp-Method")
	bodyVer := modernMetaVersion(req)
	if proto == "" || proto != bodyVer {
		return rpcError(req.ID, codeHeaderMismatch, "HeaderMismatch", nil), false
	}
	if method != req.Method {
		return rpcError(req.ID, codeHeaderMismatch, "HeaderMismatch", nil), false
	}
	switch req.Method {
	case "tools/call", "prompts/get", "resources/read":
		raw := r.Header.Get("Mcp-Name")
		if raw == "" {
			return rpcError(req.ID, codeHeaderMismatch, "HeaderMismatch", map[string]any{"missing": "Mcp-Name"}), false
		}
		p, err := decodeNamed(req)
		if err != nil {
			return rpcError(req.ID, codeHeaderMismatch, "HeaderMismatch", nil), false
		}
		want := p.Name
		if req.Method == "resources/read" {
			want = p.URI
		}
		if decodeMCPHeader(raw) != want {
			return rpcError(req.ID, codeHeaderMismatch, "HeaderMismatch", map[string]any{"header": raw}), false
		}
	}
	return Response{}, true
}

func decodeMCPHeader(v string) string {
	const prefix, suffix = "=?base64?", "?="
	if strings.HasPrefix(v, prefix) && strings.HasSuffix(v, suffix) {
		inner := strings.TrimSuffix(strings.TrimPrefix(v, prefix), suffix)
		b, err := base64.StdEncoding.DecodeString(inner)
		if err != nil {
			return v
		}
		return string(b)
	}
	return v
}

func wantsSSE(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

func writeSSE(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	b, err := json.Marshal(resp, json.Deterministic(true))
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", b)
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}
}

func writeHTTPJSON(w http.ResponseWriter, v any) error {
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

type httpExchange struct {
	Header     http.Header
	Body       []byte
	StatusCode int
}

func postJSON(t testing.TB, lab *HTTPLab, headers map[string]string, req Request) httpExchange {
	t.Helper()
	body := mustMarshal(req)
	httpReq, err := http.NewRequestWithContext(t.Context(), http.MethodPost, lab.Server.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := lab.Server.Client().Do(httpReq)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("close response body: %v", closeErr)
		}
	}()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return httpExchange{Header: resp.Header.Clone(), Body: got, StatusCode: resp.StatusCode}
}

func decodeHTTPResponse(t testing.TB, ex httpExchange) Response {
	t.Helper()
	var out Response
	if err := json.Unmarshal(ex.Body, &out); err != nil {
		t.Fatalf("decode %d: %v body=%s", ex.StatusCode, err, ex.Body)
	}
	return out
}
