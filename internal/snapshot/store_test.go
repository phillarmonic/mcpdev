package snapshot

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/model"
)

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func (f fixedClock) After(time.Duration) <-chan time.Time {
	ch := make(chan time.Time)
	close(ch)
	return ch
}

func TestStoreWriteRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	store := Store{Clock: fixedClock{t: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)}, Version: "0.1.0"}
	snap := mustSnap(t, richInventory(model.Era20251125), frozenMeta())
	if err := store.Write(context.Background(), path, snap); err != nil {
		t.Fatal(err)
	}
	got, err := store.Read(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := SemanticBytes(snap)
	right, _ := SemanticBytes(got)
	if !bytes.Equal(left, right) {
		t.Fatalf("store round trip\n%s\n%s", left, right)
	}
}

func TestStoreWriteReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	store := Store{Clock: fixedClock{t: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)}, Version: "0.1.0"}
	first := mustSnap(t, richInventory(model.Era20251125), frozenMeta())
	if err := store.Write(context.Background(), path, first); err != nil {
		t.Fatal(err)
	}
	secondInv := richInventory(model.Era20260728)
	second := mustSnap(t, secondInv, frozenMeta())
	if err := store.Write(context.Background(), path, second); err != nil {
		t.Fatal(err)
	}
	got, err := store.Read(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Protocol.Era != model.Era20260728 {
		t.Fatalf("era %s", got.Protocol.Era)
	}
}

func TestStoreRejectsDirectory(t *testing.T) {
	store := Store{}
	if err := store.Write(context.Background(), t.TempDir(), mustSnap(t, richInventory(model.Era20251125), frozenMeta())); err == nil {
		t.Fatal("expected directory path to fail")
	}
}
