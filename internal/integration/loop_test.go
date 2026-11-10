package integration

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/cli"
	"github.com/phillarmonic/mcpdev/internal/config"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func labBin(t *testing.T) string {
	t.Helper()
	lab := testkit.StartProcess(t, "legacy-stdio")
	_ = lab.Close()
	return lab.Cmd.Path
}

func isolateConfig(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MCPDEV_CONFIG", filepath.Join(integrationDir(t), "mcpdev.toml"))
}

func TestProfileTargets(t *testing.T) {
	isolateConfig(t)
	env := map[string]string{
		"MCPDEV_CONFIG": os.Getenv("MCPDEV_CONFIG"),
	}
	s, err := config.Load(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	lab, err := s.ResolveProfile("lab", env)
	if err != nil {
		t.Fatal(err)
	}
	if len(lab.Command) != 1 || lab.Command[0] != "./bin/stdio-lab" {
		t.Fatalf("command %+v", lab.Command)
	}
	if lab.Dir != "" {
		t.Fatalf("cwd %q want inherited working directory", lab.Dir)
	}
	if lab.Protocol != "2025-11-25" {
		t.Fatalf("protocol %q", lab.Protocol)
	}

	const labURL = "http://127.0.0.1:8123/mcp"
	env["MCPDEV_LAB_URL"] = labURL
	env["MCPDEV_LAB_TOKEN"] = "Bearer fixture-token"
	s, err = config.Load(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	httpTarget, err := s.ResolveProfile("http", env)
	if err != nil {
		t.Fatal(err)
	}
	if httpTarget.URL != labURL {
		t.Fatalf("url %q want %q", httpTarget.URL, labURL)
	}
	if httpTarget.Protocol != "2026-07-28" {
		t.Fatalf("protocol %q", httpTarget.Protocol)
	}
}

func TestLabInventoryAndSafeOps(t *testing.T) {
	isolateConfig(t)
	bin := labBin(t)
	kinds := []string{"tools", "prompts", "resources", "templates"}
	needles := []string{`"echo"`, `"greet"`, `fixture://stable/note`, `fixture://items/{id}`}
	for i, kind := range kinds {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{
			"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
			"--protocol", "2025-11-25", "inspect", kind,
		}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("inspect %s code %d stderr=%s stdout=%s", kind, code, stderr.String(), stdout.String())
		}
		if !strings.Contains(stdout.String(), needles[i]) {
			t.Fatalf("inspect %s missing %s: %s", kind, needles[i], stdout.String())
		}
	}
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "call", "echo", "--args", `{"text":"loop"}`,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("call %d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "prompt", "greet", "--arg", "name=lab",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("prompt %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
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
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "read", "--template", "fixture://items/{id}", "--arg", "id=n1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("template %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestSnapshotThenFixtureInterfaceChange(t *testing.T) {
	isolateConfig(t)
	bin := labBin(t)
	base := filepath.Join(t.TempDir(), "baseline.json")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "snapshot", "--out", base,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("snapshot %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-paginate",
		"--protocol", "2025-11-25", "diff", base,
	}, &stdout, &stderr)
	if code != errclass.ExitInterfaceChange {
		t.Fatalf("diff code %d want %d stderr=%s stdout=%s", code, errclass.ExitInterfaceChange, stderr.String(), stdout.String())
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env["ok"] == true {
		t.Fatalf("changed interface reported ok: %v", env)
	}
}

func TestHTTPModernAuthHeadersRedacted(t *testing.T) {
	isolateConfig(t)
	d, err := testkit.Named("modern-auth")
	if err != nil {
		t.Fatal(err)
	}
	lab := testkit.StartHTTP(t, d)
	const token = "fixture-token"
	t.Setenv("MCPDEV_LAB_URL", lab.Server.URL)
	t.Setenv("MCPDEV_LAB_TOKEN", "Bearer "+token)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--trace", "--trace-format", "json",
		"--profile", "http", "inspect", "tools",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("http inspect %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, token) {
		t.Fatalf("token leaked: %s", combined)
	}
	if !strings.Contains(stdout.String(), `"echo"`) {
		t.Fatalf("missing echo: %s", stdout.String())
	}
}
