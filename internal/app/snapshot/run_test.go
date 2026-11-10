package snapshot_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/app/snapshot"
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

type fakeSess struct{ closed bool }

func (s *fakeSess) Era() model.ProtocolEra             { return model.Era20251125 }
func (s *fakeSess) DiscoverState() model.DiscoverState { return model.DiscoverNotApplicable }
func (s *fakeSess) Info() model.SessionInfo            { return model.SessionInfo{ServerName: "lab"} }
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

func (s *fakeSess) List(context.Context, model.PrimitiveKind, model.Page) (model.CatalogPage, error) {
	return model.CatalogPage{}, nil
}

type fakeStore struct {
	wrote []string
	snap  model.Snapshot
}

func (f *fakeStore) Capture(context.Context, port.Session, []model.PrimitiveKind) (model.Snapshot, error) {
	return f.snap, nil
}

func (f *fakeStore) Write(_ context.Context, path string, _ model.Snapshot) error {
	f.wrote = append(f.wrote, path)
	return nil
}

func (f *fakeStore) Read(context.Context, string) (model.Snapshot, error) {
	return f.snap, nil
}

func TestRunWritesComplete(t *testing.T) {
	sess := &fakeSess{}
	store := &fakeStore{snap: model.Snapshot{SchemaVersion: 1, Server: model.SnapshotServer{Name: "lab"}}}
	dir := t.TempDir()
	out := filepath.Join(dir, "snap.json")
	res, err := snapshot.Run(context.Background(), fakeConn{sess: sess}, store, snapshot.Request{Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if res.Out != out || len(store.wrote) != 1 {
		t.Fatalf("write %+v out=%s", store.wrote, res.Out)
	}
	if !sess.closed {
		t.Fatal("session not closed")
	}
}

func TestRunDoesNotReplaceBaselineWhenIncomplete(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(out, []byte(`{"keep":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{snap: model.Snapshot{
		SchemaVersion:     1,
		Incomplete:        true,
		IncompleteReasons: []string{model.ReasonPaginationStopped},
	}}
	_, err := snapshot.Run(context.Background(), fakeConn{sess: &fakeSess{}}, store, snapshot.Request{Out: out})
	if errclass.ExitCode(err) != errclass.ExitIncomplete {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
	if len(store.wrote) != 0 {
		t.Fatalf("replaced baseline: %v", store.wrote)
	}
	body, _ := os.ReadFile(out)
	if string(body) != `{"keep":true}` {
		t.Fatalf("baseline mutated: %s", body)
	}
}

func TestRunWritesNewIncompleteFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "partial.json")
	store := &fakeStore{snap: model.Snapshot{
		SchemaVersion:     1,
		Incomplete:        true,
		IncompleteReasons: []string{model.ReasonPaginationStopped},
	}}
	res, err := snapshot.Run(context.Background(), fakeConn{sess: &fakeSess{}}, store, snapshot.Request{Out: out})
	if errclass.ExitCode(err) != errclass.ExitIncomplete {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
	if res.Out != out {
		t.Fatal("expected incomplete document to be written to a new path")
	}
}
