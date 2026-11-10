package diff

import "strings"

func pointer(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteByte('/')
		b.WriteString(escapePointer(part))
	}
	return b.String()
}

func escapePointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	return strings.ReplaceAll(s, "/", "~1")
}
