package testkit

import (
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
)

type metaEnvelope struct {
	Meta map[string]any `json:"_meta"`
}

func requireModernMeta(req Request) *Response {
	var env metaEnvelope
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &env); err != nil {
			resp := rpcError(req.ID, codeInvalidParams, "invalid params", nil)
			return &resp
		}
	}
	version, _ := env.Meta["io.modelcontextprotocol/protocolVersion"].(string)
	if version == "" {
		resp := rpcError(req.ID, codeInvalidParams, "missing io.modelcontextprotocol/protocolVersion", nil)
		return &resp
	}
	if version != string(Era20260728) {
		resp := rpcError(req.ID, codeUnsupportedVer, "UnsupportedProtocolVersionError", map[string]any{
			"supported": []string{string(Era20260728)},
		})
		return &resp
	}
	if _, ok := env.Meta["io.modelcontextprotocol/clientInfo"]; !ok {
		resp := rpcError(req.ID, codeInvalidParams, "missing io.modelcontextprotocol/clientInfo", nil)
		return &resp
	}
	return nil
}

func modernMetaVersion(req Request) string {
	var env metaEnvelope
	if len(req.Params) == 0 {
		return ""
	}
	_ = json.Unmarshal(req.Params, &env)
	version, _ := env.Meta["io.modelcontextprotocol/protocolVersion"].(string)
	return version
}

type namedParams struct {
	Arguments map[string]any `json:"arguments"`
	Meta      map[string]any `json:"_meta,omitzero"`
	Name      string         `json:"name"`
	URI       string         `json:"uri"`
}

func decodeNamed(req Request) (namedParams, error) {
	var p namedParams
	if len(req.Params) == 0 {
		return p, errors.New("missing params")
	}
	return p, json.Unmarshal(req.Params, &p)
}

func handleCall(d Descriptor, state *SessionState, req Request) Response {
	if resp, ok := requireReady(d, state, req); !ok {
		return resp
	}
	if d.Era == Era20260728 {
		if err := requireModernMeta(req); err != nil {
			return *err
		}
	}
	p, err := decodeNamed(req)
	if err != nil {
		return rpcError(req.ID, codeInvalidParams, "invalid tools/call params", nil)
	}
	text, _ := p.Arguments["text"].(string)
	switch p.Name {
	case "echo":
		return rpcResult(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
		})
	case "fail":
		return rpcResult(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": "failed"}},
			"isError": true,
		})
	default:
		return rpcError(req.ID, codeInvalidParams, fmt.Sprintf("unknown tool %q", p.Name), nil)
	}
}

func handleGetPrompt(d Descriptor, state *SessionState, req Request) Response {
	if resp, ok := requireReady(d, state, req); !ok {
		return resp
	}
	if d.Era == Era20260728 {
		if err := requireModernMeta(req); err != nil {
			return *err
		}
	}
	p, err := decodeNamed(req)
	if err != nil {
		return rpcError(req.ID, codeInvalidParams, "invalid prompts/get params", nil)
	}
	if p.Name != "greet" {
		return rpcError(req.ID, codeInvalidParams, fmt.Sprintf("unknown prompt %q", p.Name), nil)
	}
	who, _ := p.Arguments["name"].(string)
	if who == "" {
		who = "world"
	}
	return rpcResult(req.ID, map[string]any{
		"description": "Stable greeting prompt",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": map[string]any{
					"type": "text",
					"text": "greet " + who,
				},
			},
		},
	})
}

func handleRead(d Descriptor, state *SessionState, req Request) Response {
	if resp, ok := requireReady(d, state, req); !ok {
		return resp
	}
	if d.Era == Era20260728 {
		if err := requireModernMeta(req); err != nil {
			return *err
		}
	}
	p, err := decodeNamed(req)
	if err != nil {
		return rpcError(req.ID, codeInvalidParams, "invalid resources/read params", nil)
	}
	switch {
	case p.URI == "fixture://stable/note":
		return rpcResult(req.ID, map[string]any{
			"contents": []map[string]any{
				{
					"uri":      p.URI,
					"mimeType": "text/plain",
					"text":     "stable-note",
				},
			},
		})
	case p.URI == "fixture://stable/blob":
		return rpcResult(req.ID, map[string]any{
			"contents": []map[string]any{
				{
					"uri":      p.URI,
					"mimeType": "application/octet-stream",
					"blob":     base64.StdEncoding.EncodeToString([]byte{0x00, 0xff, 0x80}),
				},
			},
		})
	case strings.HasPrefix(p.URI, "fixture://items/"):
		id := strings.TrimPrefix(p.URI, "fixture://items/")
		return rpcResult(req.ID, map[string]any{
			"contents": []map[string]any{
				{
					"uri":      p.URI,
					"mimeType": "text/plain",
					"text":     "item:" + id,
				},
			},
		})
	default:
		return rpcError(req.ID, codeInvalidParams, fmt.Sprintf("unknown resource %q", p.URI), nil)
	}
}
