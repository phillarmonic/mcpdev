package watchrender

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/watch"
)

func sample() watch.Note {
	return watch.Note{
		At:         time.Date(2026, 9, 19, 5, 0, 0, 0, time.UTC),
		Kind:       watch.KindBuild,
		ID:         "e1",
		State:      "ok",
		Class:      "ok",
		Generation: 2,
	}
}

func TestJSONLAppendOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := JSONL(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	if err := JSONL(&buf, watch.Note{
		At:    time.Date(2026, 9, 19, 5, 0, 1, 0, time.UTC),
		Kind:  watch.KindShutdown,
		ID:    "e2",
		State: "stopped",
	}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines %d %q", len(lines), buf.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first["kind"] != "build" || first["id"] != "e1" || first["state"] != "ok" {
		t.Fatalf("%v", first)
	}
	if bytes.Contains(buf.Bytes(), []byte{0x1b}) {
		t.Fatal("ANSI in JSONL")
	}
	if bytes.Contains(buf.Bytes(), []byte{'\r'}) {
		t.Fatal("CR in JSONL")
	}
}

func TestHumanAppendOnlyWithoutTTY(t *testing.T) {
	var buf bytes.Buffer
	w := Writer{Out: &buf, Mode: Detect(&buf, nil)}
	if w.Mode.Color || w.Mode.InPlace {
		t.Fatalf("buffer treated as TTY %#v", w.Mode)
	}
	if err := w.Write(sample()); err != nil {
		t.Fatal(err)
	}
	running := sample()
	running.State = "running"
	if err := w.Write(running); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Count(out, "\n") != 2 {
		t.Fatalf("expected two lines %q", out)
	}
	if strings.Contains(out, "\r") || strings.Contains(out, "\x1b") {
		t.Fatalf("in-place or color on buffer %q", out)
	}
	if !strings.Contains(out, "e1 build ok") {
		t.Fatalf("%q", out)
	}
}

func TestNOCOLORDisablesColor(t *testing.T) {
	mode := Detect(os.Stdout, []string{"NO_COLOR=1", "TERM=xterm"})
	if mode.Color || mode.InPlace {
		t.Fatalf("%#v", mode)
	}
}

func TestInPlaceThenNewline(t *testing.T) {
	var buf bytes.Buffer
	w := Writer{Out: &buf, Mode: Mode{InPlace: true}}
	running := sample()
	running.State = "running"
	if err := w.Write(running); err != nil {
		t.Fatal(err)
	}
	done := sample()
	if err := w.Write(done); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\r") {
		t.Fatalf("expected CR %q", out)
	}
	if !strings.Contains(out, "e1 build ok") {
		t.Fatalf("%q", out)
	}
}
