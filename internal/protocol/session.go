package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

type session struct {
	closeErr error
	cs       *mcp.ClientSession
	info     model.SessionInfo
	mu       sync.Mutex
	closed   bool
}

func newSession(cs *mcp.ClientSession, info model.SessionInfo) *session {
	return &session{cs: cs, info: info}
}

var _ port.Session = (*session)(nil)

func (s *session) Era() model.ProtocolEra { return s.info.Era }

func (s *session) DiscoverState() model.DiscoverState { return s.info.DiscoverState }

func (s *session) Info() model.SessionInfo { return s.info }

func (s *session) Inventory() port.InventoryReader { return s }

func (s *session) CallTool(ctx context.Context, name string, args json.RawMessage) (model.ToolResult, error) {
	if err := s.ready(ctx); err != nil {
		return model.ToolResult{}, err
	}
	var arguments any = map[string]any{}
	if len(args) > 0 {
		arguments = args
	}
	res, err := s.cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return model.ToolResult{}, mapError(err)
	}
	return convertToolResult(res), nil
}

func (s *session) GetPrompt(ctx context.Context, name string, args json.RawMessage) (model.PromptResult, error) {
	if err := s.ready(ctx); err != nil {
		return model.PromptResult{}, err
	}
	var arguments map[string]string
	if len(args) > 0 {
		if err := json.Unmarshal(args, &arguments); err != nil {
			return model.PromptResult{}, errclass.Usage{Err: err, Msg: "protocol: prompt arguments"}
		}
	}
	res, err := s.cs.GetPrompt(ctx, &mcp.GetPromptParams{Name: name, Arguments: arguments})
	if err != nil {
		return model.PromptResult{}, mapError(err)
	}
	return convertPromptResult(res), nil
}

func (s *session) ReadResource(ctx context.Context, uri string) (model.ResourceBody, error) {
	if err := s.ready(ctx); err != nil {
		return model.ResourceBody{}, err
	}
	res, err := s.cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		return model.ResourceBody{}, mapError(err)
	}
	return convertResourceBody(res), nil
}

func (s *session) List(ctx context.Context, kind model.PrimitiveKind, page model.Page) (model.CatalogPage, error) {
	if err := s.ready(ctx); err != nil {
		return model.CatalogPage{}, err
	}
	switch kind {
	case model.KindTools:
		res, err := s.cs.ListTools(ctx, &mcp.ListToolsParams{Cursor: page.Cursor})
		if err != nil {
			return model.CatalogPage{}, mapError(err)
		}
		return convertToolPage(res), nil
	case model.KindPrompts:
		res, err := s.cs.ListPrompts(ctx, &mcp.ListPromptsParams{Cursor: page.Cursor})
		if err != nil {
			return model.CatalogPage{}, mapError(err)
		}
		return convertPromptPage(res), nil
	case model.KindResources:
		res, err := s.cs.ListResources(ctx, &mcp.ListResourcesParams{Cursor: page.Cursor})
		if err != nil {
			return model.CatalogPage{}, mapError(err)
		}
		return convertResourcePage(res), nil
	case model.KindTemplates:
		res, err := s.cs.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{Cursor: page.Cursor})
		if err != nil {
			return model.CatalogPage{}, mapError(err)
		}
		return convertTemplatePage(res), nil
	default:
		return model.CatalogPage{}, errclass.Protocol{Err: fmt.Errorf("protocol: unsupported catalog kind %q", kind)}
	}
}

func (s *session) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	// SDK Close cancels in-flight RPCs and is idempotent. Honor a canceled ctx
	// only after the transport is torn down so the session cannot leak.
	s.closeErr = mapError(s.cs.Close())
	if s.closeErr != nil {
		return s.closeErr
	}
	if err := ctx.Err(); err != nil {
		s.closeErr = mapError(err)
		return s.closeErr
	}
	return nil
}

func (s *session) ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return mapError(err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		if s.closeErr != nil {
			return s.closeErr
		}
		return errclass.Connection{Err: errors.New("protocol: session closed")}
	}
	return nil
}
