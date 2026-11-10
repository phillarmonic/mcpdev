package cli

import (
	"context"
	"io"
	"os"
	"os/signal"

	snapapp "github.com/phillarmonic/mcpdev/internal/app/snapshot"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/snapshot"
	"github.com/phillarmonic/mcpdev/internal/version"
)

func runSnapshot(in Invocation, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	target, format, err := resolveTarget(in)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	kinds, err := parseSection(in.Section)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	store := snapshot.Store{Version: version.Version, Target: target}
	res, runErr := snapapp.Run(ctx, newConnector(in, stderr, target.Sensitive), store, snapapp.Request{
		Target:          target,
		Kinds:           kinds,
		Out:             in.Out,
		AllowIncomplete: in.AllowIncomplete,
	})
	if res.Snapshot.SchemaVersion != 0 {
		body, encErr := snapshot.Encode(res.Snapshot)
		if encErr != nil && runErr == nil {
			runErr = encErr
		}
		if encErr == nil {
			if _, err := stdout.Write(body); err != nil {
				return errclass.ExitInternal
			}
		}
	}
	if res.Out != "" && format != model.FormatJSON {
		_ = writeln(stderr, "wrote "+res.Out)
	}
	if runErr != nil {
		_ = writeln(stderr, runErr.Error())
		return errclass.ExitCode(runErr)
	}
	return errclass.ExitOK
}
