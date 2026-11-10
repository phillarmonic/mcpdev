package watch

import (
	"path"
	"path/filepath"
	"strings"
)

// DefaultExcludes are directory names and globs ignored unless the caller overrides them.
var DefaultExcludes = []string{
	".git",
	".hg",
	".svn",
	"bin",
	"dist",
	"site",
	"graphify-out",
	"tmp",
	"temp",
	".tmp",
	".zensical",
	"vendor",
	"node_modules",
	".idea",
	".vscode",
	"*.exe",
	"*.test",
	"*.out",
}

// Matcher applies include/exclude globs to slash-normalized relative paths.
type Matcher struct {
	includes []string
	excludes []string
}

// NewMatcher builds a matcher. Empty includes means every non-excluded path is watched.
func NewMatcher(includes, excludes []string) Matcher {
	if len(excludes) == 0 {
		excludes = append([]string{}, DefaultExcludes...)
	}
	return Matcher{
		includes: append([]string{}, includes...),
		excludes: append([]string{}, excludes...),
	}
}

// Accept reports whether a relative path should trigger a rebuild.
func (m Matcher) Accept(rel string) bool {
	rel = cleanRel(rel)
	if rel == "" || rel == "." {
		return false
	}
	if editorNoise(rel) {
		return false
	}
	if excluded(rel, m.excludes) {
		return false
	}
	if len(m.includes) == 0 {
		return true
	}
	for _, pattern := range m.includes {
		if matchGlob(pattern, rel) {
			return true
		}
	}
	return false
}

// SkipDir reports whether a directory should not be recursively watched.
func (m Matcher) SkipDir(rel string) bool {
	rel = cleanRel(rel)
	if rel == "" || rel == "." {
		return false
	}
	return excluded(rel, m.excludes)
}

func cleanRel(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = path.Clean(rel)
	return strings.TrimPrefix(rel, "./")
}

func excluded(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchGlob(pattern, rel) {
			return true
		}
	}
	base := path.Base(rel)
	for part := range strings.SplitSeq(rel, "/") {
		for _, pattern := range patterns {
			if !strings.ContainsAny(pattern, "*?/") && part == pattern {
				return true
			}
			if matchGlob(pattern, base) {
				return true
			}
		}
	}
	return false
}

func editorNoise(rel string) bool {
	base := path.Base(rel)
	switch {
	case strings.HasSuffix(base, "~"):
		return true
	case strings.HasPrefix(base, ".#"):
		return true
	case strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#"):
		return true
	case strings.HasSuffix(base, ".swp"), strings.HasSuffix(base, ".swo"), strings.HasSuffix(base, ".swx"):
		return true
	case strings.HasSuffix(base, ".tmp"), strings.HasSuffix(base, ".temp"):
		return true
	case base == ".DS_Store":
		return true
	}
	return isVimSwapNumber(base)
}

func isVimSwapNumber(base string) bool {
	if len(base) < 3 || len(base) > 4 {
		return false
	}
	for i := 0; i < len(base); i++ {
		if base[i] < '0' || base[i] > '9' {
			return false
		}
	}
	return true
}

func matchGlob(pattern, rel string) bool {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	rel = filepath.ToSlash(rel)
	if pattern == "**" {
		return true
	}
	if suffix, ok := strings.CutPrefix(pattern, "**/"); ok {
		if matchOne(suffix, rel) || matchOne(suffix, path.Base(rel)) {
			return true
		}
		for _, dir := range parentDirs(rel) {
			if matchOne(suffix, strings.TrimPrefix(rel, dir+"/")) {
				return true
			}
		}
	}
	if prefix, ok := strings.CutSuffix(pattern, "/**"); ok {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return matchOne(pattern, rel) || matchOne(pattern, path.Base(rel))
}

func matchOne(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}

func parentDirs(rel string) []string {
	var out []string
	dir := path.Dir(rel)
	for dir != "." && dir != "/" && dir != "" {
		out = append(out, dir)
		next := path.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return out
}
