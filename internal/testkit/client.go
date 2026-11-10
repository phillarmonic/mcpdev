package testkit

import "maps"

func modernParams(extra map[string]any) []byte {
	meta := map[string]any{
		"io.modelcontextprotocol/protocolVersion": string(Era20260728),
		"io.modelcontextprotocol/clientInfo": map[string]any{
			"name":    "mcpdev-test",
			"version": "0",
		},
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}
	params := map[string]any{"_meta": meta}
	maps.Copy(params, extra)
	return mustMarshal(params)
}

func initParams() []byte {
	return mustMarshal(map[string]any{
		"protocolVersion": string(Era20251125),
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mcpdev-test",
			"version": "0",
		},
	})
}

func handshakeStdio(lab *StdioLab) error {
	resp, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "initialize", Params: initParams()})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	if err := writeJSONLine(lab.ClientIn, Request{JSONRPC: jsonRPCVersion, Method: "notifications/initialized"}); err != nil {
		return err
	}
	if lab.Desc.NotifyListChanged {
		var note map[string]any
		return readJSONLine(lab.outScan, &note)
	}
	return nil
}
