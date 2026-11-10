package watch

import (
	"path/filepath"
	"strings"
)

func relativize(roots []string, name string) (string, string) {
	abs, err := filepath.Abs(name)
	if err != nil {
		abs = name
	}
	for _, root := range roots {
		rel, relErr := filepath.Rel(root, abs)
		if relErr != nil {
			continue
		}
		if rel == "." || rel == "" {
			return root, rel
		}
		if strings.HasPrefix(rel, "..") {
			continue
		}
		return root, rel
	}
	return "", abs
}
