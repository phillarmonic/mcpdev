package trace

import (
	"bytes"
	"io"
	"testing"
)

func TestWrapEmitsOneFramePerLine(t *testing.T) {
	rec := New(Options{Clock: &stepClock{}})
	var buf bytes.Buffer
	w := WrapWriter(nopWriteCloserBuffer{&buf}, rec, "stdio")
	payload := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\n{\"jsonrpc\":\"2.0\",\"id\":2}\n"
	if _, err := io.WriteString(w, payload); err != nil {
		t.Fatal(err)
	}
	r := WrapReader(io.NopCloser(bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`+"\n"))), rec, "stdio")
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("result")) {
		t.Fatalf("passthrough %s", got)
	}
	ev := rec.Events()
	if len(ev) != 3 {
		t.Fatalf("events %d %+v", len(ev), ev)
	}
	if ev[0].Direction != DirClient || ev[0].Method != "ping" {
		t.Fatalf("write0 %+v", ev[0])
	}
	if ev[2].Direction != DirServer || ev[2].RequestID != "1" {
		t.Fatalf("read %+v", ev[2])
	}
}

type nopWriteCloserBuffer struct{ *bytes.Buffer }

func (nopWriteCloserBuffer) Close() error { return nil }
