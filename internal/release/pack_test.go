package release

import "testing"

func TestArchiveName(t *testing.T) {
	if got := ArchiveName("v1.2.3", "linux", "amd64"); got != "mcpdev_1.2.3_linux_amd64.tar.gz" {
		t.Fatalf("linux %s", got)
	}
	if got := ArchiveName("1.2.3", "windows", "amd64"); got != "mcpdev_1.2.3_windows_amd64.zip" {
		t.Fatalf("windows %s", got)
	}
}

func TestDefaultTargetsCoverSupportedOS(t *testing.T) {
	seen := map[string]bool{}
	for _, tgt := range DefaultTargets() {
		seen[tgt.GOOS] = true
	}
	for _, osName := range []string{"linux", "darwin", "windows"} {
		if !seen[osName] {
			t.Fatalf("missing %s", osName)
		}
	}
}
