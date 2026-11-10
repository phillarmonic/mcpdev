package assert

import "fmt"

func matchSchema(value, schema any) string {
	sch, ok := schema.(map[string]any)
	if !ok {
		return "schema must be an object"
	}
	if t, ok := sch["type"].(string); ok {
		if msg := checkType(value, t); msg != "" {
			return msg
		}
	}
	if req, ok := sch["required"].([]any); ok {
		obj, isObj := value.(map[string]any)
		if !isObj {
			return "required properties need an object"
		}
		for _, r := range req {
			key, _ := r.(string)
			if _, ok := obj[key]; !ok {
				return "missing required " + key
			}
		}
	}
	if props, ok := sch["properties"].(map[string]any); ok {
		obj, _ := value.(map[string]any)
		for k, sub := range props {
			v, exists := obj[k]
			if !exists {
				continue
			}
			if msg := matchSchema(v, sub); msg != "" {
				return k + ": " + msg
			}
		}
	}
	return ""
}

func checkType(v any, want string) string {
	got := jsonType(v)
	if got == want || (want == "integer" && got == "number" && isInt(v)) {
		return ""
	}
	if want == "number" && (got == "number" || got == "integer") {
		return ""
	}
	return fmt.Sprintf("type expected %s actual %s", want, got)
}

func jsonType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		return "number"
	case int, int64:
		return "integer"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func isInt(v any) bool {
	switch n := v.(type) {
	case float64:
		return n == float64(int64(n))
	case int, int64:
		return true
	default:
		return false
	}
}
