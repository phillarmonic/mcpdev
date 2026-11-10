package diff

import (
	"bytes"
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
)

type goldenCase struct {
	mutTo   func(*model.Snapshot)
	mutFrom func(*model.Snapshot)
	name    string
	file    string
	verdict model.DiffVerdict
	path    string
	cat     model.DiffCategory
	sev     model.DiffSeverity
	engine  Engine
}

func TestRulesAndGoldens(t *testing.T) {
	cases := []goldenCase{
		{name: "equal", file: "equal.json", verdict: model.VerdictEqual},
		{
			name: "incomplete", file: "incomplete.json", verdict: model.VerdictIndeterminate,
			path: "/incomplete", cat: model.CategoryChange, sev: model.SeverityUnknown,
			mutFrom: func(s *model.Snapshot) {
				s.Incomplete = true
				s.IncompleteReasons = []string{model.ReasonPaginationStopped}
				s.Tools.Status = model.SectionIncomplete
				s.Tools.Reasons = []string{model.ReasonPaginationStopped}
			},
		},
		{
			name: "tool-removed", file: "tool-removed.json", verdict: model.VerdictChanged,
			path: "/tools/echo", cat: model.CategoryRemoval, sev: model.SeverityBreaking,
			mutTo: func(s *model.Snapshot) { s.Tools.Tools = nil; s.Tools.ItemCount = 0 },
		},
		{
			name: "required-input-added", file: "required-input-added.json", verdict: model.VerdictChanged,
			path: "/tools/echo/input_schema/properties/n", cat: model.CategoryAddition, sev: model.SeverityBreaking,
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"properties":{"n":{"type":"number"},"text":{"type":"string"}},"required":["n"],"type":"object"}`)
			},
		},
		{
			name: "type-narrowing", file: "type-narrowing.json", verdict: model.VerdictChanged,
			path: "/tools/echo/input_schema/properties/text/type", cat: model.CategoryChange, sev: model.SeverityBreaking,
			mutFrom: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"properties":{"text":{"type":["string","null"]}},"type":"object"}`)
			},
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"properties":{"text":{"type":"string"}},"type":"object"}`)
			},
		},
		{
			name: "enum-narrowing", file: "enum-narrowing.json", verdict: model.VerdictChanged,
			path: "/tools/echo/input_schema/properties/text/enum", cat: model.CategoryChange, sev: model.SeverityBreaking,
			mutFrom: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"properties":{"text":{"enum":["a","b"],"type":"string"}},"type":"object"}`)
			},
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"properties":{"text":{"enum":["a"],"type":"string"}},"type":"object"}`)
			},
		},
		{
			name: "output-property-removed", file: "output-property-removed.json", verdict: model.VerdictChanged,
			path: "/tools/echo/output_schema/properties/ok", cat: model.CategoryRemoval, sev: model.SeverityBreaking,
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].OutputSchema = json.RawMessage(`{"properties":{},"type":"object"}`)
			},
		},
		{
			name: "optional-input-added", file: "optional-input-added.json", verdict: model.VerdictChanged,
			path: "/tools/echo/input_schema/properties/flag", cat: model.CategoryAddition, sev: model.SeverityAdditive,
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"properties":{"flag":{"type":"boolean"},"text":{"type":"string"}},"type":"object"}`)
			},
		},
		{
			name: "tool-added", file: "tool-added.json", verdict: model.VerdictChanged,
			path: "/tools/ping", cat: model.CategoryAddition, sev: model.SeverityAdditive,
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools = append(s.Tools.Tools, model.Tool{Name: "ping", InputSchema: json.RawMessage(`{"type":"object"}`)})
				s.Tools.ItemCount = 2
			},
		},
		{
			name: "metadata-title", file: "metadata-title.json", verdict: model.VerdictChanged,
			path: "/tools/echo/title", cat: model.CategoryChange, sev: model.SeverityMetadata,
			mutTo: func(s *model.Snapshot) { s.Tools.Tools[0].Title = "Echo v2" },
		},
		{
			name: "rename-unmapped", file: "rename-unmapped.json", verdict: model.VerdictChanged,
			path: "/tools/echo", cat: model.CategoryRemoval, sev: model.SeverityBreaking,
			mutTo: func(s *model.Snapshot) { s.Tools.Tools[0].Name = "echo2" },
		},
		{
			name: "rename-mapped", file: "rename-mapped.json", verdict: model.VerdictChanged,
			path: "/tools/echo2/name", cat: model.CategoryChange, sev: model.SeverityBreaking,
			engine: Engine{Renames: []model.Rename{{Kind: model.KindTools, From: "echo", To: "echo2"}}},
			mutTo:  func(s *model.Snapshot) { s.Tools.Tools[0].Name = "echo2" },
		},
		{
			name: "additionalProperties-unknown", file: "additional-properties-unknown.json", verdict: model.VerdictChanged,
			path: "/tools/echo/input_schema/additionalProperties", cat: model.CategoryChange, sev: model.SeverityUnknown,
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].InputSchema = json.RawMessage(`{"additionalProperties":false,"properties":{"text":{"type":"string"}},"type":"object"}`)
			},
		},
		{
			name: "ordering", file: "ordering.json", verdict: model.VerdictChanged,
			path: "/tools/echo/title", cat: model.CategoryChange, sev: model.SeverityMetadata,
			mutTo: func(s *model.Snapshot) {
				s.Tools.Tools[0].Title = "Echo v2"
				s.Tools.Tools = append(s.Tools.Tools, model.Tool{Name: "ping", InputSchema: json.RawMessage(`{"type":"object"}`)})
				s.Server.Instructions = "updated"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			from := baseSnap()
			to := baseSnap()
			if tc.mutFrom != nil {
				tc.mutFrom(&from)
			}
			if tc.mutTo != nil {
				tc.mutTo(&to)
			}
			got, err := tc.engine.Diff(context.Background(), from, to)
			if err != nil {
				t.Fatal(err)
			}
			if got.Verdict != tc.verdict {
				t.Fatalf("verdict %s, want %s", got.Verdict, tc.verdict)
			}
			wantChanged := tc.verdict == model.VerdictChanged
			if got.Changed != wantChanged {
				t.Fatalf("changed=%v, want %v", got.Changed, wantChanged)
			}
			if tc.verdict == model.VerdictEqual && len(got.Changes) != 0 {
				t.Fatalf("equal must have no changes: %+v", got.Changes)
			}
			if tc.path != "" {
				assertChange(t, got, tc.path, tc.cat, tc.sev)
			}
			if tc.name == "ordering" {
				assertSorted(t, got.Changes)
			}
			if tc.name == "rename-unmapped" {
				assertChange(t, got, "/tools/echo2", model.CategoryAddition, model.SeverityAdditive)
			}
			body, err := jsonv2.Marshal(got, jsonv2.Deterministic(true))
			if err != nil {
				t.Fatal(err)
			}
			body = append(body, '\n')
			path := filepath.Join("..", "..", "testdata", "diffs", tc.file)
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
					t.Fatal(mkErr)
				}
				if writeErr := os.WriteFile(path, body, 0o600); writeErr != nil {
					t.Fatal(writeErr)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if !bytes.Equal(body, want) {
				t.Fatalf("%s mismatch\n got %s\nwant %s", tc.file, body, want)
			}
		})
	}
}

