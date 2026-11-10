package snapshot

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func richInventory(era model.ProtocolEra) model.Inventory {
	return model.Inventory{
		ServerName:      "lab",
		ServerTitle:     "Fixture lab",
		ServerVersion:   "1.0.0",
		Instructions:    "mcpdev fixture laboratory",
		ProtocolVersion: string(era),
		Handshake:       handshakeFor(era),
		DiscoverState:   discoverFor(era),
		Era:             era,
		Capabilities: model.ServerCapabilities{
			Tools: true, Prompts: true, Resources: true,
			ToolsListChanged: true, PromptsListChanged: true,
			ResourcesListChanged: true, ResourceSubscribe: true,
		},
		Tools: model.CatalogSection{
			Kind: model.KindTools, Status: model.SectionComplete, ItemCount: 2, Pages: 1,
			Tools: []model.Tool{
				{
					Name:         "echo",
					Title:        "Echo",
					Description:  "Echo the text argument",
					InputSchema:  json.RawMessage(`{"properties":{"text":{"type":"string"}},"type":"object"}`),
					OutputSchema: json.RawMessage(`{"properties":{"ok":{"type":"boolean"}},"type":"object"}`),
					Meta:         json.RawMessage(`{"owner":"lab","token":"secret-value"}`),
					Icons:        []model.Icon{{Source: "https://example.invalid/echo.png", MIMEType: "image/png", Sizes: []string{"64x64"}}},
					Annotations:  &model.ToolAnnotations{Title: "Echo", ReadOnlyHint: true, DestructiveHint: new(bool)},
				},
				{
					Name:        "fail",
					Description: "Return isError true",
					InputSchema: json.RawMessage(`{"$defs":{"inner":{"type":"string"}},"properties":{"n":{"minimum":1.5,"type":"number"},"z":{"type":"integer"}},"required":["n"],"type":"object"}`),
				},
			},
		},
		Prompts: model.CatalogSection{
			Kind: model.KindPrompts, Status: model.SectionComplete, ItemCount: 1, Pages: 1,
			Prompts: []model.Prompt{{
				Name: "greet", Description: "Stable greeting prompt",
				Arguments: []model.PromptArgument{{Name: "name", Description: "Who to greet", Required: true}},
			}},
		},
		Resources: model.CatalogSection{
			Kind: model.KindResources, Status: model.SectionComplete, ItemCount: 2, Pages: 1,
			Resources: []model.Resource{
				{URI: "fixture://stable/note", Name: "note", Description: "Stable resource body", MIMEType: "text/plain", Size: 11},
				{URI: "file:///tmp/weird path", Name: "space", Description: "URI must stay verbatim", MIMEType: "text/plain"},
			},
		},
		Templates: model.CatalogSection{
			Kind: model.KindTemplates, Status: model.SectionComplete, ItemCount: 1, Pages: 1,
			Templates: []model.ResourceTemplate{{
				URITemplate: "fixture://items/{id}", Name: "item", Description: "Stable resource template", MIMEType: "text/plain",
			}},
		},
	}
}

func handshakeFor(era model.ProtocolEra) model.HandshakeKind {
	if era == model.Era20260728 {
		return model.HandshakeDiscover
	}
	return model.HandshakeInitialize
}

func discoverFor(era model.ProtocolEra) model.DiscoverState {
	if era == model.Era20260728 {
		return model.DiscoverOK
	}
	return model.DiscoverNotApplicable
}

func frozenMeta() Meta {
	return Meta{
		CapturedAt:       time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
		GeneratorVersion: "0.1.0",
		Kind:             "stdio",
		Command:          []string{"./bin/lab"},
		Headers:          map[string]string{"Authorization": "Bearer secret-token", "Accept": "application/json"},
		URL:              "https://user:pass@example.invalid/mcp?token=abc",
		SessionID:        "sess-volatile",
	}
}

func mustSnap(t *testing.T, inv model.Inventory, meta Meta) model.Snapshot {
	t.Helper()
	snap, err := FromInventory(inv, meta)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}
