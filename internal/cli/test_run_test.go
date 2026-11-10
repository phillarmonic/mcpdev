package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/cli"
	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func TestTestInvalidFileUsage(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "scenarios", "invalid", "shell-step.yaml")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--format", "json", "test", path}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok":false`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestTestStdioJSON(t *testing.T) {
	bin := buildLab(t)
	path := filepath.Join("..", "..", "testdata", "scenarios", "valid", "stdio.json")
	junit := filepath.Join(t.TempDir(), "report.xml")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "--jobs", "1", "--junit", junit, "test", path,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	body, err := os.ReadFile(junit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("testsuite")) {
		t.Fatalf("junit %s", body)
	}
}

func TestTestCallArgumentsNamedCommand(t *testing.T) {
	bin := buildLab(t)
	path := filepath.Join("..", "..", "testdata", "scenarios", "valid", "call-command-arg.yaml")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "--jobs", "1", "test", path,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}
