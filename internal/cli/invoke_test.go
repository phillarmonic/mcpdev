package cli_test

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/cli"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestCallStdioAndHTTP(t *testing.T) {
	bin := buildLab(t)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "call", "echo", "--args", `{"text":"hi"}`,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("stdio call %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"hi"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
	d, err := testkit.Named("modern-http")
	if err != nil {
		t.Fatal(err)
	}
	lab := testkit.StartHTTP(t, d)
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--url", lab.Server.URL, "--protocol", "2026-07-28",
		"call", "echo", "--args", `{"text":"http"}`,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("http call %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"http"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestCallIsErrorExit(t *testing.T) {
	bin := buildLab(t)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "call", "fail",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("isError should be 0, got %d stderr=%s", code, stderr.String())
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	res := env["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("%v", env)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "--fail-on-tool-error", "call", "fail",
	}, &stdout, &stderr)
	if code != errclass.ExitAssertion {
		t.Fatalf("code %d stderr=%s", code, stderr.String())
	}
}

func TestPromptAndReadStdio(t *testing.T) {
	bin := buildLab(t)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "prompt", "greet", "--arg", "name=true",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("prompt %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "greet true") {
		t.Fatalf("stdout %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "read", "fixture://stable/note",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("read %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "stable-note") {
		t.Fatalf("%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "read", "--template", "fixture://items/{id}", "--arg", "id=a b",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("template %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "item:a%20b") {
		t.Fatalf("%s", stdout.String())
	}
}

func TestCallArgsFileAndSaveBlob(t *testing.T) {
	bin := buildLab(t)
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "echo.json")
	if err := os.WriteFile(argsPath, []byte(`{"text":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "call", "echo", "--args-file", argsPath,
	}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "from-file") {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	save := filepath.Join(dir, "blob.bin")
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "read", "fixture://stable/blob", "--save", save, "--max-bytes", "1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("blob %d stderr=%s", code, stderr.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte{0x00}) {
		t.Fatal("NUL on stdout")
	}
	got, err := os.ReadFile(save)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{0x00, 0xff, 0x80}) {
		t.Fatalf("saved %x", got)
	}
}

func TestCallUnknownToolIsProtocol(t *testing.T) {
	bin := buildLab(t)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "call", "missing",
	}, &stdout, &stderr)
	if code != errclass.ExitProtocol {
		t.Fatalf("code %d stderr=%s", code, stderr.String())
	}
}
