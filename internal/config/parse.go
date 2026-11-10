package config

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

type valueKind int

const (
	kindString valueKind = iota
	kindArray
	kindBool
	kindInt
)

type value struct {
	str  string
	arr  []string
	kind valueKind
	n    int
	b    bool
}

func parseAssign(line string) (string, value, error) {
	key, rest, err := splitAssign(line)
	if err != nil {
		return "", value{}, err
	}
	v, err := parseValue(rest)
	return key, v, err
}

func splitAssign(line string) (string, string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", errors.New("expected key = value")
	}
	var key string
	var rest string
	if line[0] == '"' {
		k, rem, err := readQuoted(line)
		if err != nil {
			return "", "", err
		}
		key = k
		rest = strings.TrimSpace(rem)
	} else {
		i := 0
		for i < len(line) && line[i] != '=' && !unicode.IsSpace(rune(line[i])) {
			i++
		}
		key = line[:i]
		rest = strings.TrimSpace(line[i:])
	}
	if key == "" {
		return "", "", errors.New("empty key")
	}
	if !strings.HasPrefix(rest, "=") {
		return "", "", errors.New("expected key = value")
	}
	return key, strings.TrimSpace(rest[1:]), nil
}

func parseValue(rest string) (value, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return value{}, errors.New("missing value")
	}
	switch rest {
	case "true":
		return value{kind: kindBool, b: true}, nil
	case "false":
		return value{kind: kindBool, b: false}, nil
	}
	if rest[0] == '[' {
		arr, err := parseArray(rest)
		return value{kind: kindArray, arr: arr}, err
	}
	if rest[0] == '"' {
		s, err := parseString(rest)
		return value{kind: kindString, str: s}, err
	}
	if isInt(rest) {
		n, err := strconv.Atoi(rest)
		return value{kind: kindInt, n: n}, err
	}
	return value{}, errors.New("values must be quoted strings, arrays, integers, or booleans")
}

func isInt(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func parseArray(s string) ([]string, error) {
	if !strings.HasSuffix(s, "]") {
		return nil, errors.New("unclosed array")
	}
	body := strings.TrimSpace(s[1 : len(s)-1])
	if body == "" {
		return []string{}, nil
	}
	var out []string
	for body != "" {
		body = strings.TrimLeftFunc(body, unicode.IsSpace)
		if body == "" {
			break
		}
		if body[0] != '"' {
			return nil, errors.New("array entries must be quoted strings")
		}
		item, rest, err := readQuoted(body)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
		rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
		if rest == "" {
			break
		}
		if rest[0] != ',' {
			return nil, errors.New("expected comma in array")
		}
		body = rest[1:]
	}
	return out, nil
}

func parseString(s string) (string, error) {
	v, rest, err := readQuoted(s)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(rest) != "" {
		return "", errors.New("trailing data after string")
	}
	return v, nil
}

func readQuoted(s string) (string, string, error) {
	if len(s) == 0 || s[0] != '"' {
		return "", "", errors.New("expected quoted string")
	}
	var b strings.Builder
	esc := false
	for i := 1; i < len(s); i++ {
		c := s[i]
		if esc {
			switch c {
			case '"', '\\':
				b.WriteByte(c)
			default:
				b.WriteByte(c)
			}
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '"' {
			return b.String(), s[i+1:], nil
		}
		b.WriteByte(c)
	}
	return "", "", errors.New("unterminated string")
}

func requireString(v value, key string) (string, error) {
	if v.kind != kindString {
		return "", fmtKeyType(key, "string")
	}
	return v.str, nil
}

func requireArray(v value, key string) ([]string, error) {
	if v.kind != kindArray {
		return nil, fmtKeyType(key, "array")
	}
	return v.arr, nil
}

func requireBool(v value, key string) (bool, error) {
	if v.kind != kindBool {
		return false, fmtKeyType(key, "boolean")
	}
	return v.b, nil
}

func requireInt(v value, key string) (int, error) {
	if v.kind != kindInt {
		return 0, fmtKeyType(key, "integer")
	}
	return v.n, nil
}

func fmtKeyType(key, want string) error {
	return errors.New(key + " must be a " + want)
}
