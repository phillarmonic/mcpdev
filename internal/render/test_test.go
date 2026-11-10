package render

import (
	"bytes"
	"strings"
	"testing"

	apptest "github.com/phillarmonic/mcpdev/internal/app/test"
	"github.com/phillarmonic/mcpdev/internal/assert"
	"github.com/phillarmonic/mcpdev/internal/scenario/runner"
)

func TestTestHumanShowsFailingAssertion(t *testing.T) {
	res := apptest.Result{Suites: []runner.Report{{
		File: "suite.yaml", Name: "lab", Status: "fail", Category: "assertion",
		Cases: []runner.CaseReport{{
			Name: "c", Status: "fail", Message: "assertion failed",
			Steps: []runner.StepReport{{
				Op: "call", Status: "fail",
				Assertions: []assert.Result{
					{Kind: "contains", Path: "$.result.text", Status: "pass"},
					{Kind: "contains", Path: "$.result.text", Status: "fail", Diff: `"hello" does not contain "world"`},
				},
			}},
		}},
	}}}
	var buf bytes.Buffer
	if err := TestHuman(&buf, res); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "contains $.result.text") || !strings.Contains(got, "does not contain") {
		t.Fatalf("failing assertion detail missing:\n%s", got)
	}
	if n := strings.Count(got, "contains $.result.text"); n != 1 {
		t.Fatalf("want only the failing assertion printed, got %d:\n%s", n, got)
	}
}
