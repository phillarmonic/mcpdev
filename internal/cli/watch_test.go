package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	appwatch "github.com/phillarmonic/mcpdev/internal/app/watch"
	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	fswatch "github.com/phillarmonic/mcpdev/internal/watch"
)

func TestWatchJSONLAndCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := fswatch.NewFake()
		testWatchSource = fake
		testWatchClock = clock.Wall{}
		var live atomic.Int32
		var gen atomic.Int64
		testWatchPorts = &appwatch.Ports{
			Rebuild: func(context.Context, []fswatch.Event) appwatch.RebuildReport {
				return appwatch.RebuildReport{Class: "ok"}
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
		}
		defer func() {
			testWatchSource = nil
			testWatchPorts = nil
			testWatchClock = nil
		}()

		var stdout, stderr bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan int, 1)
		go func() {
			done <- runWatchCtx(ctx, Invocation{
				Command:     "watch",
				CommandArgv: []string{"./bin/server"},
				Format:      "json",
				FormatSet:   true,
				Debounce:    10 * time.Millisecond,
				DebounceSet: true,
				RebuildArgv: []string{"true"},
			}, &stdout, &stderr)
		}()
		synctest.Wait()
		if err := fake.Send(ctx, fswatch.Event{Path: "main.go", Op: fswatch.OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		cancel()
		code := <-done
		if code != errclass.ExitOK {
			t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
		}
		if live.Load() != 0 {
			t.Fatalf("managed child still live %d", live.Load())
		}
		if stderr.Len() != 0 {
			t.Fatalf("json mode wrote stderr %s", stderr.String())
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) < 4 {
			t.Fatalf("few events %q", stdout.String())
		}
		seen := map[string]bool{}
		for _, line := range lines {
			var env map[string]any
			if err := json.Unmarshal([]byte(line), &env); err != nil {
				t.Fatalf("line %s: %v", line, err)
			}
			kind, _ := env["kind"].(string)
			seen[kind] = true
			if env["id"] == nil || env["ts"] == nil {
				t.Fatalf("missing id/ts %v", env)
			}
		}
		for _, k := range []string{"fs", "build", "restart", "negotiate", "probe", "shutdown"} {
			if !seen[k] {
				t.Fatalf("missing %s in %v\n%s", k, seen, stdout.String())
			}
		}
	})
}

func TestWatchHumanMatchesJSONKinds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := fswatch.NewFake()
		testWatchSource = fake
		testWatchClock = clock.Wall{}
		testWatchPorts = &appwatch.Ports{
			Rebuild: func(context.Context, []fswatch.Event) appwatch.RebuildReport {
				return appwatch.RebuildReport{Class: "ok"}
			},
			Begin:     func() (int64, error) { return 1, nil },
			Promote:   func(context.Context, int64, string) error { return nil },
			FailBuild: func(int64, string, string) error { return nil },
			Shutdown:  func(context.Context) error { return nil },
		}
		defer func() {
			testWatchSource = nil
			testWatchPorts = nil
			testWatchClock = nil
		}()
		var stdout, stderr bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan int, 1)
		go func() {
			done <- runWatchCtx(ctx, Invocation{
				Command:     "watch",
				CommandArgv: []string{"./bin/server"},
				Format:      "human",
				FormatSet:   true,
				DebounceSet: true,
				Debounce:    10 * time.Millisecond,
				OnSuccess:   "none",
			}, &stdout, &stderr)
		}()
		synctest.Wait()
		if err := fake.Send(ctx, fswatch.Event{Path: "a.go", Op: fswatch.OpWrite}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		synctest.Wait()
		cancel()
		if code := <-done; code != 0 {
			t.Fatalf("code %d stderr=%s", code, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("human stdout %s", stdout.String())
		}
		if strings.Contains(stderr.String(), "\x1b") {
			t.Fatal("ANSI on non-TTY stderr")
		}
		for _, token := range []string{"fs", "build", "shutdown"} {
			if !strings.Contains(stderr.String(), token) {
				t.Fatalf("missing %s in %s", token, stderr.String())
			}
		}
	})
}

func TestWatchRequiresCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"mcpdev", "watch"}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d", code)
	}
}
