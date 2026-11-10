package testkit

import (
	"fmt"
	"strconv"
)

// Tool is a stable tools/list entry.
type Tool struct {
	InputSchema map[string]any `json:"inputSchema"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
}

// Prompt is a stable prompts/list entry.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []PromptArgument `json:"arguments,omitzero"`
}

// PromptArgument is a prompts/get argument descriptor.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Resource is a stable resources/list entry.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// ResourceTemplate is a stable resources/templates/list entry.
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

func stableTools(extra int) []Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
	}
	tools := []Tool{
		{Name: "echo", Description: "Echo the text argument", InputSchema: schema},
		{Name: "fail", Description: "Return isError true", InputSchema: schema},
	}
	for i := range extra {
		tools = append(tools, Tool{
			Name:        fmt.Sprintf("extra-%02d", i),
			Description: "Pagination filler",
			InputSchema: schema,
		})
	}
	return tools
}

func stablePrompts() []Prompt {
	return []Prompt{
		{
			Name:        "greet",
			Description: "Stable greeting prompt",
			Arguments: []PromptArgument{
				{Name: "name", Description: "Who to greet", Required: true},
			},
		},
	}
}

func stableResources() []Resource {
	return []Resource{
		{
			URI:         "fixture://stable/note",
			Name:        "note",
			Description: "Stable resource body",
			MimeType:    "text/plain",
		},
		{
			URI:         "fixture://stable/blob",
			Name:        "blob",
			Description: "Stable binary body",
			MimeType:    "application/octet-stream",
		},
	}
}

func stableTemplates() []ResourceTemplate {
	return []ResourceTemplate{
		{
			URITemplate: "fixture://items/{id}",
			Name:        "item",
			Description: "Stable resource template",
			MimeType:    "text/plain",
		},
	}
}

func paginate[T any](all []T, cursor string, size int) ([]T, string, error) {
	start := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 || n > len(all) {
			return nil, "", fmt.Errorf("invalid cursor %q", cursor)
		}
		start = n
	}
	if start >= len(all) {
		return []T{}, "", nil
	}
	end := min(start+size, len(all))
	next := ""
	if end < len(all) {
		next = strconv.Itoa(end)
	}
	return all[start:end], next, nil
}
