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

func TestUsageCommandAndURL(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--command", "./bin/x", "--url", "http://127.0.0.1/mcp", "inspect"}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "inspect"}, &stdout, &stderr)
	if code != errclass.ExitUsage {
		t.Fatalf("code %d", code)
	}
}

func TestHelpStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--help"}, &stdout, &stderr)
	if code != 0 || stdout.Len() == 0 || stderr.Len() != 0 {
		t.Fatalf("help code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestInspectStdioJSON(t *testing.T) {
	bin := buildLab(t)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
		"--protocol", "2025-11-25", "inspect",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte{0x1b}) {
		t.Fatal("ANSI on stdout")
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env["ok"] != true || env["incomplete"] != false {
		t.Fatalf("%v", env)
	}
	if !strings.Contains(stdout.String(), `"echo"`) {
		t.Fatalf("missing echo: %s", stdout.String())
	}
}

func TestInspectHTTPJSON(t *testing.T) {
	d, err := testkit.Named("modern-http")
	if err != nil {
		t.Fatal(err)
	}
	lab := testkit.StartHTTP(t, d)
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--url", lab.Server.URL, "--protocol", "2026-07-28", "inspect", "tools",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"echo"`) {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestJSONDeterministic(t *testing.T) {
	bin := buildLab(t)
	run := func() string {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{
			"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
			"--protocol", "2025-11-25", "inspect",
		}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("code %d stderr=%s", code, stderr.String())
		}
		return stdout.String()
	}
	a, b := run(), run()
	if a != b {
		t.Fatalf("json drifted\n%s\n%s", a, b)
	}
}

func TestTraceDoesNotChangeStdout(t *testing.T) {
	bin := buildLab(t)
	run := func(trace bool) (string, string) {
		var stdout, stderr bytes.Buffer
		args := []string{
			"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
			"--protocol", "2025-11-25", "inspect",
		}
		if trace {
			args = append([]string{"mcpdev", "--trace", "--trace-format", "json"}, args[1:]...)
		}
		code := cli.Run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("trace=%v code %d stderr=%s", trace, code, stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	outOff, errOff := run(false)
	outOn, errOn := run(true)
	if outOff != outOn {
		t.Fatalf("stdout changed\n%s\n%s", outOff, outOn)
	}
	if strings.Contains(errOff, `"kind":"frame"`) {
		t.Fatalf("trace leaked while disabled: %s", errOff)
	}
	if !strings.Contains(errOn, `"kind":"frame"`) {
		t.Fatalf("expected jsonl frames on stderr: %s", errOn)
	}
}

func TestVerboseDoesNotChangeStdout(t *testing.T) {
	bin := buildLab(t)
	run := func(verbose bool) (string, string) {
		var stdout, stderr bytes.Buffer
		args := []string{
			"mcpdev", "--format", "json", "--command", bin, "--env", "MCPDEV_FIXTURE=legacy-stdio",
			"--protocol", "2025-11-25", "inspect",
		}
		if verbose {
			args = append([]string{"mcpdev", "--verbose", "--trace-format", "json"}, args[1:]...)
		}
		code := cli.Run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("verbose=%v code %d stderr=%s", verbose, code, stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	outOff, errOff := run(false)
	outOn, errOn := run(true)
	if outOff != outOn {
		t.Fatalf("stdout changed\n%s\n%s", outOff, outOn)
	}
	if strings.Contains(errOff, `"kind":"frame"`) {
		t.Fatalf("verbose leaked while disabled: %s", errOff)
	}
	if !strings.Contains(errOn, `"kind":"frame"`) {
		t.Fatalf("expected jsonl frames on stderr: %s", errOn)
	}
	if strings.Contains(errOn, `"truncated":true`) {
		t.Fatalf("verbose truncated a frame: %s", errOn)
	}
}

func TestTraceRedactsHeaders(t *testing.T) {
	lab := testkit.StartHTTP(t, mustNamed(t, "modern-http"))
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--trace", "--trace-format", "json",
		"--url", lab.Server.URL, "--protocol", "2026-07-28",
		"--header", "Authorization: Bearer super-secret-token",
		"inspect", "tools",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if strings.Contains(stderr.String(), "super-secret-token") || strings.Contains(stdout.String(), "super-secret-token") {
		t.Fatalf("secret leaked stderr=%s stdout=%s", stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "[REDACTED]") {
		t.Fatalf("expected redaction %s", stderr.String())
	}
}

func mustNamed(t *testing.T, name string) testkit.Descriptor {
	t.Helper()
	d, err := testkit.Named(name)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func buildLab(t *testing.T) string {
	t.Helper()
	lab := testkit.StartProcess(t, "legacy-stdio")
	_ = lab.Close()
	return lab.Cmd.Path
}

func TestProfileHeaderEnvRedacted(t *testing.T) {
	lab := testkit.StartHTTP(t, mustNamed(t, "modern-http"))
	dir := t.TempDir()
	body := "version = 1\n[profiles.http]\nurl = \"" + lab.Server.URL + "\"\n[profiles.http.header_env]\nAuthorization = \"MCPDEV_TOKEN\"\n"
	if err := os.WriteFile(filepath.Join(dir, "mcpdev.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("MCPDEV_TOKEN", "super-secret-token")
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"mcpdev", "--format", "json", "--trace", "--trace-format", "json",
		"--profile", "http", "--protocol", "2026-07-28", "inspect", "tools",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "super-secret-token") {
		t.Fatalf("secret leaked %s", combined)
	}
}
