package render

import (
	"strings"

	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

func encodeTarget(t port.Target) targetJSON {
	o := redact.Options{ExtraKeys: t.Sensitive}
	out := targetJSON{Kind: t.Kind, Profile: t.Profile, Command: redact.Argv(t.Command, o)}
	if t.URL != "" {
		out.URL = redact.URLString(t.URL, o)
	}
	if t.Kind == "" {
		switch {
		case len(t.Command) > 0:
			out.Kind = "stdio"
		case t.URL != "":
			out.Kind = "http"
		}
	}
	if len(t.Headers) > 0 {
		out.Headers = redact.MapString(t.Headers, o)
	}
	return out
}

func humanTarget(t port.Target) string {
	switch {
	case t.Profile != "":
		return "profile " + t.Profile
	case len(t.Command) > 0:
		return "stdio " + strings.Join(t.Command, " ")
	case t.URL != "":
		return "http " + redact.URLString(t.URL, redact.Options{ExtraKeys: t.Sensitive})
	default:
		return t.Kind
	}
}
