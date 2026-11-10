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
)

func snapFile(name string) string {
	return filepath.Join("..", "..", "testdata", "snapshots", name)
}

func TestDiffFileEqual(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "diff",
		snapFile("legacy-full.json"), snapFile("legacy-full.json"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env["ok"] != true || env["outcome"] != "no-change" {
		t.Fatalf("%v", env)
	}
}

func TestDiffFileChanged(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "diff",
		snapFile("legacy-full.json"), snapFile("modern-full.json"),
	}, &stdout, &stderr)
	if code != errclass.ExitInterfaceChange {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"outcome":"threshold-failure"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestDiffIncomplete(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "diff",
		snapFile("incomplete.json"), snapFile("legacy-full.json"),
	}, &stdout, &stderr)
	if code != errclass.ExitIncomplete {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestDiffAllowIncomplete(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--allow-incomplete", "diff",
		snapFile("incomplete.json"), snapFile("legacy-full.json"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"outcome":"indeterminate"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestDiffFailOnBreakingIgnoresAdditiveOnlyWhenNone(t *testing.T) {
	// metadata-only would need a custom file; fail-on breaking still fails on era change.
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--fail-on", "breaking", "diff",
		snapFile("legacy-full.json"), snapFile("modern-full.json"),
	}, &stdout, &stderr)
	if code != errclass.ExitInterfaceChange {
		t.Fatalf("code %d stderr=%s", code, stderr.String())
	}
}

func TestSnapshotAndLiveDiff(t *testing.T) {
	bin := buildLab(t)
	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "snapshot", "--out", base,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("snapshot code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"schema_version":1`) {
		t.Fatalf("expected snapshot document on stdout: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "diff", base,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("live-to-file code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json",
		"--from-command", bin, "--to-command", bin,
		"--env", "MCPDEV_FIXTURE=legacy-stdio", "--protocol", "2025-11-25",
		"diff",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("live-to-live code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestSnapshotUsageNeedsTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "snapshot"}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d", code)
	}
}
