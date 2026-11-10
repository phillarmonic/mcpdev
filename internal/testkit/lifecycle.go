package testkit

import "encoding/json/v2"

type initializeParams struct {
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      map[string]any `json:"clientInfo"`
	ProtocolVersion string         `json:"protocolVersion"`
}

func handleInitialize(d Descriptor, state *SessionState, req Request) Response {
	if d.Era != Era20251125 {
		return rpcError(req.ID, codeMethodNotFound, "method not found: initialize", nil)
	}
	if d.Malformed == MalformedUnsupportedVersion {
		return rpcResult(req.ID, map[string]any{
			"protocolVersion": "1999-01-01",
			"capabilities":    map[string]any{},
			"serverInfo":      map[string]any{"name": "mcpdev-fixture", "version": "0"},
		})
	}
	if d.Malformed == MalformedCapabilityObject {
		return rpcResult(req.ID, map[string]any{
			"protocolVersion": string(d.Era),
			"capabilities":    []any{"tools"},
			"serverInfo":      map[string]any{"name": "mcpdev-fixture", "version": "0"},
		})
	}
	var p initializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return rpcError(req.ID, codeInvalidParams, "invalid initialize params", nil)
		}
	}
	negotiated := string(d.Era)
	if p.ProtocolVersion != "" && p.ProtocolVersion != negotiated {
		negotiated = string(d.Era)
	}
	if state != nil {
		if d.Session.MintID {
			state.setSessionID(d.sessionID())
		}
	}
	caps := map[string]any{}
	if d.Capabilities.Tools {
		toolCap := map[string]any{}
		if d.Capabilities.ListChanged {
			toolCap["listChanged"] = true
		}
		caps["tools"] = toolCap
	}
	if d.Capabilities.Prompts {
		promptCap := map[string]any{}
		if d.Capabilities.ListChanged {
			promptCap["listChanged"] = true
		}
		caps["prompts"] = promptCap
	}
	if d.Capabilities.Resources {
		resCap := map[string]any{}
		if d.Capabilities.ListChanged {
			resCap["listChanged"] = true
		}
		if d.Capabilities.ResourceSubscribe {
			resCap["subscribe"] = true
		}
		caps["resources"] = resCap
	}
	result := map[string]any{
		"protocolVersion": negotiated,
		"capabilities":    caps,
		"serverInfo": map[string]any{
			"name":    "mcpdev-fixture",
			"title":   "Fixture Lab",
			"version": "0",
		},
	}
	if d.Instructions != "" {
		result["instructions"] = d.Instructions
	}
	return rpcResult(req.ID, result)
}

func handleInitialized(d Descriptor, state *SessionState, req Request) Response {
	if d.Era != Era20251125 {
		return rpcError(req.ID, codeMethodNotFound, "method not found: notifications/initialized", nil)
	}
	if state != nil {
		state.markInitialized()
	}
	return Response{}
}

func handleDiscover(d Descriptor, state *SessionState, req Request) Response {
	if d.Era != Era20260728 {
		return rpcError(req.ID, codeMethodNotFound, "method not found: server/discover", nil)
	}
	if err := requireModernMeta(req); err != nil {
		return *err
	}
	if d.Malformed == MalformedMissingClientCap {
		return rpcError(req.ID, codeMissingCap, "MissingRequiredClientCapabilityError", map[string]any{
			"requiredCapabilities": map[string]any{"sampling": map[string]any{}},
		})
	}
	if d.Malformed == MalformedModernUnsupported && state.nextDiscover() == 1 {
		return rpcError(req.ID, codeUnsupportedVer, "UnsupportedProtocolVersionError", map[string]any{
			"supported": []string{string(Era20260728)},
			"requested": modernMetaVersion(req),
		})
	}
	caps := map[string]any{}
	if d.Capabilities.Tools {
		caps["tools"] = map[string]any{}
	}
	if d.Capabilities.Prompts {
		caps["prompts"] = map[string]any{}
	}
	if d.Capabilities.Resources {
		caps["resources"] = map[string]any{}
	}
	serverInfo := map[string]any{
		"name":    "mcpdev-fixture",
		"title":   "Fixture Lab",
		"version": "0",
	}
	return rpcResult(req.ID, map[string]any{
		"supportedVersions": []string{string(Era20260728)},
		"capabilities":      caps,
		"serverInfo":        serverInfo,
		"_meta": map[string]any{
			"io.modelcontextprotocol/serverInfo": serverInfo,
		},
	})
}
