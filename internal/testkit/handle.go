package testkit

import (
	"context"
	"encoding/json/v2"
	"time"
)

func handle(ctx context.Context, d Descriptor, state *SessionState, req Request) Response {
	if err := waitDelay(ctx, d.Delay); err != nil {
		return rpcError(req.ID, codeInvalidRequest, "cancelled", nil)
	}
	switch req.Method {
	case "initialize":
		return handleInitialize(d, state, req)
	case "notifications/initialized":
		return handleInitialized(d, state, req)
	case "server/discover":
		return handleDiscover(d, state, req)
	case "tools/list":
		return handleList(d, state, req, "tools")
	case "prompts/list":
		return handleList(d, state, req, "prompts")
	case "resources/list":
		return handleList(d, state, req, "resources")
	case "resources/templates/list":
		return handleList(d, state, req, "templates")
	case "tools/call":
		return handleCall(d, state, req)
	case "prompts/get":
		return handleGetPrompt(d, state, req)
	case "resources/read":
		return handleRead(d, state, req)
	case "notifications/cancelled":
		return Response{}
	default:
		return rpcError(req.ID, codeMethodNotFound, "method not found: "+req.Method, nil)
	}
}

func waitDelay(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func requireReady(d Descriptor, state *SessionState, req Request) (Response, bool) {
	if d.Era != Era20251125 || !d.RequireHandshake {
		return Response{}, true
	}
	if state.ready(true) {
		return Response{}, true
	}
	return rpcError(req.ID, codeInvalidRequest, "session not initialized", nil), false
}

type listParams struct {
	Meta   map[string]any `json:"_meta,omitzero"`
	Cursor string         `json:"cursor,omitzero"`
}

func decodeListParams(req Request) (listParams, error) {
	var p listParams
	if len(req.Params) == 0 {
		return p, nil
	}
	return p, json.Unmarshal(req.Params, &p)
}

func handleList(d Descriptor, state *SessionState, req Request, kind string) Response {
	if resp, ok := requireReady(d, state, req); !ok {
		return resp
	}
	if d.Era == Era20260728 {
		if err := requireModernMeta(req); err != nil {
			return *err
		}
	}
	p, err := decodeListParams(req)
	if err != nil {
		return rpcError(req.ID, codeInvalidParams, "invalid list params", nil)
	}
	var next string
	var items any
	switch kind {
	case "tools":
		var page []Tool
		page, next, err = paginate(stableTools(d.ExtraTools), p.Cursor, d.pageSize())
		items = page
	case "prompts":
		var page []Prompt
		page, next, err = paginate(stablePrompts(), p.Cursor, d.pageSize())
		items = page
	case "resources":
		var page []Resource
		page, next, err = paginate(stableResources(), p.Cursor, d.pageSize())
		items = page
	case "templates":
		var page []ResourceTemplate
		page, next, err = paginate(stableTemplates(), p.Cursor, d.pageSize())
		items = page
	default:
		return rpcError(req.ID, codeInvalidParams, "unknown list kind", nil)
	}
	if err != nil {
		return rpcError(req.ID, codeInvalidParams, err.Error(), nil)
	}
	out := map[string]any{kind: items}
	if kind == "templates" {
		out = map[string]any{"resourceTemplates": items}
	}
	if next != "" {
		out["nextCursor"] = next
	}
	return rpcResult(req.ID, out)
}
