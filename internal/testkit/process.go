package testkit

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/platform"
)

// ProcessLab is a subprocess stdio fixture. Protocol bytes stay on stdout;
// diagnostics stay on stderr.
type ProcessLab struct {
	Stdin     io.WriteCloser
	Stdout    io.ReadCloser
	Cmd       *exec.Cmd
	Stderr    *limitedBuffer
	outScan   *bufio.Scanner
	Desc      Descriptor
	closeOnce sync.Once
}

// StartProcess builds testdata/servers/stdio-lab and runs it with MCPDEV_FIXTURE.
func StartProcess(t testing.TB, fixture string) *ProcessLab {
	t.Helper()
	d, err := Named(fixture)
	if err != nil {
		t.Fatal(err)
	}
	bin := buildStdioBinary(t)
	ctx, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(ctx, bin) //nolint:gosec // G204: binary is go-build output in t.TempDir
	cmd.Env = append(os.Environ(), "MCPDEV_FIXTURE="+fixture)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := newLimitedBuffer(64 * 1024)
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lab := &ProcessLab{Desc: d, Cmd: cmd, Stdin: stdin, Stdout: stdout, Stderr: stderr, outScan: sc}
	t.Cleanup(func() {
		_ = lab.Close()
		cancel()
	})
	return lab
}

// CallProcess writes one request and reads one response from the child.
func CallProcess(lab *ProcessLab, req Request) (Response, error) {
	if err := writeJSONLine(lab.Stdin, req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := readJSONLine(lab.outScan, &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

// Close closes stdin and waits for the child. The test Cleanup also cancels the command context.
func (l *ProcessLab) Close() error {
	var err error
	l.closeOnce.Do(func() {
		_ = l.Stdin.Close()
		err = l.Cmd.Wait()
	})
	return err
}

func buildStdioBinary(t testing.TB) string {
	t.Helper()
	out := platform.BuildPath(t.TempDir(), "stdio-lab")
	src := filepath.Join(moduleRoot(), "testdata", "servers", "stdio-lab")
	cmd := exec.Command("go", "build", "-o", out, ".") //nolint:gosec // G204: fixed go build of testdata/servers/stdio-lab
	cmd.Dir = src
	outb, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build stdio-lab: %v\n%s", err, outb)
	}
	return out
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testkit: runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

type limitedBuffer struct {
	buf  []byte
	max  int
	drop int
	mu   sync.Mutex
}

func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{max: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf)+len(p) > b.max {
		b.drop += len(p)
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func (b *limitedBuffer) Dropped() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.drop
}
