package inspect_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/app/inspect"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

type fakeConn struct {
	sess *fakeSess
	err  error
}

func (f fakeConn) Connect(context.Context, port.Target) (port.Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sess, nil
}

type fakeSess struct {
	pages  map[model.PrimitiveKind][]model.CatalogPage
	info   model.SessionInfo
	closed bool
}

func (s *fakeSess) Era() model.ProtocolEra             { return s.info.Era }
func (s *fakeSess) DiscoverState() model.DiscoverState { return s.info.DiscoverState }
func (s *fakeSess) Info() model.SessionInfo            { return s.info }
func (s *fakeSess) Inventory() port.InventoryReader    { return s }
func (s *fakeSess) CallTool(context.Context, string, json.RawMessage) (model.ToolResult, error) {
	return model.ToolResult{}, errors.New("unused")
}

func (s *fakeSess) GetPrompt(context.Context, string, json.RawMessage) (model.PromptResult, error) {
	return model.PromptResult{}, errors.New("unused")
}

func (s *fakeSess) ReadResource(context.Context, string) (model.ResourceBody, error) {
	return model.ResourceBody{}, errors.New("unused")
}
func (s *fakeSess) Close(context.Context) error { s.closed = true; return nil }

func (s *fakeSess) List(_ context.Context, kind model.PrimitiveKind, _ model.Page) (model.CatalogPage, error) {
	pages := s.pages[kind]
	if len(pages) == 0 {
		return model.CatalogPage{Kind: kind}, nil
	}
	p := pages[0]
	s.pages[kind] = pages[1:]
	return p, nil
}

func fullInfo() model.SessionInfo {
	return model.SessionInfo{
		ServerName:      "lab",
		ServerTitle:     "Fixture Lab",
		ServerVersion:   "1",
		ProtocolVersion: string(model.Era20251125),
		Handshake:       model.HandshakeInitialize,
		DiscoverState:   model.DiscoverNotApplicable,
		Era:             model.Era20251125,
		Capabilities:    model.ServerCapabilities{Tools: true, Prompts: true, Resources: true},
	}
}

func TestRunEmptyComplete(t *testing.T) {
	sess := &fakeSess{info: fullInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{}}
	res, err := inspect.Run(context.Background(), fakeConn{sess: sess}, inspect.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Inventory.Incomplete || res.Inventory.Tools.Status != model.SectionComplete {
		t.Fatalf("%+v", res.Inventory)
	}
	if !sess.closed {
		t.Fatal("session not closed")
	}
}

func TestRunUnsupportedDistinct(t *testing.T) {
	info := fullInfo()
	info.Capabilities.Prompts = false
	sess := &fakeSess{info: info, pages: map[model.PrimitiveKind][]model.CatalogPage{}}
	res, err := inspect.Run(context.Background(), fakeConn{sess: sess}, inspect.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Inventory.Prompts.Status != model.SectionUnsupported {
		t.Fatalf("prompts %+v", res.Inventory.Prompts)
	}
	if res.Inventory.Incomplete {
		t.Fatal("unsupported must not be incomplete")
	}
}

func TestRunIncompleteExit(t *testing.T) {
	sess := &fakeSess{info: fullInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{
			Kind: model.KindTools, Tools: []model.Tool{{Name: "echo"}}, Incomplete: true,
			IncompleteReasons: []string{model.ReasonPaginationStopped},
		}},
	}}
	_, err := inspect.Run(context.Background(), fakeConn{sess: sess}, inspect.Request{})
	if errclass.ExitCode(err) != errclass.ExitIncomplete {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestRunNameFilterKeepsStatus(t *testing.T) {
	sess := &fakeSess{info: fullInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{
			Kind: model.KindTools, Tools: []model.Tool{{Name: "echo"}, {Name: "fail"}},
		}},
	}}
	res, err := inspect.Run(context.Background(), fakeConn{sess: sess}, inspect.Request{Names: []string{"echo"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Inventory.Tools.ItemCount != 2 || len(res.Inventory.Tools.Tools) != 1 {
		t.Fatalf("filter %+v", res.Inventory.Tools)
	}
	if res.Inventory.Tools.Status != model.SectionComplete {
		t.Fatal(res.Inventory.Tools.Status)
	}
}

func TestRunSectionFilter(t *testing.T) {
	sess := &fakeSess{info: fullInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{Kind: model.KindTools, Tools: []model.Tool{{Name: "echo"}}}},
	}}
	res, err := inspect.Run(context.Background(), fakeConn{sess: sess}, inspect.Request{Kinds: []model.PrimitiveKind{model.KindTools}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Inventory.Tools.Kind != model.KindTools || res.Inventory.Prompts.Kind != "" {
		t.Fatalf("kinds %+v", res.Inventory)
	}
}

func TestRunAllowIncomplete(t *testing.T) {
	sess := &fakeSess{info: fullInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{
			Kind: model.KindTools, Tools: []model.Tool{{Name: "echo"}}, Incomplete: true,
		}},
	}}
	res, err := inspect.Run(context.Background(), fakeConn{sess: sess}, inspect.Request{AllowIncomplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Inventory.Incomplete {
		t.Fatal("expected incomplete flag")
	}
}

func TestRunConnectionError(t *testing.T) {
	_, err := inspect.Run(context.Background(), fakeConn{err: errclass.Connection{Err: errors.New("down")}}, inspect.Request{})
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("%v", err)
	}
}
