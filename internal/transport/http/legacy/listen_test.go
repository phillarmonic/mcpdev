package legacy

import (
	"context"
	"sync"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestListenReconnectsWithLastEventID(t *testing.T) {
	d, err := testkit.Named("legacy-http")
	if err != nil {
		t.Fatal(err)
	}
	d.NotifyListChanged = true
	d.DropGETOnce = true
	lab := testkit.StartHTTP(t, d)
	conn := dialLab(t, lab, Config{MaxReconnects: 3})
	handshake(t, conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var ids []string
	done := make(chan error, 1)
	go func() {
		done <- conn.Listen(ctx, func(ev Event) error {
			mu.Lock()
			ids = append(ids, ev.ID)
			n := len(ids)
			mu.Unlock()
			if n >= 2 {
				cancel()
			}
			return nil
		})
	}()
	err = <-done
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := append([]string(nil), ids...)
	mu.Unlock()
	if len(got) < 2 {
		t.Fatalf("events %v", got)
	}
	if lab.GETCount() < 2 {
		t.Fatalf("GET count %d", lab.GETCount())
	}
	if lab.SeenLastEventID() != "1" {
		t.Fatalf("Last-Event-ID %q", lab.SeenLastEventID())
	}
}
