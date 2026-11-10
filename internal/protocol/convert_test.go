package protocol

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestConvertToolPageKeepsCursorAndNilItems(t *testing.T) {
	page := convertToolPage(&mcp.ListToolsResult{
		NextCursor: "next-1",
		Tools: []*mcp.Tool{
			{
				Name: "echo", Title: "Echo", Description: "repeat",
				InputSchema: map[string]any{"type": "object"},
				Meta:        mcp.Meta{"vendor": "keep"},
				Icons:       []mcp.Icon{{Source: "https://example.invalid/echo.svg", MIMEType: "image/svg+xml"}},
				Annotations: &mcp.ToolAnnotations{Title: "Echo tool", ReadOnlyHint: true},
			},
			nil,
		},
	})
	if page.Kind != model.KindTools {
		t.Fatalf("kind: got %q", page.Kind)
	}
	if page.NextCursor != "next-1" {
		t.Fatalf("cursor: got %q", page.NextCursor)
	}
	if len(page.Tools) != 1 || page.Tools[0].Name != "echo" {
		t.Fatalf("tools: %+v", page.Tools)
	}
	if !page.Incomplete {
		t.Fatal("nil descriptor must mark the page incomplete")
	}
	if len(page.Tools[0].InputSchema) == 0 {
		t.Fatal("input schema missing")
	}
	if page.Tools[0].Annotations == nil || !page.Tools[0].Annotations.ReadOnlyHint {
		t.Fatalf("annotations: %+v", page.Tools[0].Annotations)
	}
	if len(page.Tools[0].Icons) != 1 || page.Tools[0].Icons[0].Source == "" {
		t.Fatalf("icons: %+v", page.Tools[0].Icons)
	}
	if len(page.Tools[0].Meta) == 0 {
		t.Fatal("meta missing")
	}
}

func TestConvertContentKinds(t *testing.T) {
	text := convertContent(&mcp.TextContent{Text: "hi"})
	if text.Kind != model.ContentText || text.Text != "hi" {
		t.Fatalf("text: %+v", text)
	}
	img := convertContent(&mcp.ImageContent{MIMEType: "image/png", Data: []byte{1}})
	if img.Kind != model.ContentImage || img.MIMEType != "image/png" {
		t.Fatalf("image: %+v", img)
	}
	link := convertContent(&mcp.ResourceLink{URI: "fixture://stable/note", Name: "note"})
	if link.Kind != model.ContentResourceLink || link.URI != "fixture://stable/note" {
		t.Fatalf("link: %+v", link)
	}
	unrec := convertContent(nil)
	if unrec.Kind != model.ContentUnrecognized {
		t.Fatalf("unrecognized kind: %q", unrec.Kind)
	}
	if unrec.Description != "nil content" {
		t.Fatalf("unrecognized description: %q", unrec.Description)
	}
}

func TestConvertPromptAndResourcePages(t *testing.T) {
	prompts := convertPromptPage(&mcp.ListPromptsResult{
		Prompts: []*mcp.Prompt{{Name: "greet", Arguments: []*mcp.PromptArgument{{Name: "name", Required: true}}}},
	})
	if len(prompts.Prompts) != 1 || !prompts.Prompts[0].Arguments[0].Required {
		t.Fatalf("prompts: %+v", prompts.Prompts)
	}
	resources := convertResourcePage(&mcp.ListResourcesResult{
		Resources: []*mcp.Resource{{
			URI: "fixture://stable/note", Name: "note", MIMEType: "text/plain",
			Annotations: &mcp.Annotations{Audience: []mcp.Role{"user"}, Priority: 0.5},
		}},
	})
	if resources.Resources[0].URI != "fixture://stable/note" || resources.Resources[0].MIMEType != "text/plain" {
		t.Fatalf("resources: %+v", resources.Resources)
	}
	if resources.Resources[0].Annotations == nil || resources.Resources[0].Annotations.Priority != 0.5 {
		t.Fatalf("resource annotations: %+v", resources.Resources[0].Annotations)
	}
	templates := convertTemplatePage(&mcp.ListResourceTemplatesResult{
		ResourceTemplates: []*mcp.ResourceTemplate{{URITemplate: "fixture://items/{id}", Name: "item"}},
	})
	if templates.Templates[0].URITemplate != "fixture://items/{id}" {
		t.Fatalf("templates: %+v", templates.Templates)
	}
}

func TestConvertResults(t *testing.T) {
	tool := convertToolResult(&mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: "nope"}},
		StructuredContent: map[string]any{"ok": false},
	})
	if !tool.IsError || tool.Content[0].Text != "nope" {
		t.Fatalf("tool result: %+v", tool)
	}
	var structured map[string]any
	if err := json.Unmarshal(tool.Structured, &structured); err != nil || structured["ok"] != false {
		t.Fatalf("structured: %s err=%v", tool.Structured, err)
	}
	prompt := convertPromptResult(&mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: "hello"}}},
	})
	if prompt.Messages[0].Content.Text != "hello" {
		t.Fatalf("prompt: %+v", prompt)
	}
	body := convertResourceBody(&mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{URI: "fixture://stable/note", Text: "stable-note"}},
	})
	if body.Contents[0].Text != "stable-note" {
		t.Fatalf("body: %+v", body)
	}
}

func TestConvertCapsNilAndTools(t *testing.T) {
	if got := convertCaps(nil); got.Tools {
		t.Fatalf("nil: %+v", got)
	}
	got := convertCaps(&mcp.ServerCapabilities{
		Tools:        &mcp.ToolCapabilities{ListChanged: true},
		Experimental: map[string]any{"vendor.flag": true},
	})
	if !got.Tools || !got.ToolsListChanged {
		t.Fatalf("tools: %+v", got)
	}
	if len(got.Experimental) != 1 || got.Experimental[0] != "vendor.flag" {
		t.Fatalf("experimental: %v", got.Experimental)
	}
}

func TestDiscoverState(t *testing.T) {
	if got := discoverState("", model.Era20260728, Decision{}); got != model.DiscoverOK {
		t.Fatalf("auto modern: %q", got)
	}
	if got := discoverState("", model.Era20251125, Decision{Fallback: true}); got != model.DiscoverSkipped {
		t.Fatalf("auto fallback: %q", got)
	}
	if got := discoverState(model.Era20251125, model.Era20251125, Decision{}); got != model.DiscoverNotApplicable {
		t.Fatalf("pin legacy: %q", got)
	}
}
