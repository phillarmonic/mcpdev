package runner

import (
	"context"
	"fmt"
	"maps"
	"os"
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

// Run executes one suite. Cases may run in parallel; steps inside a case stay serial.
func Run(ctx context.Context, deps Deps, suite schema.Suite, req Request) Report {
	clk := deps.clock()
	start := clk.Now()
	rep := Report{File: req.File, Name: suite.Name, Status: statusPass, Cases: make([]CaseReport, len(suite.Cases))}
	b, err := resolveBindings(suite, req.Env)
	if err != nil {
		rep.Status = statusError
		rep.Category = errclass.Category(err)
		rep.Message = err.Error()
		rep.DurationMS = clk.Now().Sub(start).Milliseconds()
		return rep
	}
	workDir := req.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	if err := runHooks(ctx, suite.Setup, b, workDir); err != nil {
		rep.Status = statusError
		rep.Category = errclass.Category(err)
		rep.Message = err.Error()
		_ = runHooks(context.WithoutCancel(ctx), suite.Cleanup, b, workDir)
		rep.DurationMS = clk.Now().Sub(start).Milliseconds()
		return rep
	}

	jobs := req.Jobs
	if jobs < 1 {
		jobs = 4
	}
	if jobs > len(suite.Cases) {
		jobs = len(suite.Cases)
	}
	if jobs < 1 {
		jobs = 1
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i, c := range suite.Cases {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, c schema.Case) {
			defer wg.Done()
			defer func() { <-sem }()
			rep.Cases[i] = runCase(ctx, deps, suite, req.Target, c, cloneBindings(b), workDir)
		}(i, c)
	}
	wg.Wait()

	cleanupErr := runHooks(context.WithoutCancel(ctx), suite.Cleanup, b, workDir)
	for _, c := range rep.Cases {
		applySuiteStatus(&rep, c.Status, c.Category, c.Message)
	}
	if cleanupErr != nil {
		rep.Status = statusError
		rep.Category = "cleanup"
		rep.Message = cleanupErr.Error()
	}
	rep.DurationMS = clk.Now().Sub(start).Milliseconds()
	return rep
}

func runCase(ctx context.Context, deps Deps, suite schema.Suite, target port.Target, c schema.Case, b *bindings, workDir string) (out CaseReport) {
	clk := deps.clock()
	start := clk.Now()
	out = CaseReport{Name: c.Name, Status: statusPass}
	defer func() {
		out.DurationMS = clk.Now().Sub(start).Milliseconds()
		if rec := recover(); rec != nil {
			out.Status = statusError
			out.Category = "internal"
			out.Message = fmt.Sprintf("panic: %v", rec)
		}
	}()
	timeout := target.Timeout
	if c.Timeout != "" {
		d, err := time.ParseDuration(c.Timeout)
		if err == nil {
			timeout = d
		}
	} else if suite.Defaults.Timeout != "" {
		d, err := time.ParseDuration(suite.Defaults.Timeout)
		if err == nil && timeout == 0 {
			timeout = d
		}
	}
	caseTarget := target
	caseTarget.Timeout = timeout
	if suite.Defaults.ConnectTimeout != "" && caseTarget.ConnectTimeout == 0 {
		if d, err := time.ParseDuration(suite.Defaults.ConnectTimeout); err == nil {
			caseTarget.ConnectTimeout = d
		}
	}
	connectCtx := ctx
	var cancel context.CancelFunc
	if caseTarget.ConnectTimeout > 0 {
		connectCtx, cancel = context.WithTimeout(ctx, caseTarget.ConnectTimeout)
		defer cancel()
	}
	sess, err := deps.Connector.Connect(connectCtx, caseTarget)
	if err != nil {
		out.Status = statusError
		out.Category = errclass.Category(err)
		out.Message = err.Error()
		return out
	}
	defer func() {
		if closeErr := sess.Close(context.WithoutCancel(ctx)); closeErr != nil && out.Status == statusPass {
			out.Status = statusError
			out.Category = "cleanup"
			out.Message = closeErr.Error()
		}
	}()

	caseCtx := ctx
	if timeout > 0 {
		var stop context.CancelFunc
		caseCtx, stop = context.WithTimeout(ctx, timeout)
		defer stop()
	}
	stopRest := false
	for _, st := range c.Steps {
		if stopRest {
			out.Steps = append(out.Steps, StepReport{ID: st.ID, Op: stepOp(st), Status: statusSkip, Message: "skipped after failure"})
			continue
		}
		if err := caseCtx.Err(); err != nil {
			out.Steps = append(out.Steps, StepReport{ID: st.ID, Op: stepOp(st), Status: statusError, Category: "connection", Message: err.Error()})
			out.Status = statusError
			out.Category = "connection"
			out.Message = err.Error()
			stopRest = true
			continue
		}
		step := runStep(caseCtx, deps, sess, caseTarget, st, b, workDir)
		out.Steps = append(out.Steps, step)
		if step.Status == statusPass {
			continue
		}
		stopRest = true
		out.Status = step.Status
		out.Category = step.Category
		out.Message = step.Message
	}
	return out
}

func applySuiteStatus(rep *Report, status, category, message string) {
	switch status {
	case statusPass, "":
		return
	case statusFail:
		if rep.Status == statusPass {
			rep.Status = statusFail
			rep.Category = category
			rep.Message = message
		}
		if category == "incomplete" {
			rep.Category = "incomplete"
			rep.Message = message
		}
	default:
		rep.Status = statusError
		rep.Category = category
		rep.Message = message
	}
}

func resolveBindings(suite schema.Suite, env map[string]string) (*bindings, error) {
	b := &bindings{
		vars:    map[string]string{},
		secrets: map[string]string{},
		capture: map[string]any{},
	}
	maps.Copy(b.vars, suite.Variables)
	for k, s := range suite.Secrets {
		val, ok := env[s.FromEnv]
		if !ok || val == "" {
			return nil, errclass.Usage{Msg: "missing environment variable " + s.FromEnv + " for secret " + k}
		}
		b.secrets[k] = val
	}
	return b, nil
}

func cloneBindings(b *bindings) *bindings {
	out := &bindings{
		vars:    map[string]string{},
		secrets: map[string]string{},
		capture: map[string]any{},
	}
	maps.Copy(out.vars, b.vars)
	maps.Copy(out.secrets, b.secrets)
	return out
}
