package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

func testdata(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "config", name)
}

func TestLoadOKAndResolveHeaderEnv(t *testing.T) {
	cwd := t.TempDir()
	src, err := os.ReadFile(testdata(t, "ok.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), src, 0o600); werr != nil {
		t.Fatal(werr)
	}
	env := map[string]string{"MCPDEV_TOKEN": "secret-value"}
	s, err := Load(cwd, env)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ResolveProfile("http", env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Headers["Authorization"] != "secret-value" {
		t.Fatalf("header %q", got.Headers["Authorization"])
	}
	echo := DiagnosticTarget(got)
	if strings.Contains(echo.Headers["Authorization"], "secret-value") {
		t.Fatalf("secret leaked %#v", echo.Headers)
	}
	if echo.Headers["Authorization"] != redact.DefaultReplacement {
		t.Fatalf("want redacted got %q", echo.Headers["Authorization"])
	}
	custom, err := s.ResolveProfile("custom", env)
	if err != nil {
		t.Fatal(err)
	}
	if custom.Connector != "fake" || custom.Kind != "fake" {
		t.Fatalf("custom %+v", custom)
	}
}

func TestInlineSecretNamesProfileAndField(t *testing.T) {
	cwd := t.TempDir()
	src, err := os.ReadFile(testdata(t, "inline-secret.toml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cwd, "mcpdev.toml")
	if werr := os.WriteFile(path, src, 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, err = Load(cwd, map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("class %d", errclass.ExitCode(err))
	}
	msg := err.Error()
	if !strings.Contains(msg, path) || !strings.Contains(msg, "profiles.http.headers.Authorization") {
		t.Fatalf("want profile+field in %q", msg)
	}
	if strings.Contains(msg, "super-secret-token") {
		t.Fatalf("secret in error %q", msg)
	}
}

func TestConflictingProfileAndShellCommand(t *testing.T) {
	for _, name := range []string{"conflict.toml", "shell-command.toml"} {
		cwd := t.TempDir()
		src, err := os.ReadFile(testdata(t, name))
		if err != nil {
			t.Fatal(err)
		}
		if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), src, 0o600); werr != nil {
			t.Fatal(werr)
		}
		_, err = Load(cwd, map[string]string{})
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if errclass.ExitCode(err) != errclass.ExitUsage {
			t.Fatalf("%s class %d", name, errclass.ExitCode(err))
		}
	}
}

func TestUnknownVersion(t *testing.T) {
	cwd := t.TempDir()
	src, err := os.ReadFile(testdata(t, "unknown-version.toml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cwd, "mcpdev.toml")
	if werr := os.WriteFile(path, src, 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, err = Load(cwd, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("got %v", err)
	}
}

func TestMissingInterpolationNamesField(t *testing.T) {
	cwd := t.TempDir()
	src, err := os.ReadFile(testdata(t, "missing-interp.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), src, 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, err = Load(cwd, map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "profiles.remote.headers.Authorization") {
		t.Fatalf("got %q", err)
	}
}

func TestProjectOverridesUser(t *testing.T) {
	home := t.TempDir()
	userDir := filepath.Join(home, "mcpdev")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		t.Fatal(err)
	}
	user := `
version = 1
format = "json"
[profiles.local]
command = ["from-user"]
`
	if werr := os.WriteFile(filepath.Join(userDir, "config.toml"), []byte(user), 0o600); werr != nil {
		t.Fatal(werr)
	}
	cwd := t.TempDir()
	proj := `
version = 1
format = "human"
[profiles.local]
command = ["from-project"]
`
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), []byte(proj), 0o600); werr != nil {
		t.Fatal(werr)
	}
	s, err := Load(cwd, map[string]string{"XDG_CONFIG_HOME": home})
	if err != nil {
		t.Fatal(err)
	}
	if s.Format != "human" {
		t.Fatalf("format %s", s.Format)
	}
	if got := s.Profiles["local"].Command[0]; got != "from-project" {
		t.Fatalf("command %q", got)
	}
}

func TestEnvOverridesProject(t *testing.T) {
	cwd := t.TempDir()
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), []byte("version = 1\nformat = \"human\"\n"), 0o600); werr != nil {
		t.Fatal(werr)
	}
	s, err := Load(cwd, map[string]string{"MCPDEV_FORMAT": "json"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Format != "json" {
		t.Fatalf("format %s", s.Format)
	}
}

func TestInsecureHTTPRequiresOptIn(t *testing.T) {
	cwd := t.TempDir()
	bad := `
version = 1
[profiles.remote]
url = "http://10.0.0.5:8080/mcp"
`
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), []byte(bad), 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, err := Load(cwd, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "profiles.remote.url") {
		t.Fatalf("got %v", err)
	}
	src, err := os.ReadFile(testdata(t, "insecure-http.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), src, 0o600); werr != nil {
		t.Fatal(werr)
	}
	s, err := Load(cwd, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Insecure || len(s.Warnings) == 0 {
		t.Fatalf("want insecure warning %+v", s)
	}
}

func TestPEMBlobRefused(t *testing.T) {
	cwd := t.TempDir()
	doc := "version = 1\n[profiles.remote]\nurl = \"https://mcp.example.com/mcp\"\n[profiles.remote.tls]\nca_file = \"-----BEGIN CERTIFICATE-----\\nMIIB\\n-----END CERTIFICATE-----\"\n"
	if werr := os.WriteFile(filepath.Join(cwd, "mcpdev.toml"), []byte(doc), 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, err := Load(cwd, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "tls.ca_file") {
		t.Fatalf("got %v", err)
	}
}

func TestUserConfigPathWindows(t *testing.T) {
	p, err := userConfigPathGOOS(map[string]string{"APPDATA": `C:\Users\generic\AppData\Roaming`}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(`C:\Users\generic\AppData\Roaming`, "mcpdev", "config.toml")
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}

func TestExplicitConfigMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.toml")
	_, err := Load(t.TempDir(), map[string]string{"MCPDEV_CONFIG": missing})
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("got %v", err)
	}
}

func TestInterpolate(t *testing.T) {
	env := map[string]string{"HOST": "127.0.0.1", "EMPTY": ""}
	got, err := interpolate("http://${HOST}:8080/mcp", "url", env)
	if err != nil || got != "http://127.0.0.1:8080/mcp" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = interpolate("x${MISSING:-y}z", "url", env)
	if err != nil || got != "xyz" {
		t.Fatalf("default %q %v", got, err)
	}
	if _, err := interpolate("${EMPTY:?}", "url", env); err == nil {
		t.Fatal("expected required empty error")
	}
}
