package platform

import "testing"

func TestExecutableNameWindows(t *testing.T) {
	if got := ExecutableNameFor("mcpdev", "windows"); got != "mcpdev.exe" {
		t.Fatalf("got %q", got)
	}
	if got := ExecutableNameFor("mcpdev.exe", "windows"); got != "mcpdev.exe" {
		t.Fatalf("double suffix %q", got)
	}
	if got := ExecutableNameFor("mcpdev", "linux"); got != "mcpdev" {
		t.Fatalf("unix %q", got)
	}
}
