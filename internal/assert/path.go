package assert

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Select returns every match for a JSON Path ($.a[0].b or $.a[*].c) or JSON Pointer (/a/0/b).
func Select(doc any, path string) ([]any, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "$" {
		return []any{doc}, nil
	}
	if strings.HasPrefix(path, "/") {
		v, err := pointer(doc, path)
		if err != nil {
			return nil, err
		}
		return []any{v}, nil
	}
	if !strings.HasPrefix(path, "$") {
		return nil, errors.New("path must be JSON Path or JSON Pointer")
	}
	return jsonPath(doc, path[1:])
}

func pointer(doc any, path string) (any, error) {
	if path == "/" || path == "" {
		return doc, nil
	}
	cur := doc
	for part := range strings.SplitSeq(strings.TrimPrefix(path, "/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		next, ok := child(cur, part)
		if !ok {
			return nil, fmt.Errorf("missing %s", path)
		}
		cur = next
	}
	return cur, nil
}

func jsonPath(doc any, rest string) ([]any, error) {
	nodes := []any{doc}
	i := 0
	for i < len(rest) {
		switch rest[i] {
		case '.':
			i++
			start := i
			for i < len(rest) && rest[i] != '.' && rest[i] != '[' {
				i++
			}
			name := rest[start:i]
			if name == "" {
				return nil, errors.New("empty JSON Path segment")
			}
			nodes = mapNodes(nodes, name)
		case '[':
			end := strings.IndexByte(rest[i:], ']')
			if end < 0 {
				return nil, errors.New("unclosed JSON Path index")
			}
			idx := rest[i+1 : i+end]
			i += end + 1
			if idx == "*" {
				nodes = flattenArrays(nodes)
				continue
			}
			n, err := strconv.Atoi(idx)
			if err != nil {
				return nil, fmt.Errorf("JSON Path index %q", idx)
			}
			nodes = indexNodes(nodes, n)
		default:
			return nil, fmt.Errorf("unsupported JSON Path at %q", rest[i:])
		}
	}
	return nodes, nil
}

func mapNodes(nodes []any, name string) []any {
	out := make([]any, 0, len(nodes))
	for _, n := range nodes {
		if v, ok := child(n, name); ok {
			out = append(out, v)
		}
	}
	return out
}

func indexNodes(nodes []any, n int) []any {
	out := make([]any, 0, len(nodes))
	for _, node := range nodes {
		arr, ok := asArray(node)
		if !ok || n < 0 || n >= len(arr) {
			continue
		}
		out = append(out, arr[n])
	}
	return out
}

func flattenArrays(nodes []any) []any {
	out := make([]any, 0, len(nodes))
	for _, node := range nodes {
		arr, ok := asArray(node)
		if !ok {
			continue
		}
		out = append(out, arr...)
	}
	return out
}

func child(v any, key string) (any, bool) {
	switch t := v.(type) {
	case map[string]any:
		x, ok := t[key]
		return x, ok
	case []any:
		n, err := strconv.Atoi(key)
		if err != nil || n < 0 || n >= len(t) {
			return nil, false
		}
		return t[n], true
	default:
		return nil, false
	}
}

func asArray(v any) ([]any, bool) {
	a, ok := v.([]any)
	return a, ok
}
