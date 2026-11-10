package diff

import (
	"context"
	"errors"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Outcome is the CLI-facing comparison result after ignore and threshold policy.
const (
	OutcomeNoChange         = "no-change"
	OutcomeBelowThreshold   = "below-threshold"
	OutcomeThresholdFailure = "threshold-failure"
	OutcomeIndeterminate    = "indeterminate"
)

// Source is one side of a comparison: a snapshot file or a live target.
type Source struct {
	Path   string
	Label  string
	Target port.Target
	Live   bool
}

// Request is one diff command after flag resolution.
type Request struct {
	Ignore          []string
	FailOn          model.DiffSeverity
	From            Source
	To              Source
	AllowIncomplete bool
	ExitOnChange    bool
}

// Result is the classified diff plus policy outcome and side labels.
type Result struct {
	Ignored []model.Change
	Failing []model.Change
	Outcome string
	Diff    model.Diff
	From    Source
	To      Source
}

// Run loads both sources, diffs them, and maps the result to a CLI outcome.
func Run(ctx context.Context, conn port.Connector, store port.SnapshotStore, differ port.Differ, req Request) (Result, error) {
	res := Result{From: req.From, To: req.To, Outcome: OutcomeNoChange}
	from, err := load(ctx, conn, store, req.From)
	if err != nil {
		return res, err
	}
	to, err := load(ctx, conn, store, req.To)
	if err != nil {
		return res, err
	}
	raw, err := differ.Diff(ctx, from, to)
	if err != nil {
		return res, err
	}
	return Classify(raw, req.Ignore, req.FailOn, req.AllowIncomplete, req.ExitOnChange, res)
}

// Classify applies ignore and fail-on policy to an already computed Diff.
func Classify(raw model.Diff, ignore []string, failOn model.DiffSeverity, allowIncomplete, exitOnChange bool, res Result) (Result, error) {
	applied := applyPolicy(raw, ignore, failOn)
	res.Diff = applied.kept
	res.Ignored = applied.ignored
	res.Failing = applied.failing
	switch {
	case raw.Verdict == model.VerdictIndeterminate:
		res.Outcome = OutcomeIndeterminate
		res.Diff.Verdict = model.VerdictIndeterminate
		res.Diff.Changed = false
		if allowIncomplete {
			return res, nil
		}
		return res, errclass.Incomplete{Err: errors.New("diff: incomplete snapshot input")}
	case len(applied.failing) > 0:
		res.Outcome = OutcomeThresholdFailure
		if !exitOnChange {
			return res, nil
		}
		return res, errclass.InterfaceChange{Err: errors.New("diff: public interface changed")}
	case applied.kept.Changed || len(applied.kept.Changes) > 0:
		res.Outcome = OutcomeBelowThreshold
		return res, nil
	default:
		res.Outcome = OutcomeNoChange
		return res, nil
	}
}

func load(ctx context.Context, conn port.Connector, store port.SnapshotStore, src Source) (model.Snapshot, error) {
	if src.Path != "" {
		return store.Read(ctx, src.Path)
	}
	if !src.Live {
		return model.Snapshot{}, errclass.Usage{Msg: "diff: each side needs a snapshot path or a live target"}
	}
	connectCtx := ctx
	if src.Target.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, src.Target.ConnectTimeout)
		defer cancel()
	}
	sess, err := conn.Connect(connectCtx, src.Target)
	if err != nil {
		return model.Snapshot{}, err
	}
	defer func() { _ = sess.Close(ctx) }()
	rpcCtx := ctx
	if src.Target.Timeout > 0 {
		var cancel context.CancelFunc
		rpcCtx, cancel = context.WithTimeout(ctx, src.Target.Timeout)
		defer cancel()
	}
	return store.Capture(rpcCtx, sess, nil)
}
