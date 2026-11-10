package legacy

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestSessionMintListAndDelete(t *testing.T) {
	lab := startLegacyHTTP(t)
	conn := dialLab(t, lab, Config{})
	handshake(t, conn)
	if conn.SessionID() != "fixture-session" {
		t.Fatalf("session %q", conn.SessionID())
	}
	list, err := conn.Post(t.Context(), rpc(2, "tools/list", map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if !jsonHasTool(list.Body, "echo") {
		t.Fatalf("list body %s", list.Body)
	}
	if closeErr := conn.Close(t.Context()); closeErr != nil {
		t.Fatal(closeErr)
	}
	if !lab.TornDown() {
		t.Fatal("expected DELETE teardown")
	}
	_, err = conn.Post(t.Context(), rpc(3, "tools/list", map[string]any{}))
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("post after delete class %d err %v", errclass.ExitCode(err), err)
	}
}

func TestInvalidSessionIsConnection(t *testing.T) {
	lab := startLegacyHTTP(t)
	owner := dialLab(t, lab, Config{})
	handshake(t, owner)
	other := dialLab(t, lab, Config{})
	_, err := other.Post(t.Context(), rpc(1, "tools/list", map[string]any{}))
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("class %d err %v", errclass.ExitCode(err), err)
	}
}

func TestPOSTJSONAndSSE(t *testing.T) {
	d, err := testkit.Named("legacy-http")
	if err != nil {
		t.Fatal(err)
	}
	d.SSEOnCall = true
	lab := testkit.StartHTTP(t, d)
	conn := dialLab(t, lab, Config{})
	handshake(t, conn)
	call, err := conn.Post(t.Context(), rpc(3, "tools/call", map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"text": "sse"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if call.ContentType != "" && call.Body == nil {
		t.Fatal("sse decoded to empty body")
	}
	if !jsonHas(call.Body, "sse") {
		t.Fatalf("call body %s", call.Body)
	}
}

func TestEndpointDownIsConnection(t *testing.T) {
	lab := startLegacyHTTP(t)
	conn := dialLab(t, lab, Config{})
	lab.Server.Close()
	_, err := conn.Post(t.Context(), rpc(1, "initialize", initParamsMap()))
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("class %d err %v", errclass.ExitCode(err), err)
	}
}

func startLegacyHTTP(t *testing.T) *testkit.HTTPLab {
	t.Helper()
	d, err := testkit.Named("legacy-http")
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

func handshake(t *testing.T, conn *Conn) {
	t.Helper()
	ex, err := conn.Post(t.Context(), rpc(1, "initialize", initParamsMap()))
	if err != nil {
		t.Fatal(err)
	}
	if ex.SessionID == "" {
		t.Fatal("missing session id")
	}
	_, err = conn.Post(t.Context(), rpcNotify("notifications/initialized"))
	if err != nil {
		t.Fatal(err)
	}
}

func initParamsMap() map[string]any {
	return map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcpdev-test", "version": "dev"},
	}
}

func rpc(id any, method string, params any) []byte {
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

func jsonHasTool(body []byte, name string) bool {
	var parsed struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	for _, tool := range parsed.Result.Tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func jsonHas(body []byte, needle string) bool {
	return strings.Contains(string(body), needle)
}
