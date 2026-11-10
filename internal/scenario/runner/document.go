package runner

import (
	jsonv2 "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/app/inspect"
	"github.com/phillarmonic/mcpdev/internal/app/invoke"
	"github.com/phillarmonic/mcpdev/internal/model"
)

func inspectDoc(res inspect.Result) map[string]any {
	result := map[string]any{}
	if res.Inventory.Tools.Kind != "" {
		items := make([]any, 0, len(res.Inventory.Tools.Tools))
		for _, t := range res.Inventory.Tools.Tools {
			items = append(items, map[string]any{"name": t.Name, "title": t.Title, "description": t.Description})
		}
		result["tools"] = items
	}
	if res.Inventory.Prompts.Kind != "" {
		items := make([]any, 0, len(res.Inventory.Prompts.Prompts))
		for _, p := range res.Inventory.Prompts.Prompts {
			items = append(items, map[string]any{"name": p.Name, "title": p.Title, "description": p.Description})
		}
		result["prompts"] = items
	}
	if res.Inventory.Resources.Kind != "" {
		items := make([]any, 0, len(res.Inventory.Resources.Resources))
		for _, r := range res.Inventory.Resources.Resources {
			items = append(items, map[string]any{"uri": r.URI, "name": r.Name, "mimeType": r.MIMEType})
		}
		result["resources"] = items
	}
	if res.Inventory.Templates.Kind != "" {
		items := make([]any, 0, len(res.Inventory.Templates.Templates))
		for _, r := range res.Inventory.Templates.Templates {
			items = append(items, map[string]any{"uriTemplate": r.URITemplate, "name": r.Name})
		}
		result["templates"] = items
	}
	return map[string]any{
		"ok":     !res.Inventory.Incomplete,
		"result": result,
	}
}

func callDoc(res invoke.Result) (map[string]any, string) {
	out := map[string]any{"ok": true, "result": map[string]any{}}
	mime := ""
	if res.Tool != nil {
		text := firstText(res.Tool.Content)
		mime = firstMIME(res.Tool.Content)
		result := map[string]any{
			"isError": res.Tool.IsError,
			"text":    text,
			"content": contentBlocks(res.Tool.Content),
		}
		if len(res.Tool.Structured) > 0 {
			var v any
			if err := jsonv2.Unmarshal(res.Tool.Structured, &v); err == nil {
				result["structured"] = v
			}
		}
		out["ok"] = !res.Tool.IsError
		out["result"] = result
	}
	if res.Prompt != nil {
		msgs := make([]any, 0, len(res.Prompt.Messages))
		for _, m := range res.Prompt.Messages {
			msgs = append(msgs, map[string]any{"role": m.Role, "content": contentMap(m.Content)})
			if mime == "" {
				mime = m.Content.MIMEType
			}
		}
		out["result"] = map[string]any{"description": res.Prompt.Description, "messages": msgs}
	}
	if res.Resource != nil {
		parts := make([]any, 0, len(res.Resource.Contents))
		for _, c := range res.Resource.Contents {
			parts = append(parts, map[string]any{"uri": c.URI, "mimeType": c.MIMEType, "text": c.Text})
			if mime == "" {
				mime = c.MIMEType
			}
		}
		out["result"] = map[string]any{"contents": parts}
	}
	return out, mime
}

func firstText(blocks []model.Content) string {
	for _, c := range blocks {
		if c.Kind == model.ContentText && c.Text != "" {
			return c.Text
		}
	}
	return ""
}

func firstMIME(blocks []model.Content) string {
	for _, c := range blocks {
		if c.MIMEType != "" {
			return c.MIMEType
		}
	}
	return ""
}

func contentBlocks(blocks []model.Content) []any {
	out := make([]any, 0, len(blocks))
	for _, c := range blocks {
		out = append(out, contentMap(c))
	}
	return out
}

func contentMap(c model.Content) map[string]any {
	return map[string]any{
		"kind":     string(c.Kind),
		"text":     c.Text,
		"mimeType": c.MIMEType,
		"uri":      c.URI,
	}
}
