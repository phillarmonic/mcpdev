package cli_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/cli"
	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func TestReplayInspect(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "replays", "legacy-init.json")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--format", "json", "--replay", path, "inspect", "tools"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"echo"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestReplayPagination(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "replays", "pagination.json")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--format", "json", "--replay", path, "inspect", "tools"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"alpha"`) || !strings.Contains(stdout.String(), `"beta"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestRecordRequiresPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--record", "-", "--command", "./bin/x", "inspect"}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d stderr=%s", code, stderr.String())
	}
}

func TestReplayAndCommandRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--replay", "x.json", "--command", "./bin/x", "inspect"}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d", code)
	}
}
