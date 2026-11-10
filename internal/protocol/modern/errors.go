package modern

import (
	"bytes"
	"encoding/json/v2"
	"net/http"
	"strings"
)

// JSON-RPC codes for recognized 2026-07-28 validation errors.
const (
	CodeHeaderMismatch     = -32020
	CodeMissingCapability  = -32021
	CodeUnsupportedVersion = -32022
)

// Kind names a classified modern or fallback-shaped error.
type Kind string

const (
	KindHeaderMismatch     Kind = "HeaderMismatch"
	KindMissingCapability  Kind = "MissingRequiredClientCapabilityError"
	KindUnsupportedVersion Kind = "UnsupportedProtocolVersionError"
	KindEmptyBody          Kind = "empty_body"
	KindUnrecognized       Kind = "unrecognized"
)

// Classified is a discover or HTTP 400 outcome used to decide fallback.
type Classified struct {
	Kind      Kind
	Requested string
	Supported []string
	Code      int
}

// StayModern reports a recognized 2026-07-28 validation error. Those must not fall back to initialize.
func (c Classified) StayModern() bool {
	switch c.Kind {
	case KindHeaderMismatch, KindMissingCapability, KindUnsupportedVersion:
		return true
	default:
		return false
	}
}

// ClassifyRPC classifies a JSON-RPC error from server/discover or a modern HTTP body.
func ClassifyRPC(code int, data any) Classified {
	switch code {
	case CodeHeaderMismatch:
		return Classified{Kind: KindHeaderMismatch, Code: code}
	case CodeMissingCapability:
		return Classified{Kind: KindMissingCapability, Code: code}
	case CodeUnsupportedVersion:
		out := Classified{Kind: KindUnsupportedVersion, Code: code}
		out.Supported, out.Requested = supportedFromData(data)
		return out
	default:
		return Classified{Kind: KindUnrecognized, Code: code}
	}
}

// ClassifyHTTP inspects an HTTP 400 body. Empty, non-JSON, or unrecognized JSON-RPC may fall back.
// Recognized modern errors stay in the 2026-07-28 era.
func ClassifyHTTP(status int, body []byte) Classified {
	if status != http.StatusBadRequest {
		return Classified{Kind: KindUnrecognized, Code: status}
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return Classified{Kind: KindEmptyBody}
	}
	var env struct {
		Error *struct {
			Data    any    `json:"data"`
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(trimmed, &env); err != nil || env.Error == nil {
		return Classified{Kind: KindUnrecognized}
	}
	return ClassifyRPC(env.Error.Code, env.Error.Data)
}

func supportedFromData(data any) ([]string, string) {
	m, ok := data.(map[string]any)
	if !ok {
		return nil, ""
	}
	var supported []string
	switch raw := m["supported"].(type) {
	case []string:
		supported = slicesClone(raw)
	case []any:
		for _, item := range raw {
			s, ok := item.(string)
			if ok {
				supported = append(supported, s)
			}
		}
	}
	requested, _ := m["requested"].(string)
	return supported, requested
}

func slicesClone(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// ReasonFallback is the diagnostic string when ClassifyHTTP or ClassifyRPC allows initialize.
func ReasonFallback(c Classified) string {
	switch c.Kind {
	case KindEmptyBody:
		return "empty_http_body"
	case KindUnrecognized:
		if c.Code == -32601 {
			return "method_not_found"
		}
		return "unrecognized_error"
	default:
		return strings.ToLower(string(c.Kind))
	}
}
