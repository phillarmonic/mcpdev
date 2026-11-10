package watch

import "testing"

func TestMatcherDefaultExcludes(t *testing.T) {
	m := NewMatcher(nil, nil)
	skip := []string{
		".git/HEAD",
		"bin/mcpdev",
		"dist/SHA256SUMS",
		"site/index.html",
		"graphify-out/graph.json",
		"tmp/out",
		"internal/watch/loop.go~",
		"cmd/mcpdev/.foo.swp",
		"4913",
	}
	for _, path := range skip {
		if m.Accept(path) {
			t.Fatalf("accepted %s", path)
		}
	}
	if !m.Accept("internal/watch/loop.go") {
		t.Fatal("expected source path")
	}
	if !m.SkipDir(".git") || !m.SkipDir("bin") {
		t.Fatal("expected skip for vcs and bin")
	}
}

func TestMatcherIncludes(t *testing.T) {
	m := NewMatcher([]string{"internal/**", "**/*.go"}, nil)
	if !m.Accept("internal/watch/loop.go") {
		t.Fatal("internal/**")
	}
	if !m.Accept("cmd/mcpdev/main.go") {
		t.Fatal("**/*.go")
	}
	if m.Accept("README.md") {
		t.Fatal("md should miss includes")
	}
}

func TestMatcherCustomGenerated(t *testing.T) {
	m := NewMatcher(nil, append(append([]string{}, DefaultExcludes...), "generated/**"))
	if m.Accept("generated/wire.go") {
		t.Fatal("configured generated path")
	}
}
