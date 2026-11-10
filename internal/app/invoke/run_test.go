package invoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/phillarmonic/mcpdev/internal/app/invoke"
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
	toolErr  error
	promptN  string
	readURI  string
	prompt   model.PromptResult
	callArgs json.RawMessage
	promptA  json.RawMessage
	resource model.ResourceBody
	tool     model.ToolResult
	info     model.SessionInfo
	block    bool
}

func (s *fakeSess) Era() model.ProtocolEra             { return s.info.Era }
func (s *fakeSess) DiscoverState() model.DiscoverState { return s.info.DiscoverState }
func (s *fakeSess) Info() model.SessionInfo            { return s.info }
func (s *fakeSess) Inventory() port.InventoryReader    { return s }
func (s *fakeSess) List(context.Context, model.PrimitiveKind, model.Page) (model.CatalogPage, error) {
	return model.CatalogPage{}, nil
}
func (s *fakeSess) Close(context.Context) error { return nil }

func (s *fakeSess) CallTool(ctx context.Context, _ string, args json.RawMessage) (model.ToolResult, error) {
	s.callArgs = append(json.RawMessage(nil), args...)
	if s.block {
		<-ctx.Done()
		return model.ToolResult{}, errclass.Connection{Err: ctx.Err()}
	}
	if s.toolErr != nil {
		return model.ToolResult{}, s.toolErr
	}
	return s.tool, nil
}

func (s *fakeSess) GetPrompt(_ context.Context, name string, args json.RawMessage) (model.PromptResult, error) {
	s.promptN = name
	s.promptA = append(json.RawMessage(nil), args...)
	return s.prompt, nil
}

func (s *fakeSess) ReadResource(_ context.Context, uri string) (model.ResourceBody, error) {
	s.readURI = uri
	return s.resource, nil
}

func caps(tools, prompts, resources bool) model.SessionInfo {
	return model.SessionInfo{
		ServerName:      "lab",
		ProtocolVersion: string(model.Era20251125),
		Era:             model.Era20251125,
		DiscoverState:   model.DiscoverNotApplicable,
		Capabilities:    model.ServerCapabilities{Tools: tools, Prompts: prompts, Resources: resources},
	}
}

func TestCallSuccessAndFileArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	if err := os.WriteFile(path, []byte(`{"text":"hi"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sess := &fakeSess{
		info: caps(true, true, true),
		tool: model.ToolResult{Content: []model.Content{{Kind: model.ContentText, Text: "hi"}}},
	}
	res, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
		Kind:     invoke.KindCall,
		Name:     "echo",
		ArgsFile: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Tool == nil || res.Tool.Content[0].Text != "hi" {
		t.Fatalf("%+v", res.Tool)
	}
	if !bytes.Contains(sess.callArgs, []byte(`"text":"hi"`)) {
		t.Fatalf("args %s", sess.callArgs)
	}
}

func TestCallIsErrorDistinct(t *testing.T) {
	sess := &fakeSess{
		info: caps(true, false, false),
		tool: model.ToolResult{IsError: true, Content: []model.Content{{Kind: model.ContentText, Text: "failed"}}},
	}
	res, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{Kind: invoke.KindCall, Name: "fail"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Tool == nil || !res.Tool.IsError {
		t.Fatal("expected tool isError")
	}
	_, err = invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{Kind: invoke.KindCall, Name: "fail", FailOnToolError: true})
	if errclass.ExitCode(err) != errclass.ExitAssertion {
		t.Fatalf("code %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestCallProtocolAndUnsupported(t *testing.T) {
	sess := &fakeSess{info: caps(true, false, false), toolErr: errclass.Protocol{Err: errors.New("jsonrpc")}}
	_, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{Kind: invoke.KindCall, Name: "echo"})
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("%v", err)
	}
	sess2 := &fakeSess{info: caps(false, false, false)}
	_, err = invoke.Run(context.Background(), fakeConn{sess: sess2}, invoke.Request{Kind: invoke.KindCall, Name: "echo"})
	if errclass.ExitCode(err) != errclass.ExitProtocol || !strings.Contains(err.Error(), "not advertised") {
		t.Fatalf("%v", err)
	}
}

func TestCallInvalidJSON(t *testing.T) {
	sess := &fakeSess{info: caps(true, false, false)}
	_, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
		Kind:     invoke.KindCall,
		Name:     "echo",
		ArgsJSON: "{",
	})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("%v", err)
	}
}

func TestCallTimeoutAndCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := &fakeSess{info: caps(true, false, false), block: true}
		_, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
			Kind:   invoke.KindCall,
			Name:   "echo",
			Target: port.Target{Timeout: time.Second},
		})
		if errclass.ExitCode(err) != errclass.ExitConnection {
			t.Fatalf("timeout %v", err)
		}
	})
	sess := &fakeSess{info: caps(true, false, false), block: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := invoke.Run(ctx, fakeConn{sess: sess}, invoke.Request{Kind: invoke.KindCall, Name: "echo"})
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("cancel %v", err)
	}
}

func TestPromptStringArgsNotJSON(t *testing.T) {
	sess := &fakeSess{
		info:   caps(false, true, false),
		prompt: model.PromptResult{Messages: []model.PromptMessage{{Role: "user", Content: model.Content{Kind: model.ContentText, Text: "greet true"}}}},
	}
	_, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
		Kind:     invoke.KindPrompt,
		Name:     "greet",
		ArgPairs: []string{"name=true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(sess.promptA, []byte(`"name":"true"`)) {
		t.Fatalf("coerced args %s", sess.promptA)
	}
	_, err = invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
		Kind:     invoke.KindPrompt,
		Name:     "greet",
		ArgsJSON: `{"name":true}`,
	})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("%v", err)
	}
}

func TestReadTemplateAndBinaryBound(t *testing.T) {
	blob := bytes.Repeat([]byte{0xff}, 200)
	sess := &fakeSess{
		info:     caps(false, false, true),
		resource: model.ResourceBody{Contents: []model.ResourceContents{{URI: "fixture://items/a%20b", Text: "item:a%20b"}}},
	}
	_, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
		Kind:     invoke.KindRead,
		Template: "fixture://items/{id}",
		ArgPairs: []string{"id=a b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sess.readURI != "fixture://items/a%20b" {
		t.Fatalf("uri %s", sess.readURI)
	}
	sess.resource = model.ResourceBody{Contents: []model.ResourceContents{{URI: "blob", MIMEType: "application/octet-stream", Blob: blob}}}
	dir := t.TempDir()
	save := filepath.Join(dir, "out.bin")
	res, err := invoke.Run(context.Background(), fakeConn{sess: sess}, invoke.Request{
		Kind:     invoke.KindRead,
		URI:      "blob",
		MaxBytes: 50,
		SavePath: save,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Truncated || res.Resource.Contents[0].Blob != nil {
		t.Fatalf("stdout still has blob: %+v", res.Resource)
	}
	got, err := os.ReadFile(save)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("saved %d", len(got))
	}
}

func TestExpandReserved(t *testing.T) {
	got, err := invoke.ExpandURITemplate("fixture://x/{+path}", map[string]string{"path": "a/b"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "fixture://x/a/b" {
		t.Fatal(got)
	}
}
