package diff_test

import (
	"context"
	"errors"
	"testing"

	diffapp "github.com/phillarmonic/mcpdev/internal/app/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

type fakeStore struct {
	files map[string]model.Snapshot
}

func (f fakeStore) Capture(context.Context, port.Session, []model.PrimitiveKind) (model.Snapshot, error) {
	return model.Snapshot{}, errclass.Internal{Err: errors.New("unused")}
}

func (f fakeStore) Write(context.Context, string, model.Snapshot) error {
	return errclass.Internal{Err: errors.New("unused")}
}

func (f fakeStore) Read(_ context.Context, path string) (model.Snapshot, error) {
	s, ok := f.files[path]
	if !ok {
		return model.Snapshot{}, errclass.Usage{Msg: "missing " + path}
	}
	return s, nil
}

type fakeDiffer struct{ out model.Diff }

func (f fakeDiffer) Diff(context.Context, model.Snapshot, model.Snapshot) (model.Diff, error) {
	return f.out, nil
}

type unusedConn struct{}

func (unusedConn) Connect(context.Context, port.Target) (port.Session, error) {
	return nil, errclass.Connection{}
}

func TestFileEqual(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {SchemaVersion: 1}, "b": {SchemaVersion: 1}}}
	res, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictEqual, Changes: []model.Change{},
	}}, diffapp.Request{
		From:   diffapp.Source{Path: "a", Label: "a"},
		To:     diffapp.Source{Path: "b", Label: "b"},
		FailOn: model.SeverityAdditive, ExitOnChange: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != diffapp.OutcomeNoChange {
		t.Fatal(res.Outcome)
	}
}

func TestThresholdFailure(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {}, "b": {}}}
	_, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictChanged, Changed: true,
		Changes: []model.Change{{
			Path: "/tools/echo", Category: model.CategoryRemoval,
			Severity: model.SeverityBreaking, Rationale: "gone",
		}},
	}}, diffapp.Request{
		From: diffapp.Source{Path: "a"}, To: diffapp.Source{Path: "b"},
		FailOn: model.SeverityAdditive, ExitOnChange: true,
	})
	if errclass.ExitCode(err) != errclass.ExitInterfaceChange {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestMetadataBelowThreshold(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {}, "b": {}}}
	res, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictChanged, Changed: true,
		Changes: []model.Change{{
			Path: "/tools/echo/title", Category: model.CategoryChange,
			Severity: model.SeverityMetadata, Rationale: "title",
		}},
	}}, diffapp.Request{
		From: diffapp.Source{Path: "a"}, To: diffapp.Source{Path: "b"},
		FailOn: model.SeverityAdditive, ExitOnChange: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != diffapp.OutcomeBelowThreshold {
		t.Fatal(res.Outcome)
	}
}

func TestIgnoreMetadataPath(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {}, "b": {}}}
	res, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictChanged, Changed: true,
		Changes: []model.Change{{
			Path: "/tools/echo/title", Category: model.CategoryChange,
			Severity: model.SeverityMetadata, Rationale: "title",
		}},
	}}, diffapp.Request{
		From: diffapp.Source{Path: "a"}, To: diffapp.Source{Path: "b"},
		Ignore: []string{"/tools/echo/title"},
		FailOn: model.SeverityMetadata, ExitOnChange: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != diffapp.OutcomeNoChange || len(res.Ignored) != 1 {
		t.Fatalf("%s ignored=%d", res.Outcome, len(res.Ignored))
	}
}

func TestIndeterminate(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {}, "b": {}}}
	_, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictIndeterminate, Changed: false,
		Changes: []model.Change{{Path: "/incomplete", Severity: model.SeverityUnknown, Rationale: "unread"}},
	}}, diffapp.Request{
		From: diffapp.Source{Path: "a"}, To: diffapp.Source{Path: "b"},
		FailOn: model.SeverityAdditive, ExitOnChange: true,
	})
	if errclass.ExitCode(err) != errclass.ExitIncomplete {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
}

func TestAllowIncomplete(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {}, "b": {}}}
	res, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictIndeterminate,
		Changes: []model.Change{{Path: "/incomplete", Severity: model.SeverityUnknown, Rationale: "unread"}},
	}}, diffapp.Request{
		From: diffapp.Source{Path: "a"}, To: diffapp.Source{Path: "b"},
		AllowIncomplete: true, FailOn: model.SeverityAdditive, ExitOnChange: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != diffapp.OutcomeIndeterminate {
		t.Fatal(res.Outcome)
	}
}

func TestNoExitOnChange(t *testing.T) {
	store := fakeStore{files: map[string]model.Snapshot{"a": {}, "b": {}}}
	res, err := diffapp.Run(context.Background(), unusedConn{}, store, fakeDiffer{out: model.Diff{
		Verdict: model.VerdictChanged, Changed: true,
		Changes: []model.Change{{Path: "/tools/x", Category: model.CategoryAddition, Severity: model.SeverityAdditive, Rationale: "add"}},
	}}, diffapp.Request{
		From: diffapp.Source{Path: "a"}, To: diffapp.Source{Path: "b"},
		FailOn: model.SeverityAdditive, ExitOnChange: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != diffapp.OutcomeThresholdFailure {
		t.Fatal(res.Outcome)
	}
}
