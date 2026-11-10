package platform

import (
	"path/filepath"
	"runtime"
	"strings"
)

// ExecutableName returns the on-disk binary name for this GOOS.
// Windows appends .exe when the base name has no suffix so go build -o and exec agree.
func ExecutableName(base string) string {
	return ExecutableNameFor(base, runtime.GOOS)
}

// ExecutableNameFor is ExecutableName for an explicit GOOS (release archives).
func ExecutableNameFor(base, goos string) string {
	if goos == "windows" && !strings.HasSuffix(strings.ToLower(base), ".exe") {
		return base + ".exe"
	}
	return base
}

// BuildPath is filepath.Join(dir, ExecutableName(name)).
func BuildPath(dir, name string) string {
	return filepath.Join(dir, ExecutableName(name))
}
