// Package port defines SDK-free interfaces between application services and adapters.
package port

import (
	"context"
	"encoding/json"

	"github.com/phillarmonic/mcpdev/internal/model"
)

// Session is a live MCP connection owned by the protocol adapter.
type Session interface {
	Era() model.ProtocolEra
	DiscoverState() model.DiscoverState
	Info() model.SessionInfo
	Inventory() InventoryReader
	CallTool(ctx context.Context, name string, args json.RawMessage) (model.ToolResult, error)
	GetPrompt(ctx context.Context, name string, args json.RawMessage) (model.PromptResult, error)
	ReadResource(ctx context.Context, uri string) (model.ResourceBody, error)
	Close(ctx context.Context) error
}

// InventoryReader lists one catalog page. Callers loop on NextCursor and must keep Incomplete.
type InventoryReader interface {
	List(ctx context.Context, kind model.PrimitiveKind, page model.Page) (model.CatalogPage, error)
}
