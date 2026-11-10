package trace

import (
	"encoding/json/v2"
	"strconv"
	"unicode/utf8"

	"github.com/phillarmonic/mcpdev/internal/redact"
)

func preview(raw []byte, extra []string, limit int) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	o := redact.Options{ExtraKeys: extra}
	text := preparePayload(raw, o)
	return boundString(text, limit)
}

func preparePayload(raw []byte, o redact.Options) string {
	var v any
	if err := json.Unmarshal(raw, &v); err == nil {
		redacted := redact.Any(v, o)
		b, err := json.Marshal(redacted, json.Deterministic(true))
		if err == nil {
			return string(b)
		}
	}
	return redact.Text(string(raw), o)
}

func boundString(s string, n int) (string, bool) {
	if n < 0 {
		return s, false
	}
	if n == 0 {
		n = DefaultPreviewLimit
	}
	if len(s) <= n {
		return s, false
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s, true
}

func inspectRPC(raw []byte) (method, id string) {
	var env struct {
		ID     any    `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", ""
	}
	return env.Method, rpcID(env.ID)
}

func rpcID(id any) string {
	switch v := id.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		b, err := json.Marshal(v, json.Deterministic(true))
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// FlattenHeader keeps the first value of each HTTP header name.
func FlattenHeader(h map[string][]string) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, vs := range h {
		if len(vs) == 0 {
			continue
		}
		out[k] = vs[0]
	}
	return out
}

func redactHeaders(h map[string]string, extra []string) map[string]string {
	if len(h) == 0 {
		return nil
	}
	return redact.MapString(h, redact.Options{ExtraKeys: extra})
}
