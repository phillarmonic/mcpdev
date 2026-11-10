package cli

import (
	"context"
	"io"
	"os"
	"os/signal"
	"strings"

	diffapp "github.com/phillarmonic/mcpdev/internal/app/diff"
	"github.com/phillarmonic/mcpdev/internal/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/render"
	"github.com/phillarmonic/mcpdev/internal/snapshot"
	"github.com/phillarmonic/mcpdev/internal/version"
)

func runDiff(in Invocation, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	from, to, format, err := resolveDiffSources(in)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	failOn, err := diffapp.ParseFailOn(in.FailOn)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	renames, err := parseRenames(in.Renames)
	if err != nil {
		return fail(stdout, stderr, in, err)
	}
	store := snapshot.Store{Version: version.Version}
	res, runErr := diffapp.Run(ctx, newConnector(in, stderr, nil), store, diff.Engine{Renames: renames}, diffapp.Request{
		From:            from,
		To:              to,
		Ignore:          in.Ignore,
		FailOn:          failOn,
		AllowIncomplete: in.AllowIncomplete,
		ExitOnChange:    in.ExitOnChange,
	})
	if format == model.FormatJSON {
		if err := render.DiffJSON(stdout, res, runErr); err != nil {
			_ = writeln(stderr, err.Error())
			return errclass.ExitInternal
		}
	} else if runErr == nil || len(res.Diff.Changes) > 0 || res.Outcome != "" {
		if err := render.DiffHuman(stdout, res); err != nil {
			return errclass.ExitInternal
		}
	}
	if runErr != nil {
		_ = writeln(stderr, runErr.Error())
		return errclass.ExitCode(runErr)
	}
	return errclass.ExitOK
}

func resolveDiffSources(in Invocation) (diffapp.Source, diffapp.Source, model.Format, error) {
	format, err := loadFormat(in)
	if err != nil {
		return diffapp.Source{}, diffapp.Source{}, format, err
	}
	fromPath := in.From
	if fromPath == "" {
		fromPath = in.Section
	}
	toPath := in.To
	if toPath == "" {
		toPath = in.ToPath
	}
	fromLive := liveSpec(in.FromCommand, in.FromURL, in.FromProfile)
	toLive := liveSpec(in.ToCommand, in.ToURL, in.ToProfile)
	globalLive := liveSpec(in.CommandArgv, in.URL, in.Profile)
	if fromPath != "" && fromLive {
		return diffapp.Source{}, diffapp.Source{}, format, errclass.Usage{Msg: "diff: --from cannot be combined with --from-command, --from-url, or --from-profile"}
	}
	if toPath != "" && toLive {
		return diffapp.Source{}, diffapp.Source{}, format, errclass.Usage{Msg: "diff: --to cannot be combined with --to-command, --to-url, or --to-profile"}
	}
	var from, to diffapp.Source
	if fromPath != "" {
		from = diffapp.Source{Path: fromPath, Label: fromPath}
	}
	if toPath != "" {
		to = diffapp.Source{Path: toPath, Label: toPath}
	}
	if fromLive {
		target, _, specErr := resolveTargetSpec(in, in.FromCommand, in.FromURL, in.FromProfile, false, true)
		if specErr != nil {
			return diffapp.Source{}, diffapp.Source{}, format, specErr
		}
		from = liveSource(target)
	}
	if toLive {
		target, _, specErr := resolveTargetSpec(in, in.ToCommand, in.ToURL, in.ToProfile, false, true)
		if specErr != nil {
			return diffapp.Source{}, diffapp.Source{}, format, specErr
		}
		to = liveSource(target)
	}
	switch {
	case sourceReady(from) && sourceReady(to):
	case sourceReady(from) && !sourceReady(to) && globalLive:
		target, _, specErr := resolveTarget(in)
		if specErr != nil {
			return diffapp.Source{}, diffapp.Source{}, format, specErr
		}
		to = liveSource(target)
	case !sourceReady(from) && sourceReady(to) && globalLive:
		target, _, specErr := resolveTarget(in)
		if specErr != nil {
			return diffapp.Source{}, diffapp.Source{}, format, specErr
		}
		from = liveSource(target)
	default:
		return diffapp.Source{}, diffapp.Source{}, format, errclass.Usage{Msg: "diff: two snapshot paths or one path plus a live target, or --from-* and --to-* live sides"}
	}
	return from, to, format, nil
}

func liveSpec(cmd []string, url, profile string) bool {
	return len(cmd) > 0 || url != "" || profile != ""
}

func sourceReady(src diffapp.Source) bool {
	return src.Path != "" || src.Live
}

func liveSource(t port.Target) diffapp.Source {
	label := t.Kind
	switch {
	case t.Profile != "":
		label = "profile " + t.Profile
	case len(t.Command) > 0:
		label = strings.Join(t.Command, " ")
	case t.URL != "":
		label = t.URL
	}
	return diffapp.Source{Target: t, Label: label, Live: true}
}

func parseRenames(vals []string) ([]model.Rename, error) {
	return diffapp.ParseRenames(vals)
}
