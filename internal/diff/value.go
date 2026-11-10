package diff

import (
	"bytes"
	jsonv2 "encoding/json/v2"
	"slices"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func mapTools(in []model.Tool) map[string]model.Tool {
	out := make(map[string]model.Tool, len(in))
	for _, item := range in {
		out[item.Name] = item
	}
	return out
}

func mapPrompts(in []model.Prompt) map[string]model.Prompt {
	out := make(map[string]model.Prompt, len(in))
	for _, item := range in {
		out[item.Name] = item
	}
	return out
}

func mapResources(in []model.Resource) map[string]model.Resource {
	out := make(map[string]model.Resource, len(in))
	for _, item := range in {
		out[item.URI] = item
	}
	return out
}

func mapTemplates(in []model.ResourceTemplate) map[string]model.ResourceTemplate {
	out := make(map[string]model.ResourceTemplate, len(in))
	for _, item := range in {
		out[item.URITemplate] = item
	}
	return out
}

func mapArgs(in []model.PromptArgument) map[string]model.PromptArgument {
	out := make(map[string]model.PromptArgument, len(in))
	for _, item := range in {
		out[item.Name] = item
	}
	return out
}

func keysOf[T any](m map[string]T) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for key := range m {
		out[key] = struct{}{}
	}
	return out
}

func nameSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, name := range in {
		out[name] = struct{}{}
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func toolPayload(t model.Tool) map[string]any {
	return map[string]any{
		"name":          t.Name,
		"title":         t.Title,
		"description":   t.Description,
		"input_schema":  jsonValue(t.InputSchema),
		"output_schema": jsonValue(t.OutputSchema),
	}
}

func promptPayload(p model.Prompt) map[string]any {
	return map[string]any{
		"name":        p.Name,
		"title":       p.Title,
		"description": p.Description,
		"arguments":   p.Arguments,
	}
}

func resourcePayload(r model.Resource) map[string]any {
	return map[string]any{
		"uri":         r.URI,
		"name":        r.Name,
		"title":       r.Title,
		"description": r.Description,
		"mime_type":   r.MIMEType,
	}
}

func templatePayload(r model.ResourceTemplate) map[string]any {
	return map[string]any{
		"uri_template": r.URITemplate,
		"name":         r.Name,
		"title":        r.Title,
		"description":  r.Description,
		"mime_type":    r.MIMEType,
	}
}

func iconValue(icons []model.Icon) any {
	if len(icons) == 0 {
		return nil
	}
	return icons
}

func jsonValue(raw []byte) any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var v any
	if err := jsonv2.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

func diffJSON(r *report, path string, from, to any, sev model.DiffSeverity, rationale string) {
	left, err := jsonv2.Marshal(from, jsonv2.Deterministic(true))
	if err != nil {
		return
	}
	right, err := jsonv2.Marshal(to, jsonv2.Deterministic(true))
	if err != nil {
		return
	}
	if bytes.Equal(left, right) {
		return
	}
	var fromV, toV any
	_ = jsonv2.Unmarshal(left, &fromV)
	_ = jsonv2.Unmarshal(right, &toV)
	if isEmptyJSON(fromV) {
		fromV = nil
	}
	if isEmptyJSON(toV) {
		toV = nil
	}
	if fromV == nil && toV == nil {
		return
	}
	r.add(path, model.CategoryChange, sev, fromV, toV, rationale)
}

func isEmptyJSON(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == "" || t == "null"
	case map[string]any:
		return len(t) == 0
	case []any:
		return len(t) == 0
	default:
		return false
	}
}
