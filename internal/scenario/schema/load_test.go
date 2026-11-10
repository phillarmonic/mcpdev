package schema

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func fixture(t *testing.T, elem ...string) string {
	t.Helper()
	parts := append([]string{"..", "..", "..", "testdata", "scenarios"}, elem...)
	return filepath.Join(parts...)
}

func TestLoadValidProfileYAML(t *testing.T) {
	s, err := LoadFile(fixture(t, "valid", "stdio-profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "stdio-profile" || s.Target.Profile != "lab" {
		t.Fatalf("%+v", s)
	}
	if s.Secrets["token"].FromEnv != "MCPDEV_LAB_TOKEN" {
		t.Fatal("secret must be env ref only")
	}
	if len(s.Cases) != 1 || len(s.Cases[0].Steps) != 6 {
		t.Fatalf("cases %+v", s.Cases)
	}
	if !strings.Contains(string(s.Cases[0].Steps[1].Call.Arguments), "${secrets.token}") {
		t.Fatal("loaded suite dropped secret interpolation")
	}
	if s.Cases[0].Steps[4].Read.Template != "fixture://items/{id}" {
		t.Fatalf("template %q", s.Cases[0].Steps[4].Read.Template)
	}
	rep, err := ReportJSON(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rep), "s3cret") {
		t.Fatalf("report leaked a secret value: %s", rep)
	}
	if !strings.Contains(string(rep), "[REDACTED]") {
		t.Fatalf("report should redact secret-shaped keys: %s", rep)
	}
}

func TestLoadValidJSON(t *testing.T) {
	s, err := LoadFile(fixture(t, "valid", "stdio.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Target.Command) != 3 {
		t.Fatalf("command %+v", s.Target.Command)
	}
}

func TestLoadCallArgumentsNamedCommand(t *testing.T) {
	s, err := LoadFile(fixture(t, "valid", "call-command-arg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Cases[0].Steps[0].Call.Tool != "echo" {
		t.Fatalf("tool %q", s.Cases[0].Steps[0].Call.Tool)
	}
	args := string(s.Cases[0].Steps[0].Call.Arguments)
	if !strings.Contains(args, "command") || !strings.Contains(args, "vendor/bin/phpunit") {
		t.Fatalf("arguments %s", args)
	}
	if s.Cases[0].Steps[1].Call.Tool != "fail" {
		t.Fatalf("tool %q", s.Cases[0].Steps[1].Call.Tool)
	}
	if s.Cases[0].Steps[1].ExpectFailure == nil || s.Cases[0].Steps[1].ExpectFailure.Category != "tool_error" {
		t.Fatalf("expect_failure %+v", s.Cases[0].Steps[1].ExpectFailure)
	}
}

func TestInvalidManyReportsAll(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "many.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("class %d", errclass.ExitCode(err))
	}
	var se Error
	if !errors.As(err, &se) {
		t.Fatalf("want schema.Error, got %T %v", err, err)
	}
	got := se.Error()
	for _, need := range []string{
		"schema_version",
		"/name",
		"/target",
		"/shell",
		"/extra",
		"/secrets/token/value",
		"/cases",
	} {
		if !strings.Contains(got, need) {
			t.Errorf("missing %s in %s", need, got)
		}
	}
	if strings.Contains(got, "s3cret-must-not-load") {
		t.Fatal("validation error serialized an inline secret value")
	}
}

func TestInvalidShellStep(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "shell-step.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "/cases/0/steps/0/run") {
		t.Fatal(err)
	}
}

func TestInvalidTwoOps(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "two-ops.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "exactly one of inspect") {
		t.Fatal(err)
	}
}

func TestInvalidCommandString(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "command-string.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "argv array") {
		t.Fatal(err)
	}
}

func TestInvalidCommandOnStep(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "step-command.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if !strings.Contains(got, "/cases/0/steps/0/command") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "command is only allowed on target as an argv array") {
		t.Fatal(got)
	}
}

func TestInvalidUnknownInterp(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "unknown-interp.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown variable or capture ${missing}") {
		t.Fatal(err)
	}
}

func TestLoadValidAssertInterp(t *testing.T) {
	s, err := LoadFile(fixture(t, "valid", "assert-interp.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	a := s.Cases[0].Steps[1].Assert
	if len(a) != 3 {
		t.Fatalf("assertions %+v", a)
	}
	if got, _ := a[0].Contains.Value.(string); got != "${echoed}" {
		t.Fatalf("contains value %v", a[0].Contains.Value)
	}
}

func TestInvalidUnknownAssertInterp(t *testing.T) {
	_, err := LoadFile(fixture(t, "invalid", "assert-interp.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if !strings.Contains(got, "/cases/0/steps/0/assert/0/contains/value") {
		t.Fatalf("path missing: %s", got)
	}
	if !strings.Contains(got, "unknown variable or capture ${missing}") {
		t.Fatal(got)
	}
}

func TestFormalSchemaPublished(t *testing.T) {
	if len(FormalJSON) < 100 || !strings.Contains(string(FormalJSON), "schema_version") {
		t.Fatal("suite.v1.json missing")
	}
}

func TestEmptyDocument(t *testing.T) {
	_, err := Load(nil, "empty.yaml")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatal(err)
	}
}
