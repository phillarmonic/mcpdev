package modern

import (
	"encoding/base64"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/protocol/modern"
)

const (
	headerB64Prefix = "=?base64?"
	headerB64Suffix = "?="
)

// EncodeHeaderValue applies the 2026-07-28 Base64 sentinel when a value is not
// a safe ASCII HTTP field value (or already looks like the sentinel).
func EncodeHeaderValue(s string) string {
	if needsBase64Header(s) {
		return headerB64Prefix + base64.StdEncoding.EncodeToString([]byte(s)) + headerB64Suffix
	}
	return s
}

func needsBase64Header(s string) bool {
	if strings.HasPrefix(s, headerB64Prefix) && strings.HasSuffix(s, headerB64Suffix) {
		return true
	}
	if len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		return true
	}
	for _, r := range s {
		if r == 0x09 {
			continue
		}
		if r < 0x21 || r > 0x7E {
			return true
		}
	}
	return false
}

type rpcEnvelope struct {
	Method string            `json:"method"`
	Params jsonv1.RawMessage `json:"params"`
}

type namedParams struct {
	Meta map[string]any `json:"_meta"`
	Name string         `json:"name"`
	URI  string         `json:"uri"`
}

func routingFromPayload(payload []byte) (method, name string) {
	var env rpcEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return "", ""
	}
	method = env.Method
	var p namedParams
	if len(env.Params) > 0 {
		_ = json.Unmarshal(env.Params, &p)
	}
	switch method {
	case "resources/read":
		return method, p.URI
	case "tools/call", "prompts/get":
		return method, p.Name
	default:
		return method, ""
	}
}

func versionFromPayload(payload []byte) string {
	var env rpcEnvelope
	if err := json.Unmarshal(payload, &env); err != nil || len(env.Params) == 0 {
		return ProtocolVersion
	}
	var p namedParams
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return ProtocolVersion
	}
	v, _ := p.Meta[modern.MetaProtocolVersion].(string)
	if v == "" {
		return ProtocolVersion
	}
	return v
}

func methodNeedsName(method string) bool {
	switch method {
	case "tools/call", "prompts/get", "resources/read":
		return true
	default:
		return false
	}
}
