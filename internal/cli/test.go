package cli

import (
	"context"
	"io"
	"os"
	"os/signal"

	apptest "github.com/phillarmonic/mcpdev/internal/app/test"
	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/render"
	"github.com/phillarmonic/mcpdev/internal/scenario/runner"
	"github.com/phillarmonic/mcpdev/internal/snapshot"
	"github.com/phillarmonic/mcpdev/internal/version"
)

func runTest(in Invocation, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	target, format, err := resolveTargetSpec(in, in.CommandArgv, in.URL, in.Profile, true, false)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	if !in.FormatSet {
		loaded, fmtErr := loadFormat(in)
		if fmtErr != nil {
			return fail(stdout, stderr, in, fmtErr)
		}
		format = loaded
	}
	hasOverlay := len(target.Command) > 0 || target.URL != "" || target.Profile != ""
	cwd, err := os.Getwd()
	if err != nil {
		return fail(stdout, stderr, in, errclass.Internal{Err: err})
	}
	deps := runner.Deps{
		Connector: newConnector(in, stderr, target.Sensitive),
		Store:     snapshot.Store{Version: version.Version, Target: target, Clock: clock.Wall{}},
		Differ:    diff.Engine{},
		Clock:     clock.Wall{},
	}
	res, runErr := apptest.Run(ctx, deps, apptest.Request{
		Paths:           in.Paths,
		WorkDir:         cwd,
		Overlay:         target,
		HasOverlay:      hasOverlay,
		Env:             envMap(os.Environ()),
		Jobs:            in.Jobs,
		AllowIncomplete: in.AllowIncomplete,
	})
	if format == model.FormatJSON {
		if err := render.TestJSON(stdout, res, runErr); err != nil {
			_ = writeln(stderr, err.Error())
			return errclass.ExitInternal
		}
	} else if err := render.TestHuman(stdout, res); err != nil {
		return errclass.ExitInternal
	}
	if in.JUnit != "" {
		f, err := os.Create(in.JUnit)
		if err != nil {
			_ = writeln(stderr, err.Error())
			return errclass.ExitInternal
		}
		writeErr := render.TestJUnit(f, res)
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil {
			return errclass.ExitInternal
		}
	}
	if runErr != nil {
		_ = writeln(stderr, runErr.Error())
		return errclass.ExitCode(runErr)
	}
	return errclass.ExitOK
}
