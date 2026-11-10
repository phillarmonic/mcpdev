package test_test

import (
	"context"
	"path/filepath"
	"testing"

	apptest "github.com/phillarmonic/mcpdev/internal/app/test"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/scenario/runner"
)

func TestInvalidFileIsUsageBeforeConnect(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "scenarios", "invalid", "shell-step.yaml")
	_, err := apptest.Run(context.Background(), runner.Deps{}, apptest.Request{Paths: []string{path}})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("%v", err)
	}
}

func TestMissingPaths(t *testing.T) {
	_, err := apptest.Run(context.Background(), runner.Deps{}, apptest.Request{})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("%v", err)
	}
}
