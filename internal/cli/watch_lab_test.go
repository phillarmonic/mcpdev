package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

type lockedBuf struct {
	b  strings.Builder
	mu sync.Mutex
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestWatchLabRebuildRestartReconnect(t *testing.T) {
	lab := testkit.StartProcess(t, "legacy-stdio")
	_ = lab.Close()
	bin := lab.Cmd.Path
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MCPDEV_CONFIG", "")
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "keep.go"), []byte("package keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr lockedBuf
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan int, 1)
	go func() {
		done <- runWatchCtx(ctx, Invocation{
			Command:     "watch",
			CommandArgv: []string{bin},
			Env:         []string{"MCPDEV_FIXTURE=legacy-stdio"},
			Protocol:    "2025-11-25",
			ProtocolSet: true,
			Format:      model.FormatJSON,
			FormatSet:   true,
			RebuildArgv: []string{"true"},
			WatchRoots:  []string{src},
			Include:     []string{"*.go"},
			Debounce:    50 * time.Millisecond,
			DebounceSet: true,
			OnSuccess:   "reconnect",
		}, &stdout, &stderr)
	}()

	waitKind(t, &stdout, &stderr, "probe", 20*time.Second)
	if err := os.WriteFile(filepath.Join(src, "nudge.go"), []byte("package keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitKindCount(t, &stdout, &stderr, "restart", 2, 20*time.Second)
	cancel()
	select {
	case code := <-done:
		out := stdout.String()
		if !strings.Contains(out, `"generation":2`) || !strings.Contains(out, `"kind":"probe"`) {
			t.Fatalf("watch exit %d missing generation 2 probe stderr=%s stdout=%s", code, stderr.String(), out)
		}
		for _, kind := range []string{"fs", "build", "restart", "probe", "shutdown"} {
			if !strings.Contains(out, `"kind":"`+kind+`"`) {
				t.Fatalf("missing %s in %s", kind, out)
			}
		}
		if code != errclass.ExitOK && !strings.Contains(stderr.String(), "file already closed") {
			t.Fatalf("watch exit %d stderr=%s stdout=%s", code, stderr.String(), out)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("watch did not exit stderr=%s stdout=%s", stderr.String(), stdout.String())
	}
}

func waitKind(t *testing.T, stdout, stderr *lockedBuf, kind string, d time.Duration) {
	t.Helper()
	waitKindCount(t, stdout, stderr, kind, 1, d)
}

func waitKindCount(t *testing.T, stdout, stderr *lockedBuf, kind string, n int, d time.Duration) {
	t.Helper()
	needle := `"kind":"` + kind + `"`
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if strings.Count(stdout.String(), needle) >= n {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d %s stderr=%s stdout=%s", n, kind, stderr.String(), stdout.String())
}
