package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/version"
)

func TestVersionJSONIncludesSchema(t *testing.T) {
	var buf bytes.Buffer
	info := version.Info{
		Version: "1.2.3", Commit: "abc", BuildTime: "2026-01-02T03:04:05Z",
		BuildTimePolicy: version.PolicyGitCommit, GOOS: "linux", GOARCH: "amd64",
		SnapshotSchema: model.SnapshotSchemaVersion,
	}
	if err := VersionJSON(&buf, info); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{`"version":"1.2.3"`, `"commit":"abc"`, `"snapshot_schema":1`, `"build_time_policy":"git_commit"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}
