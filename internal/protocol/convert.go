package protocol

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func convertToolPage(res *mcp.ListToolsResult) model.CatalogPage {
	page := model.CatalogPage{Kind: model.KindTools}
	if res == nil {
		page.Incomplete = true
		page.IncompleteReasons = []string{"missing tools/list result"}
		return page
	}
	page.NextCursor = res.NextCursor
	page.Tools = make([]model.Tool, 0, len(res.Tools))
	for _, t := range res.Tools {
		if t == nil {
			page.Incomplete = true
			page.IncompleteReasons = append(page.IncompleteReasons, "nil tool descriptor")
			continue
		}
		page.Tools = append(page.Tools, convertTool(t))
	}
	return page
}

func convertPromptPage(res *mcp.ListPromptsResult) model.CatalogPage {
	page := model.CatalogPage{Kind: model.KindPrompts}
	if res == nil {
		page.Incomplete = true
		page.IncompleteReasons = []string{"missing prompts/list result"}
		return page
	}
	page.NextCursor = res.NextCursor
	page.Prompts = make([]model.Prompt, 0, len(res.Prompts))
	for _, p := range res.Prompts {
		if p == nil {
			page.Incomplete = true
			page.IncompleteReasons = append(page.IncompleteReasons, "nil prompt descriptor")
			continue
		}
		page.Prompts = append(page.Prompts, convertPrompt(p))
	}
	return page
}

func convertResourcePage(res *mcp.ListResourcesResult) model.CatalogPage {
	page := model.CatalogPage{Kind: model.KindResources}
	if res == nil {
		page.Incomplete = true
		page.IncompleteReasons = []string{"missing resources/list result"}
		return page
	}
	page.NextCursor = res.NextCursor
	page.Resources = make([]model.Resource, 0, len(res.Resources))
	for _, r := range res.Resources {
		if r == nil {
			page.Incomplete = true
			page.IncompleteReasons = append(page.IncompleteReasons, "nil resource descriptor")
			continue
		}
		page.Resources = append(page.Resources, convertResource(r))
	}
	return page
}

func convertTemplatePage(res *mcp.ListResourceTemplatesResult) model.CatalogPage {
	page := model.CatalogPage{Kind: model.KindTemplates}
	if res == nil {
		page.Incomplete = true
		page.IncompleteReasons = []string{"missing resources/templates/list result"}
		return page
	}
	page.NextCursor = res.NextCursor
	page.Templates = make([]model.ResourceTemplate, 0, len(res.ResourceTemplates))
	for _, tmpl := range res.ResourceTemplates {
		if tmpl == nil {
			page.Incomplete = true
			page.IncompleteReasons = append(page.IncompleteReasons, "nil resource template")
			continue
		}
		page.Templates = append(page.Templates, convertTemplate(tmpl))
	}
	return page
}

func convertToolResult(res *mcp.CallToolResult) model.ToolResult {
	if res == nil {
		return model.ToolResult{}
	}
	out := model.ToolResult{IsError: res.IsError, Content: convertContents(res.Content)}
	if res.StructuredContent != nil {
		out.Structured = marshalSchema(res.StructuredContent)
	}
	return out
}

func convertPromptResult(res *mcp.GetPromptResult) model.PromptResult {
	if res == nil {
		return model.PromptResult{}
	}
	out := model.PromptResult{Description: res.Description}
	for _, msg := range res.Messages {
		if msg == nil {
			continue
		}
		out.Messages = append(out.Messages, model.PromptMessage{
			Role:    string(msg.Role),
			Content: convertContent(msg.Content),
		})
	}
	return out
}

func convertResourceBody(res *mcp.ReadResourceResult) model.ResourceBody {
	if res == nil {
		return model.ResourceBody{}
	}
	out := model.ResourceBody{}
	for _, part := range res.Contents {
		if part == nil {
			continue
		}
		out.Contents = append(out.Contents, model.ResourceContents{
			URI:      part.URI,
			MIMEType: part.MIMEType,
			Text:     part.Text,
			Blob:     part.Blob,
		})
	}
	return out
}

func convertContents(items []mcp.Content) []model.Content {
	out := make([]model.Content, 0, len(items))
	for _, item := range items {
		out = append(out, convertContent(item))
	}
	return out
}

func convertContent(c mcp.Content) model.Content {
	if c == nil {
		return model.Content{Kind: model.ContentUnrecognized, Description: "nil content"}
	}
	switch v := c.(type) {
	case *mcp.TextContent:
		return model.Content{Kind: model.ContentText, Text: v.Text}
	case *mcp.ImageContent:
		return model.Content{Kind: model.ContentImage, MIMEType: v.MIMEType, Data: v.Data}
	case *mcp.AudioContent:
		return model.Content{Kind: model.ContentAudio, MIMEType: v.MIMEType, Data: v.Data}
	case *mcp.ResourceLink:
		return model.Content{
			Kind:        model.ContentResourceLink,
			URI:         v.URI,
			Name:        v.Name,
			Title:       v.Title,
			Description: v.Description,
			MIMEType:    v.MIMEType,
		}
	case *mcp.EmbeddedResource:
		out := model.Content{Kind: model.ContentResource}
		if v.Resource != nil {
			out.URI = v.Resource.URI
			out.MIMEType = v.Resource.MIMEType
			out.Text = v.Resource.Text
			out.Data = v.Resource.Blob
		}
		return out
	default:
		return model.Content{Kind: model.ContentUnrecognized, Description: fmt.Sprintf("%T", c)}
	}
}

func convertCaps(c *mcp.ServerCapabilities) model.ServerCapabilities {
	if c == nil {
		return model.ServerCapabilities{}
	}
	out := model.ServerCapabilities{
		Experimental: sortedKeys(c.Experimental),
		Extensions:   sortedKeys(c.Extensions),
		Completions:  c.Completions != nil,
		Logging:      legacyLoggingDeclared(c),
	}
	if c.Tools != nil {
		out.Tools = true
		out.ToolsListChanged = c.Tools.ListChanged
	}
	if c.Prompts != nil {
		out.Prompts = true
		out.PromptsListChanged = c.Prompts.ListChanged
	}
	if c.Resources != nil {
		out.Resources = true
		out.ResourcesListChanged = c.Resources.ListChanged
		out.ResourceSubscribe = c.Resources.Subscribe
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := slices.Collect(maps.Keys(m))
	slices.Sort(keys)
	return keys
}

func legacyLoggingDeclared(c *mcp.ServerCapabilities) bool {
	// 2025-11-25 initialize still advertises logging during the deprecation window.
	return c.Logging != nil //nolint:staticcheck // SEP-2577 deprecates logging in 2026-07-28 only.
}

func marshalSchema(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
