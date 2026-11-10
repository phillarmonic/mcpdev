package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func integrationDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "integration")
}

func TestCommittedAssetsHaveNoPersonalPaths(t *testing.T) {
	root := integrationDir(t)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(b)
		for _, needle := range []string{"/Users/", "/home/", "C:\\Users\\", "andersonpem"} {
			if strings.Contains(text, needle) {
				t.Errorf("%s contains %q", path, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	tutorial := filepath.Join(filepath.Dir(root), "..", "docs", "tutorials", "stdio-loop.md")
	b, err := os.ReadFile(tutorial)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, needle := range []string{"/Users/", "/home/", "C:\\Users\\", "andersonpem"} {
		if strings.Contains(text, needle) {
			t.Errorf("%s contains %q", tutorial, needle)
		}
	}
}
