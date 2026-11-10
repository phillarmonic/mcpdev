package stdio

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/platform"
	"github.com/phillarmonic/mcpdev/internal/port"
)

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
		panic("stdio: runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestOpenKeepsProtocolOnStdout(t *testing.T) {
	bin := helperBin(t)
	conn, err := Open(t.Context(), port.ProcSpec{
		Argv:        []string{bin},
		Env:         map[string]string{"MCPDEV_HELPER": "stderr-flood"},
		StderrLimit: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	line, err := readLine(conn.Stream().Reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(line, []byte(`"ok":true`)) {
		t.Fatalf("stdout %s", line)
	}
	if conn.StderrText() == "" {
		t.Fatal("expected captured stderr")
	}
}

func TestInvalidStdoutIsProtocolNotStderr(t *testing.T) {
	bin := helperBin(t)
	conn, err := Open(t.Context(), port.ProcSpec{
		Argv: []string{bin},
		Env:  map[string]string{"MCPDEV_HELPER": "bad-stdout"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	_, err = readLine(conn.Stream().Reader)
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("exit %d err %v", errclass.ExitCode(err), err)
	}
}

func TestEarlyExitOnStdoutIsConnection(t *testing.T) {
	bin := helperBin(t)
	conn, err := Open(t.Context(), port.ProcSpec{
		Argv: []string{bin},
		Env:  map[string]string{"MCPDEV_HELPER": "early-exit"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	_, err = readLine(conn.Stream().Reader)
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("exit %d err %v", errclass.ExitCode(err), err)
	}
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
