package watch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	fswatch "github.com/phillarmonic/mcpdev/internal/watch"
)

func TestSessionJSONLShapeAndShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := fswatch.NewFake()
		var live atomic.Int32
		var notes []fswatch.Note
		var gen atomic.Int64
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		sess := &Session{
			Source:       fake,
			Clock:        clock.Wall{},
			Matcher:      fswatch.NewMatcher(nil, nil),
			Debounce:     10 * time.Millisecond,
			OnSuccess:    OnSuccessReconnect,
			KeepLastGood: true,
			Emit:         func(n fswatch.Note) { notes = append(notes, n) },
			Ports: Ports{
				Rebuild: func(context.Context, []fswatch.Event) RebuildReport {
					return RebuildReport{Class: "ok"}
				},
				Begin: func() (int64, error) { return gen.Add(1), nil },
				Promote: func(context.Context, int64, string) error {
					live.Add(1)
					return nil
				},
				FailBuild: func(int64, string, string) error { return nil },
				Shutdown: func(context.Context) error {
					live.Store(0)
					return nil
				},
			},
		}
		done := make(chan error, 1)
		go func() { done <- sess.Run(ctx) }()
		synctest.Wait()
		if live.Load() != 1 {
			t.Fatalf("boot live %d", live.Load())
		}
		if err := fake.Send(ctx, fswatch.Event{Path: "a.go", Op: fswatch.OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		cancel()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if live.Load() != 0 {
			t.Fatalf("child still live %d", live.Load())
		}
		kinds := map[fswatch.Kind]int{}
		for _, n := range notes {
			kinds[n.Kind]++
			if n.ID == "" || n.At.IsZero() {
				t.Fatalf("note missing id/ts %#v", n)
			}
		}
		for _, k := range []fswatch.Kind{fswatch.KindFS, fswatch.KindBuild, fswatch.KindRestart, fswatch.KindNegotiate, fswatch.KindProbe, fswatch.KindShutdown} {
			if kinds[k] == 0 {
				t.Fatalf("missing kind %s in %#v", k, kinds)
			}
		}
	})
}

func TestFailFastBuild(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := fswatch.NewFake()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		sess := &Session{
			Source:    fake,
			Clock:     clock.Wall{},
			Matcher:   fswatch.NewMatcher(nil, nil),
			Debounce:  10 * time.Millisecond,
			FailFast:  true,
			OnSuccess: OnSuccessNone,
			Ports: Ports{
				Rebuild: func(context.Context, []fswatch.Event) RebuildReport {
					return RebuildReport{Class: "nonzero", Err: errors.New("compile")}
				},
				Shutdown: func(context.Context) error { return nil },
			},
		}
		done := make(chan error, 1)
		go func() { done <- sess.Run(ctx) }()
		synctest.Wait()
		if err := fake.Send(ctx, fswatch.Event{Path: "bad.go", Op: fswatch.OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		err := <-done
		if err == nil {
			t.Fatal("expected fail-fast")
		}
	})
}

func TestExitOnChangeFromPromote(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := fswatch.NewFake()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var n int
		sess := &Session{
			Source:       fake,
			Clock:        clock.Wall{},
			Matcher:      fswatch.NewMatcher(nil, nil),
			Debounce:     10 * time.Millisecond,
			ExitOnChange: true,
			OnSuccess:    OnSuccessReconnect,
			Ports: Ports{
				Rebuild: func(context.Context, []fswatch.Event) RebuildReport {
					return RebuildReport{Class: "ok"}
				},
				Begin: func() (int64, error) { n++; return int64(n), nil },
				Promote: func(context.Context, int64, string) error {
					if n == 1 {
						return nil
					}
					return errclass.InterfaceChange{Err: errors.New("breaking")}
				},
				Shutdown: func(context.Context) error { return nil },
			},
		}
		done := make(chan error, 1)
		go func() { done <- sess.Run(ctx) }()
		synctest.Wait()
		if err := fake.Send(ctx, fswatch.Event{Path: "a.go", Op: fswatch.OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		err := <-done
		if errclass.ExitCode(err) != errclass.ExitInterfaceChange {
			t.Fatalf("code %d err %v", errclass.ExitCode(err), err)
		}
	})
}
