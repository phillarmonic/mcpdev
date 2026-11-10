package runner_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/scenario/runner"
	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

type fakeConn struct {
	err   error
	sess  *fakeSess
	opens atomic.Int32
}

func (f *fakeConn) Connect(context.Context, port.Target) (port.Session, error) {
	f.opens.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	src := f.sess
	clone := &fakeSess{info: src.info, tool: src.tool, panic: src.panic}
	if src.pages != nil {
		clone.pages = map[model.PrimitiveKind][]model.CatalogPage{}
		for k, v := range src.pages {
			clone.pages[k] = append([]model.CatalogPage{}, v...)
		}
	}
	return clone, nil
}

type fakeSess struct {
	pages  map[model.PrimitiveKind][]model.CatalogPage
	tool   model.ToolResult
	info   model.SessionInfo
	closed atomic.Bool
	panic  bool
}

func (s *fakeSess) Era() model.ProtocolEra             { return s.info.Era }
func (s *fakeSess) DiscoverState() model.DiscoverState { return s.info.DiscoverState }
func (s *fakeSess) Info() model.SessionInfo            { return s.info }
func (s *fakeSess) Inventory() port.InventoryReader    { return s }
func (s *fakeSess) Close(context.Context) error        { s.closed.Store(true); return nil }
func (s *fakeSess) List(_ context.Context, kind model.PrimitiveKind, _ model.Page) (model.CatalogPage, error) {
	pages := s.pages[kind]
	if len(pages) == 0 {
		return model.CatalogPage{Kind: kind}, nil
	}
	p := pages[0]
	s.pages[kind] = pages[1:]
	return p, nil
}

func (s *fakeSess) CallTool(context.Context, string, json.RawMessage) (model.ToolResult, error) {
	if s.panic {
		panic("boom")
	}
	return s.tool, nil
}

func (s *fakeSess) GetPrompt(context.Context, string, json.RawMessage) (model.PromptResult, error) {
	return model.PromptResult{Messages: []model.PromptMessage{{Role: "user", Content: model.Content{Kind: model.ContentText, Text: "hi"}}}}, nil
}

func (s *fakeSess) ReadResource(context.Context, string) (model.ResourceBody, error) {
	return model.ResourceBody{Contents: []model.ResourceContents{{MIMEType: "text/markdown", Text: "# tools"}}}, nil
}

func labInfo() model.SessionInfo {
	return model.SessionInfo{
		ServerName: "lab", ProtocolVersion: string(model.Era20251125), Era: model.Era20251125,
		Capabilities: model.ServerCapabilities{Tools: true, Prompts: true, Resources: true},
	}
}

func TestIsolatedCasesCloseAndAssert(t *testing.T) {
	sess := &fakeSess{info: labInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{Kind: model.KindTools, Tools: []model.Tool{{Name: "list_files"}}}},
	}, tool: model.ToolResult{Content: []model.Content{{Kind: model.ContentText, Text: "module x"}}}}
	conn := &fakeConn{sess: sess}
	exists := true
	suite := schema.Suite{
		Name: "lab", SchemaVersion: 1,
		Cases: []schema.Case{
			{Name: "a", Steps: []schema.Step{
				{Inspect: &schema.InspectOp{Kind: "tools"}, Assert: []schema.Assertion{
					{Contains: &schema.ContainsAssert{Path: "$.result.tools[*].name", Value: "list_files"}},
					{JSONPath: &schema.JSONPathAssert{Path: "$.result.tools", Exists: &exists}},
				}},
				{Call: &schema.CallOp{Tool: "echo", Arguments: []byte(`{"x":1}`)}, Assert: []schema.Assertion{
					{Equal: &schema.EqualAssert{Path: "$.ok", Value: true}},
				}},
			}},
			{Name: "b", Steps: []schema.Step{
				{Read: &schema.ReadOp{URI: "kit://tools"}, Assert: []schema.Assertion{
					{ContentType: &schema.ContentTypeAssert{Equals: "text/markdown"}},
				}},
			}},
		},
	}
	rep := runner.Run(context.Background(), runner.Deps{Connector: conn}, suite, runner.Request{Jobs: 2})
	if rep.Status != "pass" {
		t.Fatalf("%+v", rep)
	}
	if conn.opens.Load() != 2 {
		t.Fatalf("opens %d", conn.opens.Load())
	}
}

func TestAssertionFailureStopsSteps(t *testing.T) {
	sess := &fakeSess{info: labInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{Kind: model.KindTools, Tools: []model.Tool{{Name: "echo"}}}},
	}}
	suite := schema.Suite{Name: "fail", SchemaVersion: 1, Cases: []schema.Case{{
		Name: "c", Steps: []schema.Step{
			{Inspect: &schema.InspectOp{Kind: "tools"}, Assert: []schema.Assertion{
				{Equal: &schema.EqualAssert{Path: "$.result.tools[0].name", Value: "nope"}},
			}},
			{Call: &schema.CallOp{Tool: "echo"}},
		},
	}}}
	rep := runner.Run(context.Background(), runner.Deps{Connector: &fakeConn{sess: sess}}, suite, runner.Request{Jobs: 1})
	if rep.Status != "fail" || rep.Category != "assertion" {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Cases[0].Steps) != 2 || rep.Cases[0].Steps[1].Status != "skip" {
		t.Fatalf("steps %+v", rep.Cases[0].Steps)
	}
}

