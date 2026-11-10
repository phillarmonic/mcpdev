package testkit

import (
	"strings"
	"testing"
)

func TestProcessStdioKeepsProtocolOnStdout(t *testing.T) {
	lab := StartProcess(t, "legacy-stdio")
	resp, err := CallProcess(lab, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "initialize", Params: initParams()})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if writeErr := writeJSONLine(lab.Stdin, Request{JSONRPC: jsonRPCVersion, Method: "notifications/initialized"}); writeErr != nil {
		t.Fatal(writeErr)
	}
	listed, err := CallProcess(lab, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "tools/list", Params: mustMarshal(map[string]any{})})
	if err != nil {
		t.Fatal(err)
	}
	if listed.Error != nil {
		t.Fatal(listed.Error)
	}
	logs := lab.Stderr.String()
	if !strings.Contains(logs, "fixture: started") {
		t.Fatalf("stderr logs missing, got %q", logs)
	}
	if strings.Contains(logs, `"jsonrpc"`) {
		t.Fatalf("protocol leaked to stderr: %q", logs)
	}
	if err := lab.Close(); err != nil {
		t.Fatal(err)
	}
}