func TestIncompleteNeverEquals(t *testing.T) {
	from := baseSnap()
	to := baseSnap()
	from.Incomplete = true
	from.IncompleteReasons = []string{model.ReasonListFailed}
	from.Tools.Status = model.SectionFailed
	got, err := Engine{}.Diff(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != model.VerdictIndeterminate || got.Changed {
		t.Fatalf("incomplete equal items must be indeterminate, not changed/equal: %+v", got)
	}
}

func TestResourcePointerEscapesSlash(t *testing.T) {
	from := baseSnap()
	to := baseSnap()
	to.Resources.Resources = nil
	got, err := Engine{}.Diff(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	assertChange(t, got, "/resources/fixture:~1~1note", model.CategoryRemoval, model.SeverityBreaking)
}

func assertChange(t *testing.T, got model.Diff, path string, cat model.DiffCategory, sev model.DiffSeverity) {
	t.Helper()
	for _, ch := range got.Changes {
		if ch.Path == path {
			if ch.Category != cat || ch.Severity != sev {
				t.Fatalf("%s: got %s/%s want %s/%s", path, ch.Category, ch.Severity, cat, sev)
			}
			if ch.Rationale == "" {
				t.Fatalf("%s: empty rationale", path)
			}
			return
		}
	}
	t.Fatalf("missing change %s in %+v", path, got.Changes)
}

func assertSorted(t *testing.T, changes []model.Change) {
	t.Helper()
	for i := 1; i < len(changes); i++ {
		if cmpChange(changes[i-1], changes[i]) > 0 {
			t.Fatalf("changes not sorted at %d: %s then %s", i, changes[i-1].Path, changes[i].Path)
		}
	}
}

func baseSnap() model.Snapshot {
	return model.Snapshot{
		SchemaVersion: model.SnapshotSchemaVersion,
		Protocol: model.SnapshotProtocol{
			Era: model.Era20251125, Version: string(model.Era20251125),
			Handshake: model.HandshakeInitialize, Discover: model.DiscoverNotApplicable,
		},
		Server:       model.SnapshotServer{Name: "lab", Title: "Lab", Version: "1.0.0"},
		Capabilities: model.ServerCapabilities{Tools: true, Prompts: true, Resources: true},
		Tools: model.CatalogSection{
			Kind: model.KindTools, Status: model.SectionComplete, ItemCount: 1,
			Tools: []model.Tool{{
				Name: "echo", Title: "Echo",
				InputSchema:  json.RawMessage(`{"properties":{"text":{"type":"string"}},"type":"object"}`),
				OutputSchema: json.RawMessage(`{"properties":{"ok":{"type":"boolean"}},"type":"object"}`),
			}},
		},
		Prompts: model.CatalogSection{
			Kind: model.KindPrompts, Status: model.SectionComplete, ItemCount: 1,
			Prompts: []model.Prompt{{
				Name: "greet", Arguments: []model.PromptArgument{{Name: "name", Required: true}},
			}},
		},
		Resources: model.CatalogSection{
			Kind: model.KindResources, Status: model.SectionComplete, ItemCount: 1,
			Resources: []model.Resource{{URI: "fixture://note", Name: "note"}},
		},
		Templates: model.CatalogSection{
			Kind: model.KindTemplates, Status: model.SectionComplete, ItemCount: 1,
			Templates: []model.ResourceTemplate{{URITemplate: "fixture://items/{id}", Name: "item"}},
		},
	}
}
