package protocol_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"testing/synctest"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/protocol"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestConnectLegacyStdioUsesModelTypes(t *testing.T) {
	lab := startNamed(t, "legacy-stdio")
	sess := connectLab(t, lab, protocol.Options{Identity: model.Identity{Name: "mcpdev-test", Version: "dev"}})
	if sess.Era() != model.Era20251125 {
		t.Fatalf("era: %q", sess.Era())
	}
	if sess.DiscoverState() != model.DiscoverSkipped {
		t.Fatalf("discover: %q", sess.DiscoverState())
	}
	info := sess.Info()
	if info.ServerName != "mcpdev-fixture" {
		t.Fatalf("server: %+v", info)
	}
	if info.Handshake != model.HandshakeInitialize {
		t.Fatalf("handshake: %q", info.Handshake)
	}
	if info.Instructions == "" || !info.Capabilities.Tools {
		t.Fatalf("summary: %+v", info)
	}
	page, err := sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasTool(page.Tools, "echo") {
		t.Fatalf("tools: %+v", page.Tools)
	}
	echo, err := sess.CallTool(context.Background(), "echo", json.RawMessage(`{"text":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	if echo.IsError {
		t.Fatalf("echo isError: %+v", echo)
	}
	prompt, err := sess.GetPrompt(context.Background(), "greet", json.RawMessage(`{"name":"lab"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(prompt.Messages) == 0 {
		t.Fatal("expected prompt messages")
	}
	body, err := sess.ReadResource(context.Background(), "fixture://stable/note")
	if err != nil {
		t.Fatal(err)
	}
	if len(body.Contents) == 0 || body.Contents[0].Text != "stable-note" {
		t.Fatalf("resource: %+v", body)
	}
}

func TestConnectModernStdioDiscoverOK(t *testing.T) {
	lab := startNamed(t, "modern-stdio")
	sess := connectLab(t, lab, protocol.Options{})
	if sess.Era() != model.Era20260728 {
		t.Fatalf("era: %q", sess.Era())
	}
	if sess.DiscoverState() != model.DiscoverOK {
		t.Fatalf("discover: %q", sess.DiscoverState())
	}
	if sess.Info().Handshake != model.HandshakeDiscover {
		t.Fatalf("handshake: %q", sess.Info().Handshake)
	}
	page, err := sess.Inventory().List(context.Background(), model.KindPrompts, model.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Prompts) == 0 {
		t.Fatal("expected prompts")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	lab := startNamed(t, "legacy-stdio")
	sess := connectLab(t, lab, protocol.Options{})
	ctx := context.Background()
	if err := sess.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := sess.Close(ctx); err != nil {
		t.Fatalf("second close: %v", err)
	}
	_, err := sess.CallTool(ctx, "echo", nil)
	if err == nil {
		t.Fatal("expected error after close")
	}
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestConnectCancelStopsHandshake(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lab := startNamed(t, "delay-stdio")
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := protocol.Connect(ctx, protocol.Stream{Reader: lab.ClientOut, Writer: lab.ClientIn}, protocol.Options{})
			done <- err
		}()
		synctest.Wait()
		cancel()
		err := <-done
		if err == nil {
			t.Fatal("expected canceled connect")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
		if errclass.ExitCode(err) != errclass.ExitConnection {
			t.Fatalf("class: %d", errclass.ExitCode(err))
		}
	})
}

func TestCallToolCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lab := startNamed(t, "delay-stdio")
		sess := connectLab(t, lab, protocol.Options{PreferEra: model.Era20251125})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := sess.CallTool(ctx, "echo", json.RawMessage(`{"text":"hi"}`))
			done <- err
		}()
		synctest.Wait()
		cancel()
		err := <-done
		if err == nil {
			t.Fatal("expected canceled call")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	})
}

func TestMissingStreamIsUsage(t *testing.T) {
	_, err := protocol.Connect(context.Background(), protocol.Stream{}, protocol.Options{})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
}

func startNamed(t *testing.T, name string) *testkit.StdioLab {
	t.Helper()
	d, err := testkit.Named(name)
	if err != nil {
		t.Fatal(err)
	}
	lab, err := testkit.StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := lab.Close(); closeErr != nil && !errors.Is(closeErr, io.ErrClosedPipe) && !errors.Is(closeErr, io.EOF) {
			t.Errorf("close fixture: %v", closeErr)
		}
	})
	return lab
}

func connectLab(t *testing.T, lab *testkit.StdioLab, opts protocol.Options) port.Session {
	t.Helper()
	sess, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: lab.ClientIn,
	}, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sess.Close(context.Background()); closeErr != nil {
			t.Errorf("close session: %v", closeErr)
		}
	})
	return sess
}

func hasTool(tools []model.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
