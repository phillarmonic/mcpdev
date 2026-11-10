package invoke

import (
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// ExpandURITemplate expands RFC 6570 level-1 `{var}` and `{+var}` placeholders.
func ExpandURITemplate(template string, vars map[string]string) (string, error) {
	if template == "" {
		return "", errclass.Usage{Msg: "resource URI or --template is required"}
	}
	var b strings.Builder
	i := 0
	for i < len(template) {
		if template[i] != '{' {
			b.WriteByte(template[i])
			i++
			continue
		}
		end := strings.IndexByte(template[i:], '}')
		if end < 0 {
			return "", errclass.Usage{Msg: "unclosed URI template expression"}
		}
		expr := template[i+1 : i+end]
		i += end + 1
		reserved := false
		if strings.HasPrefix(expr, "+") {
			reserved = true
			expr = expr[1:]
		}
		if expr == "" || strings.ContainsAny(expr, ":,.*") {
			return "", errclass.Usage{Msg: "unsupported URI template expression"}
		}
		val := vars[expr]
		if reserved {
			b.WriteString(encodeReserved(val))
		} else {
			b.WriteString(encodeSimple(val))
		}
	}
	return b.String(), nil
}

func encodeSimple(s string) string {
	return percentEncode(s, false)
}

func encodeReserved(s string) string {
	return percentEncode(s, true)
}

func percentEncode(s string, allowReserved bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) || (allowReserved && isReserved(c)) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0f])
	}
	return b.String()
}

const hex = "0123456789ABCDEF"

func isUnreserved(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_' || c == '~'
}

func isReserved(c byte) bool {
	switch c {
	case ':', '/', '?', '#', '[', ']', '@', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', '%':
		return true
	default:
		return false
	}
}
