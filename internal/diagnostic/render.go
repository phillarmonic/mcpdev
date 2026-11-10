package diagnostic

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/phillarmonic/mcpdev/internal/redact"
)

type envelope struct {
	Details      map[string]any `json:"details,omitempty"`
	Message      string         `json:"message,omitempty"`
	Code         string         `json:"code"`
	ChildStderr  string         `json:"child_stderr,omitempty"`
	Era          string         `json:"era,omitempty"`
	Cause        string         `json:"cause,omitempty"`
	Operation    string         `json:"operation,omitempty"`
	Severity     string         `json:"severity"`
	Transport    string         `json:"transport,omitempty"`
	ElapsedMS    int64          `json:"elapsed_ms,omitempty"`
	PreviewLimit int            `json:"preview_limit"`
	Retryable    bool           `json:"retryable"`
	Truncated    bool           `json:"truncated"`
}

func (r Record) previewLimit() int {
	if r.PreviewLimit <= 0 {
		return DefaultPreviewLimit
	}
	return r.PreviewLimit
}

func (r Record) prepare(o redact.Options) (envelope, bool) {
	limit := r.previewLimit()
	truncated := false
	msg, t := boundString(redact.Text(r.Message, o), limit)
	truncated = truncated || t
	var cause string
	if r.Cause != nil {
		cause, t = boundString(redact.Text(r.Cause.Error(), o), limit)
		truncated = truncated || t
	}
	stderr, t := boundString(redact.Text(r.ChildStderr, o), limit)
	truncated = truncated || t
	var details map[string]any
	if r.Details != nil {
		details = redact.MapAny(r.Details, o)
		var dt bool
		details, dt = boundMap(details, limit)
		truncated = truncated || dt
	}
	return envelope{
		Cause:        cause,
		ChildStderr:  stderr,
		Code:         string(r.Code),
		Details:      details,
		Era:          string(r.Era),
		ElapsedMS:    r.Elapsed.Milliseconds(),
		Message:      msg,
		Operation:    r.Operation,
		PreviewLimit: limit,
		Retryable:    r.Retryable,
		Severity:     string(r.Severity),
		Transport:    r.Transport,
		Truncated:    truncated,
	}, truncated
}

// MarshalJSON renders a redacted, bounded diagnostic object with stable keys.
func (r Record) MarshalJSON() ([]byte, error) {
	return r.MarshalJSONRedacted(redact.Options{})
}

// MarshalJSONRedacted applies extra sensitive keys before encoding.
func (r Record) MarshalJSONRedacted(o redact.Options) ([]byte, error) {
	env, _ := r.prepare(o)
	return json.Marshal(env, json.Deterministic(true))
}

// WriteJSON writes one JSON document. Truncation happens on string fields, never by slicing JSON.
func WriteJSON(w io.Writer, r Record, o redact.Options) error {
	b, err := r.MarshalJSONRedacted(o)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// WriteHuman writes a single diagnostic block for stderr.
func WriteHuman(w io.Writer, r Record, o redact.Options) error {
	env, _ := r.prepare(o)
	sev := env.Severity
	if sev == "" {
		sev = string(SeverityError)
	}
	line := fmt.Sprintf("mcpdev: %s %s", sev, env.Code)
	if env.Message != "" {
		line += ": " + env.Message
	}
	if _, err := io.WriteString(w, line+"\n"); err != nil {
		return err
	}
	var meta []string
	if env.Operation != "" {
		meta = append(meta, "operation="+env.Operation)
	}
	if env.Transport != "" {
		meta = append(meta, "transport="+env.Transport)
	}
	if env.Era != "" {
		meta = append(meta, "era="+env.Era)
	}
	if env.ElapsedMS != 0 {
		meta = append(meta, fmt.Sprintf("elapsed_ms=%d", env.ElapsedMS))
	}
	meta = append(meta, fmt.Sprintf("retryable=%t", env.Retryable))
	if env.Truncated {
		meta = append(meta, "truncated=true")
	}
	if _, err := io.WriteString(w, "  "+strings.Join(meta, " ")+"\n"); err != nil {
		return err
	}
	if env.Cause != "" {
		if _, err := io.WriteString(w, "  cause: "+env.Cause+"\n"); err != nil {
			return err
		}
	}
	if env.ChildStderr != "" {
		if _, err := io.WriteString(w, "  child_stderr: "+env.ChildStderr+"\n"); err != nil {
			return err
		}
	}
	return nil
}

func boundString(s string, n int) (string, bool) {
	if n <= 0 {
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

func boundMap(m map[string]any, n int) (map[string]any, bool) {
	if m == nil {
		return nil, false
	}
	out := make(map[string]any, len(m))
	trunc := false
	for k, v := range m {
		nv, t := boundValue(v, n)
		out[k] = nv
		trunc = trunc || t
	}
	return out, trunc
}

func boundValue(v any, n int) (any, bool) {
	switch x := v.(type) {
	case string:
		return boundString(x, n)
	case []string:
		out := make([]string, len(x))
		trunc := false
		for i, s := range x {
			ns, t := boundString(s, n)
			out[i] = ns
			trunc = trunc || t
		}
		return out, trunc
	case []any:
		out := make([]any, len(x))
		trunc := false
		for i, item := range x {
			nv, t := boundValue(item, n)
			out[i] = nv
			trunc = trunc || t
		}
		return out, trunc
	case map[string]any:
		return boundMap(x, n)
	default:
		return v, false
	}
}
