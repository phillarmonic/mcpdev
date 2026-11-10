package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/cli"
	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func TestVersionJSONReportsSchema(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"mcpdev", "--format", "json", "version"}, &stdout, &stderr)
	if code != errclass.ExitOK {
		t.Fatalf("code %d stderr=%s", code, stderr.String())
	}
	body := stdout.String()
	for _, want := range []string{
		`"ok":true`,
		`"command":"version"`,
		`"snapshot_schema":1`,
		`"build_time_policy"`,
		`"commit"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
}