func TestToolErrorExpectFailure(t *testing.T) {
	sess := &fakeSess{info: labInfo(), tool: model.ToolResult{IsError: true, Content: []model.Content{{Kind: model.ContentText, Text: "bad"}}}}
	suite := schema.Suite{Name: "err", SchemaVersion: 1, Cases: []schema.Case{{
		Name: "c", Steps: []schema.Step{{
			Call:          &schema.CallOp{Tool: "fail"},
			ExpectFailure: &schema.ExpectFailure{Category: "tool_error"},
			Assert:        []schema.Assertion{{ErrorCategory: &schema.ErrorCategoryAssert{Equals: "tool_error"}}},
		}},
	}}}
	rep := runner.Run(context.Background(), runner.Deps{Connector: &fakeConn{sess: sess}}, suite, runner.Request{Jobs: 1})
	if rep.Status != "pass" {
		t.Fatalf("%+v", rep)
	}
}

func TestConnectionErrorCategory(t *testing.T) {
	conn := &fakeConn{err: errclass.Connection{Err: errors.New("down")}}
	suite := schema.Suite{Name: "down", SchemaVersion: 1, Cases: []schema.Case{{
		Name: "c", Steps: []schema.Step{{Inspect: &schema.InspectOp{}}},
	}}}
	rep := runner.Run(context.Background(), runner.Deps{Connector: conn}, suite, runner.Request{Jobs: 1})
	if rep.Status != "error" || rep.Category != "connection" {
		t.Fatalf("%+v", rep)
	}
}

func TestPanicRecoveryStillCloses(t *testing.T) {
	sess := &fakeSess{info: labInfo(), panic: true}
	conn := &fakeConn{sess: sess}
	suite := schema.Suite{Name: "p", SchemaVersion: 1, Cases: []schema.Case{{
		Name: "c", Steps: []schema.Step{{Call: &schema.CallOp{Tool: "x"}}},
	}}}
	rep := runner.Run(context.Background(), runner.Deps{Connector: conn}, suite, runner.Request{Jobs: 1})
	if rep.Status != "error" || rep.Category != "internal" {
		t.Fatalf("%+v", rep)
	}
}

func TestCancelSkipsRemaining(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		sess := &fakeSess{info: labInfo(), pages: map[model.PrimitiveKind][]model.CatalogPage{}}
		suite := schema.Suite{Name: "c", SchemaVersion: 1, Cases: []schema.Case{{
			Name: "c", Steps: []schema.Step{{Inspect: &schema.InspectOp{}}, {Inspect: &schema.InspectOp{}}},
		}}}
		rep := runner.Run(ctx, runner.Deps{Connector: &fakeConn{sess: sess}}, suite, runner.Request{Jobs: 1})
		if rep.Status == "pass" {
			t.Fatalf("cancelled run passed: %+v", rep)
		}
	})
}

func TestAssertionValueInterpolation(t *testing.T) {
	sess := &fakeSess{
		info: labInfo(),
		tool: model.ToolResult{
			Structured: json.RawMessage(`{"echoed":"hello-world","kind":"string"}`),
			Content:    []model.Content{{Kind: model.ContentText, Text: "hello-world"}},
		},
	}
	suite := schema.Suite{Name: "interp", SchemaVersion: 1, Cases: []schema.Case{{
		Name: "c", Steps: []schema.Step{
			{Call: &schema.CallOp{Tool: "echo"}, Capture: []schema.Capture{
				{Name: "echoed", Path: "$.result.structured.echoed"},
				{Name: "kind", Path: "$.result.structured.kind"},
			}},
			{Call: &schema.CallOp{Tool: "echo"}, Assert: []schema.Assertion{
				{Contains: &schema.ContainsAssert{Path: "$.result.structured.echoed", Value: "${echoed}"}},
				{Equal: &schema.EqualAssert{Path: "$.result.structured.echoed", Value: "${echoed}"}},
				{Schema: &schema.SchemaAssert{
					Path:   "$.result.structured",
					Schema: json.RawMessage(`{"type":"object","properties":{"echoed":{"type":"${kind}"}}}`),
				}},
			}},
		},
	}}}
	rep := runner.Run(context.Background(), runner.Deps{Connector: &fakeConn{sess: sess}}, suite, runner.Request{Jobs: 1})
	if rep.Status != "pass" {
		t.Fatalf("%+v", rep)
	}
}

func TestAssertionValueInterpolationUnknown(t *testing.T) {
	sess := &fakeSess{info: labInfo(), tool: model.ToolResult{Content: []model.Content{{Kind: model.ContentText, Text: "hi"}}}}
	suite := schema.Suite{Name: "interp-miss", SchemaVersion: 1, Cases: []schema.Case{{
		Name: "c", Steps: []schema.Step{
			{Call: &schema.CallOp{Tool: "echo"}, Assert: []schema.Assertion{
				{Contains: &schema.ContainsAssert{Path: "$.result.text", Value: "${missing}"}},
			}},
		},
	}}}
	rep := runner.Run(context.Background(), runner.Deps{Connector: &fakeConn{sess: sess}}, suite, runner.Request{Jobs: 1})
	if rep.Status != "fail" || rep.Category != "incomplete" {
		t.Fatalf("%+v", rep)
	}
}

func TestMissingSecret(t *testing.T) {
	suite := schema.Suite{
		Name: "s", SchemaVersion: 1,
		Secrets: map[string]schema.Secret{"token": {FromEnv: "MISSING_SECRET_XYZ"}},
		Cases:   []schema.Case{{Name: "c", Steps: []schema.Step{{Inspect: &schema.InspectOp{}}}}},
	}
	rep := runner.Run(context.Background(), runner.Deps{Connector: &fakeConn{sess: &fakeSess{info: labInfo()}}}, suite, runner.Request{})
	if rep.Status != "error" || rep.Category != "usage" {
		t.Fatalf("%+v", rep)
	}
}
