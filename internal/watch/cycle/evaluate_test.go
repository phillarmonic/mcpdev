package cycle_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/watch/cycle"
)

type fakeStore struct {
	snaps []model.Snapshot
	errs  []error
	n     int
}

func (f *fakeStore) Capture(context.Context, port.Session, []model.PrimitiveKind) (model.Snapshot, error) {
	i := f.n
	f.n++
	if i < len(f.errs) && f.errs[i] != nil {
		return model.Snapshot{}, f.errs[i]
	}
	if i < len(f.snaps) {
		return f.snaps[i], nil
	}
	return model.Snapshot{}, errors.New("no snapshot")
}

func (f *fakeStore) Write(context.Context, string, model.Snapshot) error {
	return nil
}

func (f *fakeStore) Read(context.Context, string) (model.Snapshot, error) {
	return model.Snapshot{}, errors.New("unused")
}

type fakeSmoke struct {
	fail string
	ran  int
}

func (f *fakeSmoke) Run(_ context.Context, _ port.Session, names []string) error {
	f.ran++
	for _, n := range names {
		if n == f.fail {
			return errclass.Assertion{Err: errors.New("smoke failed: " + n)}
		}
	}
	return nil
}

type fakeSess struct{}

func (fakeSess) Era() model.ProtocolEra             { return model.Era20251125 }
func (fakeSess) DiscoverState() model.DiscoverState { return model.DiscoverNotApplicable }
func (fakeSess) Info() model.SessionInfo            { return model.SessionInfo{ServerName: "lab"} }
func (fakeSess) Inventory() port.InventoryReader    { return fakeSess{} }
func (fakeSess) CallTool(context.Context, string, json.RawMessage) (model.ToolResult, error) {
	return model.ToolResult{}, errors.New("unused")
}

func (fakeSess) GetPrompt(context.Context, string, json.RawMessage) (model.PromptResult, error) {
	return model.PromptResult{}, errors.New("unused")
}

func (fakeSess) ReadResource(context.Context, string) (model.ResourceBody, error) {
	return model.ResourceBody{}, errors.New("unused")
}

func (fakeSess) Close(context.Context) error { return nil }

func (fakeSess) List(context.Context, model.PrimitiveKind, model.Page) (model.CatalogPage, error) {
	return model.CatalogPage{}, nil
}

func snap(tools ...string) model.Snapshot {
	items := make([]model.Tool, 0, len(tools))
	for _, n := range tools {
		items = append(items, model.Tool{Name: n, InputSchema: json.RawMessage(`{"type":"object"}`)})
	}
	return model.Snapshot{
		SchemaVersion: 1,
		Server:        model.SnapshotServer{Name: "lab"},
		Tools: model.CatalogSection{
			Kind: model.KindTools, Status: model.SectionComplete, Tools: items, ItemCount: len(items),
		},
	}
}

func incompleteSnap() model.Snapshot {
	s := snap("echo")
	s.Incomplete = true
	s.IncompleteReasons = []string{model.ReasonListFailed}
	s.Tools.Status = model.SectionFailed
	return s
}

func newEng(t *testing.T, store *fakeStore, smoke *fakeSmoke, cfg cycle.Config, reports *[]cycle.Result) *cycle.Engine {
	t.Helper()
	opts := cycle.Options{Store: store, Differ: diff.Engine{}, Smoke: smoke, Config: cfg}
	if reports != nil {
		opts.OnReport = func(r cycle.Result) { *reports = append(*reports, r) }
	}
	return cycle.New(opts)
}

func TestInventoryReadinessSetsBaseline(t *testing.T) {
	e := newEng(t, &fakeStore{snaps: []model.Snapshot{snap("echo")}}, nil, cycle.Config{}, nil)
	res, err := e.Evaluate(context.Background(), fakeSess{})
	if err != nil || !res.Ready || res.Outcome != cycle.OutcomeUnchanged || !e.HasBaseline() {
		t.Fatalf("res %+v err %v baseline %v", res, err, e.HasBaseline())
	}
}

