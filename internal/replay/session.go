package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

type session struct {
	clock  port.Clock
	events []Delivery
	doc    Document
	info   model.SessionInfo
	mu     sync.Mutex
	next   int
	closed bool
}

var _ port.Session = (*session)(nil)

func newSession(doc Document, clk port.Clock) *session {
	if clk == nil {
		clk = clock.Wall{}
	}
	return &session{doc: doc, clock: clk, info: sessionInfo(doc.Session)}
}

func (s *session) Era() model.ProtocolEra { return s.info.Era }

func (s *session) DiscoverState() model.DiscoverState { return s.info.DiscoverState }

func (s *session) Info() model.SessionInfo { return s.info }

func (s *session) Inventory() port.InventoryReader { return s }

func (s *session) ConnectorName() string { return "replay" }

func (s *session) Constructed() string { return Kind }

func (s *session) Diagnostics() map[string]string {
	return map[string]string{"replay": "1"}
}

func (s *session) Events() []Delivery {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Delivery, len(s.events))
	copy(out, s.events)
	return out
}

func (s *session) CallTool(ctx context.Context, name string, args json.RawMessage) (model.ToolResult, error) {
	params := map[string]any{"name": name}
	if len(args) > 0 {
		var v any
		if err := json.Unmarshal(args, &v); err != nil {
			return model.ToolResult{}, errclass.Usage{Err: err, Msg: "replay: tool arguments"}
		}
		params["arguments"] = v
	}
	payload, err := s.play(ctx, "tools/call", params)
	if err != nil {
		return model.ToolResult{}, err
	}
	return toolResultFrom(payload)
}

func (s *session) GetPrompt(ctx context.Context, name string, args json.RawMessage) (model.PromptResult, error) {
	params := map[string]any{"name": name}
	if len(args) > 0 {
		var v any
		if err := json.Unmarshal(args, &v); err != nil {
			return model.PromptResult{}, errclass.Usage{Err: err, Msg: "replay: prompt arguments"}
		}
		params["arguments"] = v
	}
	payload, err := s.play(ctx, "prompts/get", params)
	if err != nil {
		return model.PromptResult{}, err
	}
	return promptResultFrom(payload)
}

func (s *session) ReadResource(ctx context.Context, uri string) (model.ResourceBody, error) {
	payload, err := s.play(ctx, "resources/read", map[string]any{"uri": uri})
	if err != nil {
		return model.ResourceBody{}, err
	}
	return resourceBodyFrom(payload)
}

func (s *session) List(ctx context.Context, kind model.PrimitiveKind, page model.Page) (model.CatalogPage, error) {
	params := map[string]any{}
	if page.Cursor != "" {
		params["cursor"] = page.Cursor
	}
	payload, err := s.play(ctx, listMethod(kind), params)
	if err != nil {
		return model.CatalogPage{}, err
	}
	return catalogFrom(kind, payload)
}

func (s *session) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return ctx.Err()
}

func (s *session) play(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, errclass.Connection{Err: err}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errclass.Connection{Err: errors.New("replay: session closed")}
	}
	if s.next >= len(s.doc.Steps) {
		return nil, Mismatch{Step: s.next + 1, Method: method, Field: "method", Expected: "(end)", Actual: method}
	}
	st := s.doc.Steps[s.next]
	s.next++
	if err := matchExpect(s.next, st.Expect, method, params); err != nil {
		return nil, err
	}
	var primary json.RawMessage
	var primaryErr error
	havePrimary := false
	for _, d := range st.Deliver {
		if err := waitDelay(ctx, s.clock, d.DelayMS); err != nil {
			return nil, err
		}
		switch d.Kind {
		case KindSSE, KindNotification, KindMalformed:
			s.events = append(s.events, d)
			continue
		case KindProcessExit:
			return nil, exitErr(d)
		case KindError:
			if !havePrimary {
				primaryErr = rpcErr(d)
				havePrimary = true
			}
		case KindResult:
			if !havePrimary {
				primary = d.Payload
				havePrimary = true
			}
		}
	}
	if !havePrimary {
		return nil, errclass.Protocol{Err: fmt.Errorf("replay: step %d has no result or error", s.next)}
	}
	return primary, primaryErr
}

func waitDelay(ctx context.Context, clk port.Clock, ms int) error {
	if ms <= 0 {
		return ctx.Err()
	}
	if clk == nil {
		clk = clock.Wall{}
	}
	select {
	case <-ctx.Done():
		return errclass.Connection{Err: ctx.Err()}
	case <-clk.After(time.Duration(ms) * time.Millisecond):
		return nil
	}
}

func rpcErr(d Delivery) error {
	msg := d.Message
	if msg == "" {
		msg = "replay: scripted error"
	}
	switch d.Category {
	case "connection":
		return errclass.Connection{Err: fmt.Errorf("%s", msg)}
	case "usage":
		return errclass.Usage{Msg: msg}
	default:
		return errclass.Protocol{Err: fmt.Errorf("%s", msg)}
	}
}

func exitErr(d Delivery) error {
	return errclass.Connection{Err: fmt.Errorf("replay: process exited with code %d", d.Code)}
}
