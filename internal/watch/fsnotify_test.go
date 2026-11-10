package watch

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestMapOp(t *testing.T) {
	got := mapOp(fsnotify.Create | fsnotify.Write | fsnotify.Rename)
	want := OpCreate | OpWrite | OpRename
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
