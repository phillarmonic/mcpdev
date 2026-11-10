package cycle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Evaluate runs the configured readiness policy, optional smoke, snapshot, and diff.
// On error the last ready snapshot is left unchanged.
func (e *Engine) Evaluate(ctx context.Context, sess port.Session) (Result, error) {
	start := e.now()
	if err := ctx.Err(); err != nil {
		return e.fail(start, "canceled", errclass.Connection{Err: err})
	}
	if e.store == nil {
		return e.fail(start, "misconfigured", errclass.Usage{Msg: "watch cycle: snapshot store required"})
	}
	if e.cfg.Readiness == ReadinessScenario {
		if len(e.cfg.ReadinessScenarios) == 0 {
			return e.fail(start, "misconfigured", errclass.Usage{Msg: "watch cycle: scenario readiness needs names"})
		}
		if err := e.runSmoke(ctx, sess, e.cfg.ReadinessScenarios); err != nil {
			return e.fail(start, "readiness_scenario", err)
		}
	}
	snap, err := e.store.Capture(ctx, sess, nil)
	if err != nil {
		return e.fail(start, "capture", err)
	}
	if snap.Incomplete {
		reason := strings.Join(snap.IncompleteReasons, ", ")
		if reason == "" {
			reason = "incomplete collection"
		}
		return e.fail(start, "inventory", errclass.Incomplete{Err: errors.New("watch cycle: " + reason)})
	}
	if len(e.cfg.Smoke) > 0 {
		if err := e.runSmoke(ctx, sess, e.cfg.Smoke); err != nil {
			return e.fail(start, "smoke", err)
		}
	}
	res := Result{
		Snapshot: snap,
		Ready:    true,
		Duration: e.now().Sub(start),
	}
	prev, ok := e.snapshotLocked()
	if !ok {
		res.Outcome = OutcomeUnchanged
		res.Diff = model.Diff{Verdict: model.VerdictEqual, Changes: []model.Change{}}
		res.Message = "ready (baseline)"
	} else {
		if e.differ == nil {
			return e.fail(start, "misconfigured", errclass.Usage{Msg: "watch cycle: differ required after baseline"})
		}
		raw, diffErr := e.differ.Diff(ctx, prev, snap)
		if diffErr != nil {
			return e.fail(start, "diff", diffErr)
		}
		res.Diff = filterIgnored(raw, e.cfg.Ignore)
		res.Outcome = rollup(res.Diff)
		res.Message = string(res.Outcome)
	}
	fp := fingerprint(res.Outcome, res.Diff)
	e.mu.Lock()
	res.Repeated = e.lastFingerprint != "" && fp == e.lastFingerprint
	e.mu.Unlock()
	e.commitReady(res, fp)
	if e.onReport != nil && !res.Repeated {
		e.onReport(res)
	}
	return res, nil
}

func (e *Engine) runSmoke(ctx context.Context, sess port.Session, names []string) error {
	if e.smoke == nil {
		return errclass.Usage{Msg: "watch cycle: smoke runner required"}
	}
	return e.smoke.Run(ctx, sess, names)
}

func (e *Engine) fail(start time.Time, reason string, err error) (Result, error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	e.commitFail(reason, e.now().Sub(start), OutcomeIndeterminate, msg)
	return Result{Duration: e.now().Sub(start), Message: msg, Outcome: OutcomeIndeterminate}, err
}
