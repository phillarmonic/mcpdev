package testkit

import (
	"net/http"
	"strings"
	"testing"
)

func TestLegacyHTTPSessionAndPrimitives(t *testing.T) {
	d, err := Named("legacy-http")
	if err != nil {
		t.Fatal(err)
	}
	lab := StartHTTP(t, d)
	resp := postJSON(t, lab, map[string]string{
		"MCP-Protocol-Version": string(Era20251125),
	}, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "initialize", Params: initParams()})
	if resp.Header.Get("Mcp-Session-Id") != "fixture-session" {
		t.Fatalf("session id: %q", resp.Header.Get("Mcp-Session-Id"))
	}
	decoded := decodeHTTPResponse(t, resp)
	if decoded.Error != nil {
		t.Fatal(decoded.Error)
	}
	_ = postJSON(t, lab, map[string]string{
		"Mcp-Session-Id":       "fixture-session",
		"MCP-Protocol-Version": string(Era20251125),
	}, Request{JSONRPC: jsonRPCVersion, Method: "notifications/initialized"})

	list := postJSON(t, lab, map[string]string{
		"Mcp-Session-Id":       "fixture-session",
		"MCP-Protocol-Version": string(Era20251125),
	}, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "tools/list", Params: mustMarshal(map[string]any{})})
	out := decodeHTTPResponse(t, list)
	if out.Error != nil {
		t.Fatal(out.Error)
	}

	del, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, lab.Server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	del.Header.Set("Mcp-Session-Id", "fixture-session")
	delResp, err := lab.Server.Client().Do(del)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := delResp.Body.Close(); err != nil {
			t.Errorf("close DELETE body: %v", err)
		}
	})
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status %d", delResp.StatusCode)
	}
	if !lab.TornDown() {
		t.Fatal("expected teardown")
	}
	after := postJSON(t, lab, map[string]string{
		"Mcp-Session-Id":       "fixture-session",
		"MCP-Protocol-Version": string(Era20251125),
	}, Request{JSONRPC: jsonRPCVersion, ID: 3, Method: "tools/list", Params: mustMarshal(map[string]any{})})
	if after.StatusCode != http.StatusNotFound {
		t.Fatalf("post-teardown status %d", after.StatusCode)
	}
}

func TestModernHTTPDiscoverHeadersAnd405(t *testing.T) {
	d, err := Named("modern-http")
	if err != nil {
		t.Fatal(err)
	}
	lab := StartHTTP(t, d)
	get, err := http.NewRequestWithContext(t.Context(), http.MethodGet, lab.Server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	getResp, err := lab.Server.Client().Do(get)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := getResp.Body.Close(); err != nil {
			t.Errorf("close GET body: %v", err)
		}
	})
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status %d", getResp.StatusCode)
	}

	bad := postJSON(t, lab, map[string]string{
		"MCP-Protocol-Version": "2025-11-25",
		"Mcp-Method":           "server/discover",
	}, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "server/discover", Params: modernParams(nil)})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("header mismatch status %d", bad.StatusCode)
	}
	badRPC := decodeHTTPResponse(t, bad)
	if badRPC.Error == nil || badRPC.Error.Code != codeHeaderMismatch {
		t.Fatalf("want HeaderMismatch, got %+v", badRPC)
	}

	ok := postJSON(t, lab, map[string]string{
		"MCP-Protocol-Version": string(Era20260728),
		"Mcp-Method":           "server/discover",
	}, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "server/discover", Params: modernParams(nil)})
	out := decodeHTTPResponse(t, ok)
	if out.Error != nil {
		t.Fatal(out.Error)
	}
}

func TestModernAuthAndSSE(t *testing.T) {
	auth, err := Named("modern-auth")
	if err != nil {
		t.Fatal(err)
	}
	lab := StartHTTP(t, auth)
	unauth := postJSON(t, lab, map[string]string{
		"MCP-Protocol-Version": string(Era20260728),
		"Mcp-Method":           "server/discover",
	}, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "server/discover", Params: modernParams(nil)})

	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", unauth.StatusCode)
	}

	authOK := postJSON(t, lab, map[string]string{
		"Authorization":        "Bearer fixture-token",
		"MCP-Protocol-Version": string(Era20260728),
		"Mcp-Method":           "server/discover",
	}, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "server/discover", Params: modernParams(nil)})
	out := decodeHTTPResponse(t, authOK)
	if out.Error != nil {
		t.Fatal(out.Error)
	}

	sseDesc, err := Named("modern-sse")
	if err != nil {
		t.Fatal(err)
	}
	sseLab := StartHTTP(t, sseDesc)
	call := postJSON(t, sseLab, map[string]string{
		"MCP-Protocol-Version": string(Era20260728),
		"Mcp-Method":           "tools/call",
		"Mcp-Name":             "echo",
		"Accept":               "text/event-stream",
	}, Request{
		JSONRPC: jsonRPCVersion, ID: 3, Method: "tools/call",
		Params: modernParams(map[string]any{"name": "echo", "arguments": map[string]any{"text": "sse"}}),
	})

	if !strings.Contains(call.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content-type %q", call.Header.Get("Content-Type"))
	}
	if !strings.Contains(string(call.Body), "event: message") {
		t.Fatalf("sse body %s", call.Body)
	}
}

func TestModernHTTPNotification202(t *testing.T) {
	d, err := Named("modern-http")
	if err != nil {
		t.Fatal(err)
	}
	lab := StartHTTP(t, d)
	ex := postJSON(t, lab, map[string]string{
		"MCP-Protocol-Version": string(Era20260728),
		"Mcp-Method":           "notifications/progress",
	}, Request{JSONRPC: jsonRPCVersion, Method: "notifications/progress", Params: modernParams(nil)})
	if ex.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", ex.StatusCode)
	}
	if len(ex.Body) != 0 {
		t.Fatalf("202 body %s", ex.Body)
	}
}

func TestMalformedHTTPIsExplicit(t *testing.T) {
	d, err := Named("malformed-http")
	if err != nil {
		t.Fatal(err)
	}
	lab := StartHTTP(t, d)
	resp := postJSON(t, lab, nil, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "server/discover", Params: modernParams(nil)})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if len(resp.Body) != 0 {
		t.Fatalf("empty-http-body must not invent JSON, got %s", resp.Body)
	}
}