func TestIncompleteInventoryDoesNotReplaceBaseline(t *testing.T) {
	store := &fakeStore{snaps: []model.Snapshot{snap("echo"), incompleteSnap(), snap("echo", "sum")}}
	e := newEng(t, store, nil, cycle.Config{}, nil)
	if _, err := e.Evaluate(context.Background(), fakeSess{}); err != nil {
		t.Fatal(err)
	}
	_, err := e.Evaluate(context.Background(), fakeSess{})
	if errclass.ExitCode(err) != errclass.ExitIncomplete {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
	if !e.HasBaseline() {
		t.Fatal("lost baseline")
	}
	res, err := e.Evaluate(context.Background(), fakeSess{})
	if err != nil || res.Outcome != cycle.OutcomeAdditive {
		t.Fatalf("recovery %+v err %v", res, err)
	}
	h := e.History()
	if len(h) != 3 || h[1].Ready || h[1].Reason != "inventory" || !h[2].Ready {
		t.Fatalf("history %+v", h)
	}
}

func TestUnchangedRebuildDoesNotFlood(t *testing.T) {
	var reports []cycle.Result
	store := &fakeStore{snaps: []model.Snapshot{snap("echo"), snap("echo"), snap("echo")}}
	e := newEng(t, store, nil, cycle.Config{}, &reports)
	for range 3 {
		if _, err := e.Evaluate(context.Background(), fakeSess{}); err != nil {
			t.Fatal(err)
		}
	}
	if len(reports) != 1 {
		t.Fatalf("reports %d", len(reports))
	}
	h := e.History()
	if !h[1].Ready || e.History()[2].Ready != true {
		t.Fatalf("history %+v", h)
	}
}

func TestBreakingSchemaIsReportedAndPromoted(t *testing.T) {
	store := &fakeStore{snaps: []model.Snapshot{
		snap("echo"),
		func() model.Snapshot {
			s := snap("echo")
			s.Tools.Tools[0].InputSchema = json.RawMessage(`{"type":"object","required":["q"]}`)
			return s
		}(),
	}}
	e := newEng(t, store, nil, cycle.Config{}, nil)
	if _, err := e.Evaluate(context.Background(), fakeSess{}); err != nil {
		t.Fatal(err)
	}
	res, err := e.Evaluate(context.Background(), fakeSess{})
	if err != nil || !res.Ready || res.Outcome != cycle.OutcomeBreaking {
		t.Fatalf("%+v err %v", res, err)
	}
}

func TestSmokeFailureKeepsBaseline(t *testing.T) {
	store := &fakeStore{snaps: []model.Snapshot{snap("echo"), snap("echo", "sum")}}
	smoke := &fakeSmoke{fail: "ci"}
	e := newEng(t, store, smoke, cycle.Config{Smoke: []string{"ci"}}, nil)
	if _, err := e.Evaluate(context.Background(), fakeSess{}); err == nil {
		t.Fatal("expected first smoke to fail before baseline")
	}
	if e.HasBaseline() {
		t.Fatal("smoke failure became baseline")
	}
	smoke.fail = ""
	if _, err := e.Evaluate(context.Background(), fakeSess{}); err != nil || !e.HasBaseline() {
		t.Fatalf("recovery err %v", err)
	}
}

func TestScenarioReadinessRunsBeforeCapture(t *testing.T) {
	store := &fakeStore{snaps: []model.Snapshot{snap("echo")}}
	smoke := &fakeSmoke{fail: "boot"}
	e := newEng(t, store, smoke, cycle.Config{Readiness: cycle.ReadinessScenario, ReadinessScenarios: []string{"boot"}}, nil)
	_, err := e.Evaluate(context.Background(), fakeSess{})
	if errclass.ExitCode(err) != errclass.ExitAssertion {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
	if store.n != 0 {
		t.Fatal("capture ran after failed readiness scenario")
	}
	if e.HasBaseline() {
		t.Fatal("baseline set")
	}
}

func TestHistoryIsBounded(t *testing.T) {
	store := &fakeStore{snaps: []model.Snapshot{incompleteSnap(), incompleteSnap(), incompleteSnap()}}
	e := newEng(t, store, nil, cycle.Config{HistoryBound: 2}, nil)
	for range 3 {
		_, _ = e.Evaluate(context.Background(), fakeSess{})
	}
	if len(e.History()) != 2 {
		t.Fatalf("len %d", len(e.History()))
	}
}

func TestWrapProtocolFailureKeepsBaseline(t *testing.T) {
	store := &fakeStore{snaps: []model.Snapshot{snap("echo")}}
	e := newEng(t, store, nil, cycle.Config{}, nil)
	if _, err := e.Evaluate(context.Background(), fakeSess{}); err != nil {
		t.Fatal(err)
	}
	hs := func(context.Context, port.Proc) (port.Session, error) {
		return nil, errclass.Protocol{Err: errors.New("initialize rejected")}
	}
	_, err := e.Wrap(hs)(context.Background(), nil)
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("exit %d err=%v", errclass.ExitCode(err), err)
	}
	if !e.HasBaseline() {
		t.Fatal("protocol failure cleared baseline")
	}
	h := e.History()
	if h[len(h)-1].Reason != "protocol" || h[len(h)-1].Ready {
		t.Fatalf("history %+v", h)
	}
}
