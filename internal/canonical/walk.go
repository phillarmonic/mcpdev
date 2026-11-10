package canonical

import (
	"slices"
	"strings"

	jsonv2 "encoding/json/v2"
)

func normalizeSchema(v any) any {
	switch t := v.(type) {
	case bool:
		return t
	case map[string]any:
		return normalizeSchemaObject(t)
	default:
		return normalizeJSON(v)
	}
}

func normalizeSchemaObject(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, val := range in {
		out[key] = normalizeKeyword(key, val)
	}
	if req, ok := out["required"].([]any); ok && len(req) == 0 {
		delete(out, "required")
	}
	return out
}

func normalizeKeyword(key string, val any) any {
	switch key {
	case "properties", "$defs", "definitions", "patternProperties", "dependentSchemas":
		return normalizeSchemaMap(val)
	case "allOf", "anyOf", "oneOf", "prefixItems":
		return normalizeSchemaList(val)
	case "items", "additionalItems", "additionalProperties", "unevaluatedItems", "unevaluatedProperties",
		"not", "if", "then", "else", "contains", "propertyNames", "contentSchema":
		return normalizeItemsOrSchema(key, val)
	case "required":
		return normalizeStringSet(val)
	case "type":
		return normalizeType(val)
	case "enum":
		return normalizeEnum(val)
	case "dependentRequired":
		return normalizeDependentRequired(val)
	default:
		return normalizeJSON(val)
	}
}

func normalizeItemsOrSchema(key string, val any) any {
	if key == "items" {
		if _, ok := val.([]any); ok {
			return normalizeSchemaList(val)
		}
	}
	return normalizeSchema(val)
}

func normalizeSchemaMap(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return normalizeJSON(v)
	}
	out := make(map[string]any, len(m))
	for key, val := range m {
		out[key] = normalizeSchema(val)
	}
	return out
}

func normalizeSchemaList(v any) any {
	arr, ok := v.([]any)
	if !ok {
		return normalizeJSON(v)
	}
	out := make([]any, len(arr))
	for i, item := range arr {
		out[i] = normalizeSchema(item)
	}
	return out
}

func normalizeStringSet(v any) any {
	strs, ok := stringSlice(v)
	if !ok {
		return normalizeJSON(v)
	}
	slices.Sort(strs)
	strs = slices.Compact(strs)
	return toAnyStrings(strs)
}

func normalizeType(v any) any {
	if s, ok := v.(string); ok {
		return s
	}
	strs, ok := stringSlice(v)
	if !ok {
		return normalizeJSON(v)
	}
	slices.Sort(strs)
	strs = slices.Compact(strs)
	switch len(strs) {
	case 0:
		return []any{}
	case 1:
		return strs[0]
	default:
		return toAnyStrings(strs)
	}
}

func normalizeEnum(v any) any {
	arr, ok := v.([]any)
	if !ok {
		return normalizeJSON(v)
	}
	out := make([]any, len(arr))
	for i, item := range arr {
		out[i] = normalizeJSON(item)
	}
	return sortByCanonical(out)
}

func normalizeDependentRequired(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return normalizeJSON(v)
	}
	out := make(map[string]any, len(m))
	for key, val := range m {
		out[key] = normalizeStringSet(val)
	}
	return out
}

func normalizeJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for key, val := range t {
			out[key] = normalizeJSON(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = normalizeJSON(val)
		}
		return out
	default:
		return v
	}
}

func stringSlice(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out[i] = s
	}
	return out, true
}

func toAnyStrings(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

func sortByCanonical(in []any) []any {
	keys := make([]string, len(in))
	for i, val := range in {
		body, err := jsonv2.Marshal(val, jsonv2.Deterministic(true))
		if err != nil {
			continue
		}
		keys[i] = string(body)
	}
	order := make([]int, len(in))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return strings.Compare(keys[a], keys[b])
	})
	out := make([]any, len(in))
	for i, idx := range order {
		out[i] = in[idx]
	}
	return out
}
