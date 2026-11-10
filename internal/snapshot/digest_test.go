package snapshot

import (
	"encoding/json"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestSemanticDigestsIgnoreProvenance(t *testing.T) {
	inv := richInventory(model.Era20251125)
	a := mustSnap(t, inv, frozenMeta())
	meta := frozenMeta()
	meta.SessionID = "other"
	b := mustSnap(t, inv, meta)
	left, err := SemanticDigests(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := SemanticDigests(b)
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("%+v vs %+v", left, right)
	}
	if left.Snapshot == "" || left.Tools == "" || left.Prompts == "" {
		t.Fatal("digests must be populated")
	}
}

func TestSemanticDigestsChangeWithSchema(t *testing.T) {
	inv := richInventory(model.Era20251125)
	base := mustSnap(t, inv, frozenMeta())
	inv.Tools.Tools[0].InputSchema = json.RawMessage(`{"type":"object","additionalProperties":false}`)
	changed := mustSnap(t, inv, frozenMeta())
	left, err := SemanticDigests(base)
	if err != nil {
		t.Fatal(err)
	}
	right, err := SemanticDigests(changed)
	if err != nil {
		t.Fatal(err)
	}
	if left.Snapshot == right.Snapshot || left.Tools == right.Tools {
		t.Fatal("schema change must change snapshot and tools digests")
	}
	if left.Prompts != right.Prompts || left.Resources != right.Resources {
		t.Fatal("unrelated primitives must keep their digests")
	}
}
