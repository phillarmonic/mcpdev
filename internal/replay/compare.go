package replay

import (
	"fmt"
	"slices"

	jsonv2 "encoding/json/v2"
)

func matchExpect(step int, exp *Expect, method string, params any) error {
	if exp == nil {
		return nil
	}
	if exp.Method != method {
		return Mismatch{Step: step, Method: method, Field: "method", Expected: exp.Method, Actual: method}
	}
	if len(exp.Meta) > 0 {
		actMeta := metaOf(params)
		path, ok := firstDiff("", decodeJSON(exp.Meta), actMeta)
		if !ok {
			return Mismatch{
				Step:     step,
				Method:   method,
				Field:    "metadata",
				Path:     path,
				Expected: previewJSON(exp.Meta),
				Actual:   previewAny(actMeta),
			}
		}
	}
	if len(exp.Params) > 0 {
		path, ok := firstDiff("", decodeJSON(exp.Params), params)
		if !ok {
			return Mismatch{
				Step:     step,
				Method:   method,
				Field:    "payload",
				Path:     path,
				Expected: previewJSON(exp.Params),
				Actual:   previewAny(params),
			}
		}
	}
	return nil
}

func metaOf(params any) any {
	m, ok := params.(map[string]any)
	if !ok {
		return nil
	}
	return m["_meta"]
}

func decodeJSON(raw []byte) any {
	var v any
	if err := jsonv2.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

func previewJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	b, err := jsonv2.Marshal(decodeJSON(raw), jsonv2.Deterministic(true))
	if err != nil {
		return string(raw)
	}
	return string(b)
}

func previewAny(v any) string {
	if v == nil {
		return "null"
	}
	b, err := jsonv2.Marshal(v, jsonv2.Deterministic(true))
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func firstDiff(path string, exp, act any) (string, bool) {
	if exp == nil {
		return path, true
	}
	switch e := exp.(type) {
	case map[string]any:
		am, ok := act.(map[string]any)
		if !ok {
			return pathOrRoot(path), false
		}
		keys := make([]string, 0, len(e))
		for k := range e {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			child := path + "/" + k
			g, has := am[k]
			if !has {
				return child, false
			}
			if p, eq := firstDiff(child, e[k], g); !eq {
				return p, false
			}
		}
		return path, true
	case []any:
		aa, ok := act.([]any)
		if !ok || len(aa) != len(e) {
			return pathOrRoot(path), false
		}
		for i := range e {
			child := fmt.Sprintf("%s/%d", path, i)
			if p, eq := firstDiff(child, e[i], aa[i]); !eq {
				return p, false
			}
		}
		return path, true
	default:
		if !jsonEqual(exp, act) {
			return pathOrRoot(path), false
		}
		return path, true
	}
}

func pathOrRoot(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func jsonEqual(a, b any) bool {
	ab, err1 := jsonv2.Marshal(a, jsonv2.Deterministic(true))
	bb, err2 := jsonv2.Marshal(b, jsonv2.Deterministic(true))
	if err1 != nil || err2 != nil {
		return false
	}
	return string(ab) == string(bb)
}
