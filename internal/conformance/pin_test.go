package conformance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestLoadPinIsolatesRevisions(t *testing.T) {
	pin, err := LoadPin(moduleRoot())
	if err != nil {
		t.Fatal(err)
	}
	if pin.Package != "@modelcontextprotocol/conformance" {
		t.Fatalf("package %q", pin.Package)
	}
	if pin.Version == "" {
		t.Fatal("empty version")
	}
	if pin.NPXPackage() != pin.Package+"@"+pin.Version {
		t.Fatalf("npx package %q", pin.NPXPackage())
	}
	want := map[string]string{
		"2025-11-25": "legacy-http",
		"2026-07-28": "modern-http",
	}
	if len(pin.Revisions) != len(want) {
		t.Fatalf("got %d revisions, want %d", len(pin.Revisions), len(want))
	}
	for _, rev := range pin.Revisions {
		fixture, ok := want[rev.Requirements]
		if !ok {
			t.Fatalf("unexpected requirements %q", rev.Requirements)
		}
		if rev.Fixture != fixture {
			t.Fatalf("%s fixture %q, want %q", rev.Requirements, rev.Fixture, fixture)
		}
		d, err := testkit.Named(rev.Fixture)
		if err != nil {
			t.Fatal(err)
		}
		if string(d.Era) != rev.Requirements {
			t.Fatalf("fixture %s era %s, requirements %s", rev.Fixture, d.Era, rev.Requirements)
		}
		if _, err := os.Stat(filepath.Join(moduleRoot(), rev.Baseline)); err != nil {
			t.Fatalf("baseline %s: %v", rev.Baseline, err)
		}
		args := ServerArgs("http://127.0.0.1:9/mcp", rev.Requirements, rev.Baseline, "artifacts/conformance/"+rev.Requirements)
		for _, a := range args {
			if strings.Contains(a, ",") {
				t.Fatalf("requirements must be a single revision, got %v", args)
			}
		}
	}
}

func TestRevisionByRequirementsUnknown(t *testing.T) {
	pin, err := LoadPin(moduleRoot())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pin.RevisionByRequirements("1999-01-01"); err == nil {
		t.Fatal("expected error")
	}
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("conformance: runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
