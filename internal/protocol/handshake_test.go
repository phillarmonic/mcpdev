package protocol_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/protocol"
)

func TestLegacyHandshakeOrderAndSummary(t *testing.T) {
	lab := startNamed(t, "legacy-stdio")
	tap := newMethodTap(lab.ClientIn)
	sess, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: tap,
	}, protocol.Options{PreferEra: model.Era20251125, Identity: model.Identity{Name: "mcpdev-test", Version: "dev"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sess.Close(context.Background()); closeErr != nil {
			t.Errorf("close session: %v", closeErr)
		}
	})
	if !slices.Equal(tap.methods(), []string{"initialize", "notifications/initialized"}) {
		t.Fatalf("methods: %v", tap.methods())
	}
	info := sess.Info()
	if info.Handshake != model.HandshakeInitialize {
		t.Fatalf("handshake: %q", info.Handshake)
	}
	if info.ProtocolVersion != string(model.Era20251125) {
		t.Fatalf("version: %q", info.ProtocolVersion)
	}
	if info.Instructions != "mcpdev fixture laboratory" {
		t.Fatalf("instructions: %q", info.Instructions)
	}
	if info.ServerTitle != "Fixture Lab" {
		t.Fatalf("title: %q", info.ServerTitle)
	}
	if !info.Capabilities.Tools || !info.Capabilities.Prompts || !info.Capabilities.Resources {
		t.Fatalf("caps: %+v", info.Capabilities)
	}
	if !info.Capabilities.ToolsListChanged || !info.Capabilities.ResourceSubscribe {
		t.Fatalf("cap flags: %+v", info.Capabilities)
	}
	page, err := sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasTool(page.Tools, "echo") {
		t.Fatalf("tools after handshake: %+v", page.Tools)
	}
}

func TestLegacyHandshakeDowngradeFromDiscover(t *testing.T) {
	lab := startNamed(t, "legacy-stdio")
	tap := newMethodTap(lab.ClientIn)
	sess, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: tap,
	}, protocol.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sess.Close(context.Background())
	})
	got := tap.methods()
	if !slices.Contains(got, "server/discover") || !slices.Contains(got, "initialize") || !slices.Contains(got, "notifications/initialized") {
		t.Fatalf("downgrade methods: %v", got)
	}
	i := slices.Index(got, "initialize")
	j := slices.Index(got, "notifications/initialized")
	if i < 0 || j < i {
		t.Fatalf("initialized before initialize: %v", got)
	}
	if sess.Era() != model.Era20251125 || sess.DiscoverState() != model.DiscoverSkipped {
		t.Fatalf("era=%q discover=%q", sess.Era(), sess.DiscoverState())
	}
	if sess.Info().Handshake != model.HandshakeInitialize {
		t.Fatalf("handshake: %q", sess.Info().Handshake)
	}
	if !sess.Info().Fallback || sess.Info().RequestedVersion != string(model.Era20260728) {
		t.Fatalf("fallback diagnostics: %+v", sess.Info())
	}
}

func TestConnectHandshakeTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lab := startNamed(t, "delay-stdio")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := protocol.Connect(ctx, protocol.Stream{Reader: lab.ClientOut, Writer: lab.ClientIn}, protocol.Options{PreferEra: model.Era20251125})
			done <- err
		}()
		synctest.Wait()
		synctest.Sleep(10 * time.Millisecond)
		err := <-done
		if err == nil {
			t.Fatal("expected handshake timeout")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want deadline, got %v", err)
		}
		if errclass.ExitCode(err) != errclass.ExitConnection {
			t.Fatalf("class: %d", errclass.ExitCode(err))
		}
	})
}

func TestConnectMalformedInitializeIsProtocol(t *testing.T) {
	lab := startNamed(t, "malformed-stdio")
	_, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: lab.ClientIn,
	}, protocol.Options{PreferEra: model.Era20251125})
	if err == nil {
		t.Fatal("expected malformed handshake error")
	}
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestConnectMalformedCapabilitiesIsProtocol(t *testing.T) {
	lab := startNamed(t, "legacy-bad-caps")
	_, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: lab.ClientIn,
	}, protocol.Options{PreferEra: model.Era20251125})
	if err == nil {
		t.Fatal("expected capability decode error")
	}
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestConnectUnsupportedVersionIsProtocol(t *testing.T) {
	lab := startNamed(t, "legacy-unsupported")
	_, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: lab.ClientIn,
	}, protocol.Options{PreferEra: model.Era20251125})
	if err == nil {
		t.Fatal("expected unsupported version")
	}
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestConnectServerExitDuringInitializeIsConnection(t *testing.T) {
	lab := startNamed(t, "legacy-exit")
	_, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: lab.ClientIn,
	}, protocol.Options{PreferEra: model.Era20251125})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
}

type methodTap struct {
	io.WriteCloser
	buf []byte
	got []string
	mu  sync.Mutex
}

func newMethodTap(w io.WriteCloser) *methodTap {
	return &methodTap{WriteCloser: w}
}

func (t *methodTap) Write(p []byte) (int, error) {
	n, err := t.WriteCloser.Write(p)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p[:n]...)
	for {
		i := bytes.IndexByte(t.buf, '\n')
		if i < 0 {
			break
		}
		line := t.buf[:i]
		t.buf = t.buf[i+1:]
		var msg struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(line, &msg) == nil && msg.Method != "" {
			t.got = append(t.got, msg.Method)
		}
	}
	return n, err
}

func (t *methodTap) methods() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.got)
}
