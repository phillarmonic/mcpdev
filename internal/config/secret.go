package config

import (
	"slices"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/redact"
)

func looksLikePEM(s string) bool {
	upper := strings.ToUpper(s)
	return strings.Contains(upper, "BEGIN ") && (strings.Contains(upper, "PRIVATE KEY") ||
		strings.Contains(upper, "CERTIFICATE") ||
		strings.Contains(upper, "PUBLIC KEY"))
}

func looksLikeInterpolation(s string) bool {
	return strings.Contains(s, "${") && strings.Contains(s, "}")
}

// ExtraKeys are names renderers must treat as credentials in addition to built-ins.
func ExtraKeys(extra []string, maps ...map[string]string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	for _, k := range extra {
		add(k)
	}
	for _, m := range maps {
		for k := range m {
			if redact.Sensitive(k, extra) {
				add(k)
			}
		}
	}
	slices.Sort(out)
	return out
}

// RedactOptions is the shared redaction policy for this merged configuration.
func (s Settings) RedactOptions() redact.Options {
	extra := append([]string{}, s.Sensitive...)
	for _, p := range s.Profiles {
		extra = ExtraKeys(extra, p.Headers, p.HeaderEnv, p.Env)
	}
	return redact.Options{ExtraKeys: extra}
}
