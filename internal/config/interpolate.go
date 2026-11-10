package config

import (
	"strings"
	"unicode"
)

func interpolate(s, field string, env map[string]string) (string, error) {
	if !strings.Contains(s, "$") {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '$' {
			b.WriteByte('$')
			i += 2
			continue
		}
		if i+1 >= len(s) || s[i+1] != '{' {
			return "", configErr("", field, "use ${NAME}, ${NAME:?}, or ${NAME:-default} ($$ for a literal $)")
		}
		end := strings.IndexByte(s[i+2:], '}')
		if end < 0 {
			return "", configErr("", field, "unclosed ${...} interpolation")
		}
		body := s[i+2 : i+2+end]
		rep, err := expandBody(body, field, env)
		if err != nil {
			return "", err
		}
		b.WriteString(rep)
		i += 3 + end
	}
	return b.String(), nil
}

func expandBody(body, field string, env map[string]string) (string, error) {
	name := body
	required := false
	hasDefault := false
	defaultVal := ""
	switch {
	case strings.HasSuffix(body, ":?"):
		name = strings.TrimSuffix(body, ":?")
		required = true
	case strings.Contains(body, ":-"):
		parts := strings.SplitN(body, ":-", 2)
		name, defaultVal = parts[0], parts[1]
		hasDefault = true
	}
	if !validEnvName(name) {
		return "", configErr("", field, "invalid environment name "+name)
	}
	val, ok := env[name]
	if ok && val != "" {
		return val, nil
	}
	if required {
		return "", configErr("", field, "missing required environment variable "+name)
	}
	if hasDefault {
		return defaultVal, nil
	}
	if !ok || val == "" {
		return "", configErr("", field, "missing environment variable "+name)
	}
	return val, nil
}

func validEnvName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case unicode.IsLetter(r):
		case i > 0 && unicode.IsDigit(r):
		default:
			return false
		}
	}
	return true
}

func interpolateMap(m map[string]string, fieldPrefix string, env map[string]string) (map[string]string, error) {
	if len(m) == 0 {
		return m, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		s, err := interpolate(v, fieldPrefix+"."+k, env)
		if err != nil {
			return nil, err
		}
		out[k] = s
	}
	return out, nil
}
