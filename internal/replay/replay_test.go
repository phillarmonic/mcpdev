package replay_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/replay"
)

func TestLegacyFixtureInspect(t *testing.T) {
	conn := replay.File{Path: fixture("legacy-init.json")}
	sess, err := conn.Connect(context.Background(), port.Target{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close(context.Background()) }()
	if sess.Info().Handshake != model.HandshakeInitialize {
		t.Fatalf("handshake %q", sess.Info().Handshake)
	}
	page, err := sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Tools) != 1 || page.Tools[0].Name != "echo" {
		t.Fatalf("%+v", page.Tools)
	}
}

func TestPaginationCursor(t *testing.T) {
	conn := replay.File{Path: fixture("pagination.json")}
	sess, err := conn.Connect(context.Background(), port.Target{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close(context.Background()) }()
	first, err := sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextCursor != "page-2" {
		t.Fatalf("cursor %q", first.NextCursor)
	}
	second, err := sess.Inventory().List(context.Background(), model.KindTools, model.Page{Cursor: "page-2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Tools) != 1 || second.Tools[0].Name != "beta" {
		t.Fatalf("%+v", second.Tools)
	}
}

func TestMismatchNamesPayloadPath(t *testing.T) {
	conn := replay.File{Path: fixture("pagination.json")}
	sess, err := conn.Connect(context.Background(), port.Target{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close(context.Background()) }()
	_, _ = sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	_, err = sess.Inventory().List(context.Background(), model.KindTools, model.Page{Cursor: "wrong"})
	var m replay.Mismatch
	if !errors.As(err, &m) {
		t.Fatalf("err %v", err)
	}
	if m.Field != "payload" || m.Path != "/cursor" {
		t.Fatalf("field=%q path=%q err=%v", m.Field, m.Path, err)
	}
	if !strings.Contains(err.Error(), "/cursor") {
		t.Fatalf("error %v", err)
	}
}

func TestSanitizeSecrets(t *testing.T) {
	doc, err := replay.Load(fixture("legacy-init.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc.Steps[0].Expect.Params = []byte(`{"authorization":"s3cret","name":"echo"}`)
	out := replay.Sanitize(doc, nil)
	if strings.Contains(string(out.Steps[0].Expect.Params), "s3cret") {
		t.Fatalf("leaked: %s", out.Steps[0].Expect.Params)
	}
	if !strings.Contains(string(out.Steps[0].Expect.Params), "[REDACTED]") {
		t.Fatalf("missing redaction: %s", out.Steps[0].Expect.Params)
	}
}

func TestProcessExit(t *testing.T) {
	conn := replay.File{Path: fixture("process-exit.json")}
	sess, err := conn.Connect(context.Background(), port.Target{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close(context.Background()) }()
	_, err = sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("code %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestHTTPSSEvents(t *testing.T) {
	conn := replay.File{Path: fixture("http-sse.json")}
	sess, err := conn.Connect(context.Background(), port.Target{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close(context.Background()) }()
	_, err = sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
	if err != nil {
		t.Fatal(err)
	}
	ev := sess.(interface{ Events() []replay.Delivery }).Events()
	if len(ev) != 1 || ev[0].Kind != replay.KindSSE {
		t.Fatalf("%+v", ev)
	}
}

func TestDelayNoSleep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		doc, err := replay.Load(fixture("legacy-init.json"))
		if err != nil {
			t.Fatal(err)
		}
		doc.Steps[0].Deliver[0].DelayMS = 50
		sess := replayForTest(t, doc, clock.Wall{})
		_, err = sess.Inventory().List(context.Background(), model.KindTools, model.Page{})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestCancelDuringDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		doc, err := replay.Load(fixture("legacy-init.json"))
		if err != nil {
			t.Fatal(err)
		}
		doc.Steps[0].Deliver[0].DelayMS = 50
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		sess := replayForTest(t, doc, clock.Wall{})
		_, err = sess.Inventory().List(ctx, model.KindTools, model.Page{})
		if errclass.ExitCode(err) != errclass.ExitConnection {
			t.Fatalf("err %v", err)
		}
	})
}

func TestVirtualAdvance(t *testing.T) {
	clk := clock.NewVirtual(time.Unix(0, 0).UTC())
	ch := clk.After(time.Second)
	select {
	case <-ch:
		t.Fatal("fired early")
	default:
	}
	clk.Advance(time.Second)
	select {
	case got := <-ch:
		if got.Unix() != 1 {
			t.Fatalf("now %v", got)
		}
	default:
		t.Fatal("missing fire")
	}
}

func TestMalformedStream(t *testing.T) {
	doc, err := replay.Load(fixture("malformed-frame.json"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := replay.OpenStream(context.Background(), doc, clock.Wall{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Reader.Close() }()
	defer func() { _ = st.Writer.Close() }()
	_, _ = st.Writer.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"))
	buf := make([]byte, 64)
	n, err := st.Reader.Read(buf)
	if err != nil && n == 0 {
		t.Fatal(err)
	}
	if !strings.Contains(string(buf[:n]), "{not-json") {
		t.Fatalf("got %q", buf[:n])
	}
}

func TestDiscoverWire(t *testing.T) {
	doc, err := replay.Load(fixture("modern-discover.json"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := replay.OpenStream(context.Background(), doc, clock.Wall{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Reader.Close() }()
	defer func() { _ = st.Writer.Close() }()
	_, _ = st.Writer.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{}}` + "\n"))
	buf := make([]byte, 256)
	n, err := st.Reader.Read(buf)
	if err != nil && n == 0 {
		t.Fatal(err)
	}
	if !strings.Contains(string(buf[:n]), "2026-07-28") {
		t.Fatalf("got %q", buf[:n])
	}
}

func TestRecordingWritesSanitizedFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "saved.json")
	inner := replay.File{Path: fixture("legacy-init.json")}
	sess, err := replay.Recording{Inner: inner, Path: out}.Connect(context.Background(), port.Target{Kind: "stdio"})
	if err != nil {
		t.Fatal(err)
	}
	if _, listErr := sess.Inventory().List(context.Background(), model.KindTools, model.Page{}); listErr != nil {
		t.Fatal(listErr)
	}
	if closeErr := sess.Close(context.Background()); closeErr != nil {
		t.Fatal(closeErr)
	}
	doc, err := replay.Load(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Steps) != 1 || doc.Steps[0].Expect.Method != "tools/list" {
		t.Fatalf("%+v", doc.Steps)
	}
}

func TestRecordRequiresPath(t *testing.T) {
	_, err := replay.Recording{Inner: replay.File{Path: fixture("legacy-init.json")}}.Connect(context.Background(), port.Target{})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("err %v", err)
	}
}

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", "replays", name)
}

func replayForTest(t *testing.T, doc replay.Document, clk port.Clock) port.Session {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "in.json")
	if err := replay.Save(path, doc, nil); err != nil {
		t.Fatal(err)
	}
	sess, err := replay.File{Path: path, Clock: clk}.Connect(context.Background(), port.Target{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sess.Close(context.Background()) })
	return sess
}
