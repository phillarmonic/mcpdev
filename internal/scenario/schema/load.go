package schema

import (
	"bytes"
	jsonv2 "encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"gopkg.in/yaml.v3"
)

// LoadFile reads path as YAML or JSON and validates it. It does not connect.
func LoadFile(path string) (Suite, error) {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return Suite{}, errclass.Usage{Msg: "scenario: path is required"}
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return Suite{}, errclass.Usage{Msg: "scenario: read " + clean, Err: err}
	}
	s, err := Load(data, clean)
	if err != nil {
		return Suite{}, err
	}
	return s, nil
}

// Load parses YAML or JSON bytes. source is used in error messages (file path or "-").
func Load(data []byte, source string) (Suite, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return Suite{}, usageErr(source, Issue{Path: "/", Message: "document is empty"})
	}
	tree, err := decodeTree(trimmed)
	if err != nil {
		return Suite{}, errclass.Usage{Msg: fmt.Sprintf("scenario: %s: parse", label(source)), Err: err}
	}
	var issues []Issue
	walkForbidden(pointer(""), tree, &issues)
	walkUnknown(pointer(""), tree, allowedSuite, &issues)
	suite, convErr := decodeSuite(tree)
	if convErr != nil {
		issues = append(issues, Issue{Path: "/", Message: convErr.Error()})
	}
	if convErr == nil {
		issues = append(issues, validateSuite(suite)...)
	}
	if len(issues) > 0 {
		return Suite{}, usageErr(source, issues...)
	}
	return suite, nil
}

func decodeTree(data []byte) (any, error) {
	if isJSON(data) {
		var v any
		if err := jsonv2.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	}
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return normalizeYAML(v), nil
}

func isJSON(data []byte) bool {
	switch data[0] {
	case '{', '[':
		return true
	}
	return false
}

func normalizeYAML(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = normalizeYAML(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[fmt.Sprint(k)] = normalizeYAML(val)
		}
		return out
	case []any:
		for i, val := range t {
			t[i] = normalizeYAML(val)
		}
		return t
	default:
		return v
	}
}

func decodeSuite(tree any) (Suite, error) {
	b, err := jsonv2.Marshal(tree)
	if err != nil {
		return Suite{}, fmt.Errorf("encode tree: %w", err)
	}
	var s Suite
	if err := jsonv2.Unmarshal(b, &s); err != nil {
		return Suite{}, fmt.Errorf("typed decode: %w", err)
	}
	return s, nil
}

func usageErr(source string, issues ...Issue) error {
	return errclass.Usage{Err: Error{File: label(source), Issues: issues}}
}

func label(source string) string {
	if source == "" {
		return "scenario"
	}
	return filepath.ToSlash(source)
}

func pointer(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteByte('/')
		b.WriteString(escapePointer(p))
	}
	if b.Len() == 0 {
		return ""
	}
	return b.String()
}

func joinPointer(base, next string) string {
	if next == "" {
		return base
	}
	return base + "/" + escapePointer(next)
}

func escapePointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	return strings.ReplaceAll(s, "/", "~1")
}
