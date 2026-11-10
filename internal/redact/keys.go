package redact

import "strings"

const DefaultReplacement = "[REDACTED]"

// Options control extra sensitive names and the replacement token.
type Options struct {
	Replacement string
	ExtraKeys   []string
}

func (o Options) replacement() string {
	if o.Replacement == "" {
		return DefaultReplacement
	}
	return o.Replacement
}

var exactSensitive = map[string]struct{}{
	"authorization":      {},
	"proxyauthorization": {},
	"cookie":             {},
	"setcookie":          {},
	"apikey":             {},
	"xapikey":            {},
	"token":              {},
	"accesstoken":        {},
	"refreshtoken":       {},
	"idtoken":            {},
	"authtoken":          {},
	"xauthtoken":         {},
	"secret":             {},
	"clientsecret":       {},
	"password":           {},
	"passwd":             {},
	"pwd":                {},
}

var lastSegmentSensitive = map[string]struct{}{
	"token":         {},
	"secret":        {},
	"password":      {},
	"passwd":        {},
	"pwd":           {},
	"authorization": {},
	"cookie":        {},
	"apikey":        {},
}

// Sensitive reports whether a header, env, query, or map key names a credential.
// Matching is by key identity, not by scanning values, so names like tokenizer stay clear.
func Sensitive(key string, extra []string) bool {
	norm := normalizeKey(key)
	if _, ok := exactSensitive[norm]; ok {
		return true
	}
	for _, e := range extra {
		if normalizeKey(e) == norm && norm != "" {
			return true
		}
	}
	segs := splitKey(key)
	if len(segs) == 0 {
		return false
	}
	last := segs[len(segs)-1]
	if _, ok := lastSegmentSensitive[last]; ok {
		return true
	}
	if len(segs) >= 2 && segs[len(segs)-2] == "api" && last == "key" {
		return true
	}
	return false
}

func normalizeKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + ('a' - 'A'))
		case c == '-' || c == '_' || c == '.' || c == ' ':
			continue
		default:
			if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func splitKey(key string) []string {
	var segs []string
	start := 0
	flush := func(end int) {
		if end <= start {
			start = end + 1
			return
		}
		seg := strings.ToLower(key[start:end])
		segs = append(segs, seg)
		start = end + 1
	}
	for i := 0; i < len(key); i++ {
		switch key[i] {
		case '-', '_', '.', '/', ' ':
			flush(i)
		}
	}
	flush(len(key))
	return segs
}
