package process

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/platform"
	"github.com/phillarmonic/mcpdev/internal/port"
)

type immediateClock struct{}

func (immediateClock) Now() time.Time { return time.Unix(0, 0).UTC() }

func (immediateClock) After(time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Unix(0, 0).UTC()
	return ch
}

func helperBin(t *testing.T) string {
	t.Helper()
	out := platform.BuildPath(t.TempDir(), "process-helper")
	src := filepath.Join(moduleRoot(), "testdata", "process-helper")
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = src
	outb, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build process-helper: %v\n%s", err, outb)
	}
	return out
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("process: runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func startHelper(t *testing.T, spec port.ProcSpec) *Proc {
	t.Helper()
	p, err := Supervisor{}.Start(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	proc := p.(*Proc)
	t.Cleanup(func() {
		_ = proc.Close(context.Background())
	})
	return proc
}

func TestMergeEnvOverridesAndSorts(t *testing.T) {
	spec := port.ProcSpec{
		ClearEnv: true,
		Env: map[string]string{
			"B": "2",
			"A": "1",
		},
	}
	got := mergeEnv(spec)
	want := []string{"A=1", "B=2"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %#v", got)
	}
}

func TestStartRequiresArgv(t *testing.T) {
	_, err := Supervisor{}.Start(t.Context(), port.ProcSpec{})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("exit %d err %v", errclass.ExitCode(err), err)
	}
}

func TestPathWithSpacesAndEnvOverride(t *testing.T) {
	bin := helperBin(t)
	dir := filepath.Join(t.TempDir(), "dir with spaces")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Windows resolves executables by PATHEXT, so the copy needs the .exe suffix.
	spaced := platform.BuildPath(dir, "helper bin")
	data, readErr := os.ReadFile(bin)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if writeErr := os.WriteFile(spaced, data, 0o750); writeErr != nil {
		t.Fatal(writeErr)
	}
	p := startHelper(t, port.ProcSpec{
		Argv: []string{spaced},
		Env:  map[string]string{"MCPDEV_HELPER": "echo-env", "MCPDEV_HELPER_KEY": "CUSTOM_VAR", "CUSTOM_VAR": "from-spec"},
	})
	line, err := readLine(p.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(line, []byte(`"from-spec"`)) {
		t.Fatalf("stdout %s", line)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCwd(t *testing.T) {
	bin := helperBin(t)
	dir := t.TempDir()
	p := startHelper(t, port.ProcSpec{
		Argv: []string{bin},
		Dir:  dir,
		Env:  map[string]string{"MCPDEV_HELPER": "cwd"},
	})
	line, err := readLine(p.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Cwd string `json:"cwd"`
	}
	if unmarshalErr := json.Unmarshal(line, &got); unmarshalErr != nil {
		t.Fatalf("cwd line %s: %v", line, unmarshalErr)
	}
	// Windows reports the 8.3 short form of a temp path; compare the file identity.
	want, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	have, err := os.Stat(got.Cwd)
	if err != nil {
		t.Fatalf("cwd %q: %v", got.Cwd, err)
	}
	if !os.SameFile(want, have) {
		t.Fatalf("cwd %q is not %q", got.Cwd, dir)
	}
}

func TestStderrBoundDoesNotBlockStdout(t *testing.T) {
	bin := helperBin(t)
	p := startHelper(t, port.ProcSpec{
		Argv:        []string{bin},
		Env:         map[string]string{"MCPDEV_HELPER": "stderr-flood"},
		StderrLimit: 64,
	})
	line, err := readLine(p.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(line, []byte(`"ok":true`)) {
		t.Fatalf("stdout %s", line)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.StderrDropped() == 0 {
		t.Fatal("expected dropped stderr bytes")
	}
	if len(p.StderrText()) > 64 {
		t.Fatalf("stderr len %d", len(p.StderrText()))
	}
}

func TestEarlyExitIsConnection(t *testing.T) {
	bin := helperBin(t)
	p := startHelper(t, port.ProcSpec{
		Argv: []string{bin},
		Env:  map[string]string{"MCPDEV_HELPER": "early-exit"},
	})
	err := p.Wait()
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("exit %d err %v", errclass.ExitCode(err), err)
	}
	if p.StderrText() == "" {
		t.Fatal("expected child stderr on the connection error path")
	}
}

func TestCancelKillsChild(t *testing.T) {
	bin := helperBin(t)
	ctx, cancel := context.WithCancel(context.Background())
	p, err := Supervisor{}.Start(ctx, port.ProcSpec{
		Argv:        []string{bin},
		Env:         map[string]string{"MCPDEV_HELPER": "ignore-term"},
		GracePeriod: time.Nanosecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	proc := p.(*Proc)
	pid := proc.PID()
	_, err = readLine(proc.Stdout())
	if err != nil {
		cancel()
		_ = proc.Close(context.Background())
		t.Fatal(err)
	}
	cancel()
	waitErr := proc.Wait()
	if waitErr == nil {
		t.Fatal("expected wait error after cancel kill")
	}
	assertGone(t, pid)
}

func TestCloseKillsOrphan(t *testing.T) {
	bin := helperBin(t)
	p := startHelper(t, port.ProcSpec{
		Argv:        []string{bin},
		Env:         map[string]string{"MCPDEV_HELPER": "ignore-term"},
		Clock:       immediateClock{},
		GracePeriod: time.Second,
	})
	pid := p.PID()
	if _, err := readLine(p.Stdout()); err != nil {
		t.Fatal(err)
	}
	_ = p.Close(context.Background())
	assertGone(t, pid)
}

func readLine(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
			if b[0] == '\n' {
				return buf.Bytes(), nil
			}
		}
		if err != nil {
			return buf.Bytes(), err
		}
	}
}

func assertGone(t *testing.T, pid int) {
	t.Helper()
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	err = signalAlive(proc)
	if err == nil {
		t.Fatalf("pid %d still running", pid)
	}
}
