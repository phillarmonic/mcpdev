package watch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
)

func TestEditorBurstOneBuild(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := NewFake()
		builds := make(chan []Event, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		loop := Loop{
			Source:   fake,
			Clock:    clock.Wall{},
			Debounce: 50 * time.Millisecond,
			Matcher:  NewMatcher(nil, nil),
			Build: func(_ context.Context, batch []Event) error {
				builds <- batch
				return nil
			},
		}
		done := make(chan error, 1)
		go func() { done <- loop.Run(ctx) }()
		synctest.Wait()

		for _, op := range []Op{OpCreate, OpWrite, OpChmod} {
			if err := fake.Send(ctx, Event{Path: "internal/watch/loop.go", Op: op}); err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(50 * time.Millisecond)
		synctest.Wait()
		batch := recvBatch(t, ctx, builds)
		if len(batch) != 3 {
			t.Fatalf("batch len %d", len(batch))
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("run: %v", err)
		}
		select {
		case extra := <-builds:
			t.Fatalf("extra build %#v", extra)
		default:
		}
	})
}

func TestChangesDuringBuildOneFollowUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := NewFake()
		release := make(chan struct{})
		var mu sync.Mutex
		var n int
		started := make(chan int, 2)
		finished := make(chan int, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		loop := Loop{
			Source:   fake,
			Clock:    clock.Wall{},
			Debounce: 50 * time.Millisecond,
			Matcher:  NewMatcher(nil, nil),
			Build: func(bctx context.Context, _ []Event) error {
				mu.Lock()
				n++
				id := n
				mu.Unlock()
				started <- id
				if id == 1 {
					select {
					case <-release:
					case <-bctx.Done():
						return bctx.Err()
					}
				}
				finished <- id
				return nil
			},
		}
		done := make(chan error, 1)
		go func() { done <- loop.Run(ctx) }()
		synctest.Wait()

		if err := fake.Send(ctx, Event{Path: "a.go", Op: OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
		synctest.Wait()
		if got := recvInt(t, ctx, started); got != 1 {
			t.Fatalf("started %d", got)
		}
		if err := fake.Send(ctx, Event{Path: "b.go", Op: OpWrite}); err != nil {
			t.Fatal(err)
		}
		if err := fake.Send(ctx, Event{Path: "c.go", Op: OpWrite}); err != nil {
			t.Fatal(err)
		}
		close(release)
		synctest.Wait()
		if got := recvInt(t, ctx, finished); got != 1 {
			t.Fatalf("finished %d", got)
		}
		if got := recvInt(t, ctx, started); got != 2 {
			t.Fatalf("follow-up started %d", got)
		}
		if got := recvInt(t, ctx, finished); got != 2 {
			t.Fatalf("follow-up finished %d", got)
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("run: %v", err)
		}
	})
}

func TestBuildFailureKeepsWatching(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := NewFake()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reports := make(chan Report, 2)
		loop := Loop{
			Source:   fake,
			Clock:    clock.Wall{},
			Debounce: 10 * time.Millisecond,
			Matcher:  NewMatcher(nil, nil),
			Build: func(_ context.Context, batch []Event) error {
				if batch[0].Path == "bad.go" {
					return errors.New("compile failed")
				}
				return nil
			},
			OnReport: func(r Report) { reports <- r },
		}
		done := make(chan error, 1)
		go func() { done <- loop.Run(ctx) }()
		synctest.Wait()

		if err := fake.Send(ctx, Event{Path: "bad.go", Op: OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		first := recvReport(t, ctx, reports)
		if first.Err == nil {
			t.Fatal("expected build error")
		}
		if err := fake.Send(ctx, Event{Path: "good.go", Op: OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		second := recvReport(t, ctx, reports)
		if second.Err != nil {
			t.Fatalf("second build %v", second.Err)
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("run %v", err)
		}
	})
}

func TestIgnoredPathDoesNotBuild(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := NewFake()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		builds := make(chan []Event, 1)
		loop := Loop{
			Source:   fake,
			Clock:    clock.Wall{},
			Debounce: 10 * time.Millisecond,
			Matcher:  NewMatcher(nil, nil),
			Build: func(_ context.Context, batch []Event) error {
				builds <- batch
				return nil
			},
		}
		done := make(chan error, 1)
		go func() { done <- loop.Run(ctx) }()
		synctest.Wait()
		if err := fake.Send(ctx, Event{Path: "bin/mcpdev", Op: OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		select {
		case batch := <-builds:
			t.Fatalf("built %#v", batch)
		default:
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("run: %v", err)
		}
	})
}

func recvBatch(t *testing.T, ctx context.Context, ch <-chan []Event) []Event {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return nil
	}
}

func recvInt(t *testing.T, ctx context.Context, ch <-chan int) int {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return 0
	}
}

func recvReport(t *testing.T, ctx context.Context, ch <-chan Report) Report {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return Report{}
	}
}
