package snapshot

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestSemanticBytesIgnoreProvenance(t *testing.T) {
	inv := richInventory(model.Era20251125)
	a := mustSnap(t, inv, frozenMeta())
	other := frozenMeta()
	other.CapturedAt = other.CapturedAt.Add(time.Minute)
	other.GeneratorVersion = "9.9.9"
	other.SessionID = "other-session"
	other.URL = "https://other.invalid/mcp"
	other.Command = []string{"./bin/other"}
	b := mustSnap(t, inv, other)
	left, err := SemanticBytes(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := SemanticBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatalf("semantic bytes differ\n%s\n%s", left, right)
	}
	fullA, err := Encode(a)
	if err != nil {
		t.Fatal(err)
	}
	fullB, err := Encode(b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(fullA, fullB) {
		t.Fatal("full documents should retain volatile provenance")
	}
}

func TestIncompleteCannotMasquerade(t *testing.T) {
	inv := richInventory(model.Era20260728)
	inv.Tools.Status = model.SectionIncomplete
	inv.Tools.Reasons = []string{model.ReasonPaginationStopped}
	inv.Tools.StoppedCursor = "page-9"
	inv.Incomplete = true
	inv.Reasons = []string{model.ReasonPaginationStopped}
	snap := mustSnap(t, inv, frozenMeta())
	if !snap.Incomplete {
		t.Fatal("incomplete inventory must stay incomplete")
	}
	body, err := Encode(snap)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"incomplete":true`)) {
		t.Fatalf("encoded complete by mistake: %s", body)
	}
	snap.Incomplete = false
	if _, err := Encode(snap); err == nil {
		t.Fatal("expected masquerade to be rejected")
	}
}

func TestDecodeUnknownVersion(t *testing.T) {
	_, err := Decode([]byte(`{"schema_version":99,"incomplete":false,"incomplete_reasons":[]}`))
	if err == nil {
		t.Fatal("expected unknown version error")
	}
	if _, ok := errors.AsType[errclass.Usage](err); !ok {
		t.Fatalf("want Usage, got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "99") {
		t.Fatalf("error should name the version: %v", err)
	}
}

func TestDecodeCorrupt(t *testing.T) {
	_, err := Decode([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected corrupt error")
	}
	if _, ok := errors.AsType[errclass.Usage](err); !ok {
		t.Fatalf("want Usage, got %T %v", err, err)
	}
}

func TestCanonicalSchemaAndURI(t *testing.T) {
	inv := richInventory(model.Era20251125)
	inv.Tools.Tools[1].InputSchema = jsonRaw(`{"type":"object","required":["n"],"properties":{"z":{"type":"integer"},"n":{"type":"number","minimum":1.5}},"$defs":{"inner":{"type":"string"}}}`)
	snap := mustSnap(t, inv, frozenMeta())
	schema := string(snap.Tools.Tools[1].InputSchema)
	if !strings.HasPrefix(schema, `{"$defs"`) {
		t.Fatalf("schema keys should be sorted: %s", schema)
	}
	if !strings.Contains(schema, `1.5`) {
		t.Fatalf("number token lost: %s", schema)
	}
	if snap.Resources.Resources[1].URI != "file:///tmp/weird path" {
		t.Fatalf("URI rewritten: %q", snap.Resources.Resources[1].URI)
	}
	if string(snap.Tools.Tools[0].Meta) != `{"owner":"lab","token":"[REDACTED]"}` {
		t.Fatalf("meta redaction: %s", snap.Tools.Tools[0].Meta)
	}
}

func TestRoundTrip(t *testing.T) {
	snap := mustSnap(t, richInventory(model.Era20260728), frozenMeta())
	body, err := Encode(snap)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(body)
	if err != nil {
		t.Fatal(err)
	}
	left, err := SemanticBytes(snap)
	if err != nil {
		t.Fatal(err)
	}
	right, err := SemanticBytes(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatalf("round trip semantic mismatch\n%s\n%s", left, right)
	}
}

func TestGoldens(t *testing.T) {
	cases := []struct {
		mut  func(*model.Inventory)
		name string
		era  model.ProtocolEra
	}{
		{name: "legacy-full.json", era: model.Era20251125},
		{name: "modern-full.json", era: model.Era20260728},
		{name: "incomplete.json", era: model.Era20251125, mut: func(inv *model.Inventory) {
			inv.Tools.Status = model.SectionIncomplete
			inv.Tools.Reasons = []string{model.ReasonPaginationStopped}
			inv.Tools.StoppedCursor = "page-9"
			inv.Incomplete = true
			inv.Reasons = []string{model.ReasonPaginationStopped}
		}},
	}
	for _, tc := range cases {
		inv := richInventory(tc.era)
		if tc.mut != nil {
			tc.mut(&inv)
		}
		snap := mustSnap(t, inv, frozenMeta())
		got, err := Encode(snap)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("..", "..", "testdata", "snapshots", tc.name)
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
				t.Fatal(mkErr)
			}
			if writeErr := os.WriteFile(path, got, 0o600); writeErr != nil {
				t.Fatal(writeErr)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s mismatch\n got %s\nwant %s", tc.name, got, want)
		}
	}
}

func jsonRaw(s string) []byte { return []byte(s) }
