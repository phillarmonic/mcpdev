package redact

import (
	"net/http"
	"strings"
)

// Any redacts credentials in maps, slices, headers, argv-shaped strings, and URLs.
// It copies containers; the input is not mutated.
func Any(v any, o Options) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return URLString(x, o)
	case []string:
		return Argv(x, o)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = Any(item, o)
		}
		return out
	case map[string]any:
		return MapAny(x, o)
	case map[string]string:
		return MapString(x, o)
	case http.Header:
		return Header(x, o)
	case map[string][]string:
		return Header(x, o)
	default:
		return v
	}
}

// MapAny redacts sensitive keys and walks nested values.
func MapAny(in map[string]any, o Options) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if Sensitive(k, o.ExtraKeys) {
			out[k] = o.replacement()
			continue
		}
		out[k] = Any(v, o)
	}
	return out
}

// MapString redacts sensitive keys and URL-shaped values.
func MapString(in map[string]string, o Options) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if Sensitive(k, o.ExtraKeys) {
			out[k] = o.replacement()
			continue
		}
		out[k] = URLString(v, o)
	}
	return out
}

// Header redacts authorization, cookie, API-key, token, secret, and password fields.
func Header(h map[string][]string, o Options) http.Header {
	if h == nil {
		return nil
	}
	out := make(http.Header, len(h))
	for k, vs := range h {
		cp := make([]string, len(vs))
		if Sensitive(k, o.ExtraKeys) {
			for i := range vs {
				cp[i] = o.replacement()
			}
		} else {
			copy(cp, vs)
		}
		out[k] = cp
	}
	return out
}

// Argv redacts KEY=VALUE env pairs, --header/--env values, and URL userinfo in command args.
func Argv(args []string, o Options) []string {
	if args == nil {
		return nil
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = redactArg(args, i, a, o)
	}
	return out
}

func redactArg(args []string, i int, a string, o Options) string {
	switch {
	case strings.HasPrefix(a, "--header="):
		return "--header=" + redactHeaderLine(strings.TrimPrefix(a, "--header="), o)
	case strings.HasPrefix(a, "--env="):
		return "--env=" + redactAssign(strings.TrimPrefix(a, "--env="), o)
	case i > 0 && (args[i-1] == "--header" || args[i-1] == "-H"):
		return redactHeaderLine(a, o)
	case i > 0 && args[i-1] == "--env":
		return redactAssign(a, o)
	default:
		if k, v, ok := strings.Cut(a, "="); ok && !strings.HasPrefix(a, "-") {
			if Sensitive(k, o.ExtraKeys) {
				return k + "=" + o.replacement()
			}
			return k + "=" + URLString(v, o)
		}
		return URLString(a, o)
	}
}

func redactHeaderLine(line string, o Options) string {
	name, val, ok := strings.Cut(line, ":")
	if !ok {
		return line
	}
	if Sensitive(strings.TrimSpace(name), o.ExtraKeys) {
		return strings.TrimSpace(name) + ": " + o.replacement()
	}
	return name + ":" + val
}

func redactAssign(pair string, o Options) string {
	k, v, ok := strings.Cut(pair, "=")
	if !ok {
		return pair
	}
	if Sensitive(k, o.ExtraKeys) {
		return k + "=" + o.replacement()
	}
	return k + "=" + URLString(v, o)
}
