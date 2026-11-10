package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

func runHooks(ctx context.Context, hooks []schema.Hook, b *bindings, workDir string) error {
	for i, h := range hooks {
		if err := ctx.Err(); err != nil {
			return errclass.Cleanup{Err: err}
		}
		if h.Snapshot != nil {
			continue
		}
		if h.Exec == nil {
			continue
		}
		argv := make([]string, 0, len(h.Exec.Argv))
		for _, a := range h.Exec.Argv {
			s, err := b.interpString(a)
			if err != nil {
				return errclass.Cleanup{Err: err}
			}
			argv = append(argv, s)
		}
		if len(argv) == 0 {
			return errclass.Cleanup{Err: fmt.Errorf("hook %d: empty argv", i)}
		}
		dir := h.Exec.CWD
		if dir == "" {
			dir = workDir
		} else if !filepath.IsAbs(dir) {
			dir = filepath.Join(workDir, dir)
		}
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // argv is from the local suite, never server output
		cmd.Dir = dir
		cmd.Env = os.Environ()
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return errclass.Cleanup{Err: fmt.Errorf("hook %d %s: %w", i, argv[0], err)}
		}
	}
	return nil
}
