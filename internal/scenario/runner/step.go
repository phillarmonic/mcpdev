package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"

	json "encoding/json/v2"

	diffapp "github.com/phillarmonic/mcpdev/internal/app/diff"
	"github.com/phillarmonic/mcpdev/internal/app/inspect"
	"github.com/phillarmonic/mcpdev/internal/app/invoke"
	"github.com/phillarmonic/mcpdev/internal/assert"
	"github.com/phillarmonic/mcpdev/internal/diff"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

func runStep(ctx context.Context, deps Deps, sess port.Session, target port.Target, st schema.Step, b *bindings, workDir string) StepReport {
	clk := deps.clock()
	start := clk.Now()
	rep := StepReport{ID: st.ID, Op: stepOp(st), Status: statusPass}
	doc := map[string]any{"ok": true, "result": map[string]any{}}
	mime := ""
	var stepErr error

	switch {
	case st.Inspect != nil:
		doc, stepErr = doInspect(ctx, sess, target, *st.Inspect)
	case st.Call != nil:
		doc, mime, stepErr = doCall(ctx, sess, target, *st.Call, b)
	case st.Prompt != nil:
		doc, mime, stepErr = doPrompt(ctx, sess, target, *st.Prompt, b)
	case st.Read != nil:
		doc, mime, stepErr = doRead(ctx, sess, target, *st.Read, b)
	case st.Snapshot != nil:
		doc, stepErr = doSnapshot(ctx, deps, sess, target, *st.Snapshot, workDir)
	default:
		stepErr = errclass.Usage{Msg: "step has no operation"}
	}

	dur := clk.Now().Sub(start)
	rep.DurationMS = dur.Milliseconds()
	cat := errclass.Category(stepErr)
	if st.Call != nil {
		if tool, ok := doc["result"].(map[string]any); ok {
			if isErr, _ := tool["isError"].(bool); isErr && cat == "" {
				cat = "tool_error"
				stepErr = errclass.ToolError{Name: st.Call.Tool}
			}
		}
	}

	if st.ExpectFailure != nil {
		want := st.ExpectFailure.Category
		if stepErr == nil || cat != want {
			rep.Status = statusFail
			rep.Category = "assertion"
			rep.Message = fmt.Sprintf("expected failure %s, got %s", want, cat)
		} else {
			doc["ok"] = true
		}
	} else if stepErr != nil {
		rep.Category = cat
		rep.Message = stepErr.Error()
		if cat == "tool_error" || cat == "assertion" {
			rep.Status = statusFail
		} else {
			rep.Status = statusError
		}
	}

	if doc == nil {
		doc = map[string]any{"ok": false, "result": map[string]any{}}
	}
	doc["captured"] = captureMap(b)

	for _, cap := range st.Capture {
		if rep.Status == statusError {
			break
		}
		matches, err := assert.Select(doc, cap.Path)
		if err != nil || len(matches) == 0 {
			rep.Status = statusError
			rep.Category = "incomplete"
			rep.Message = fmt.Sprintf("capture %s: missing %s", cap.Name, cap.Path)
			break
		}
		val := matches[0]
		if len(matches) > 1 {
			val = matches
		}
		b.capture[cap.Name] = val
	}
	doc["captured"] = captureMap(b)

	actx := assert.Context{
		Doc:      doc,
		Bindings: captureMap(b),
		Duration: dur,
		Category: cat,
		MIMEType: mime,
		WorkDir:  workDir,
	}
	for k, v := range b.vars {
		actx.Bindings[k] = v
	}
	unavailable := false
	failed := false
	for _, a := range st.Assert {
		ia, interpErr := b.interpAssertion(a)
		if interpErr != nil {
			rep.Assertions = append(rep.Assertions, assert.Result{
				Kind: assertionKind(a), Status: assert.StatusSkip, Message: interpErr.Error(),
			})
			unavailable = true
			continue
		}
		r := assert.Eval(ia, actx)
		rep.Assertions = append(rep.Assertions, r)
		switch r.Status {
		case assert.StatusFail:
			failed = true
		case assert.StatusSkip:
			unavailable = true
		}
	}
	if unavailable && rep.Status == statusPass {
		rep.Status = statusFail
		rep.Category = "incomplete"
		rep.Message = "assertion could not be evaluated"
	} else if failed && (rep.Status == statusPass || st.ExpectFailure != nil) {
		rep.Status = statusFail
		if rep.Category == "" {
			rep.Category = "assertion"
		}
		if rep.Message == "" {
			rep.Message = "assertion failed"
		}
	}
	return rep
}

func doInspect(ctx context.Context, sess port.Session, target port.Target, op schema.InspectOp) (map[string]any, error) {
	kinds, err := inspectKinds(op.Kind)
	if err != nil {
		return nil, err
	}
	res, err := inspect.OnSession(ctx, sess, inspect.Request{Target: target, Kinds: kinds})
	return inspectDoc(res), err
}

func doCall(ctx context.Context, sess port.Session, target port.Target, op schema.CallOp, b *bindings) (map[string]any, string, error) {
	raw, err := b.interpRaw(op.Arguments)
	if err != nil {
		return nil, "", err
	}
	res, err := invoke.OnSession(ctx, sess, invoke.Request{
		Kind: invoke.KindCall, Name: op.Tool, ArgsJSON: string(raw), Target: target,
	})
	doc, mime := callDoc(res)
	return doc, mime, err
}

func doPrompt(ctx context.Context, sess port.Session, target port.Target, op schema.PromptOp, b *bindings) (map[string]any, string, error) {
	args, err := b.interpMap(op.Arguments)
	if err != nil {
		return nil, "", err
	}
	body, err := jsonv2Marshal(args)
	if err != nil {
		return nil, "", err
	}
	res, err := invoke.OnSession(ctx, sess, invoke.Request{
		Kind: invoke.KindPrompt, Name: op.Name, ArgsJSON: string(body), Target: target,
	})
	doc, mime := callDoc(res)
	return doc, mime, err
}

func doRead(ctx context.Context, sess port.Session, target port.Target, op schema.ReadOp, b *bindings) (map[string]any, string, error) {
	args, err := b.interpMap(op.Arguments)
	if err != nil {
		return nil, "", err
	}
	uri, err := b.interpString(op.URI)
	if err != nil {
		return nil, "", err
	}
	tmpl, err := b.interpString(op.Template)
	if err != nil {
		return nil, "", err
	}
	body, err := jsonv2Marshal(args)
	if err != nil {
		return nil, "", err
	}
	res, err := invoke.OnSession(ctx, sess, invoke.Request{
		Kind: invoke.KindRead, URI: uri, Template: tmpl, ArgsJSON: string(body), Target: target,
	})
	doc, mime := callDoc(res)
	return doc, mime, err
}

func doSnapshot(ctx context.Context, deps Deps, sess port.Session, target port.Target, op schema.SnapshotOp, workDir string) (map[string]any, error) {
	if deps.Store == nil || deps.Differ == nil {
		return nil, errclass.Internal{Err: errors.New("snapshot step needs store and differ")}
	}
	path := op.Compare
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	from, err := deps.Store.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	to, err := deps.Store.Capture(ctx, sess, nil)
	if err != nil {
		return nil, err
	}
	renames, err := diffapp.ParseRenames(op.Rename)
	if err != nil {
		return nil, err
	}
	failOn, err := diffapp.ParseFailOn(op.FailOn)
	if err != nil {
		return nil, err
	}
	raw, err := diff.Engine{Renames: renames}.Diff(ctx, from, to)
	if err != nil {
		return nil, err
	}
	classified, err := diffapp.Classify(raw, op.Ignore, failOn, false, true, diffapp.Result{})
	doc := map[string]any{
		"ok":     err == nil,
		"result": map[string]any{"outcome": classified.Outcome, "changed": classified.Diff.Changed},
	}
	return doc, err
}

func inspectKinds(kind string) ([]model.PrimitiveKind, error) {
	switch kind {
	case "", "all":
		return nil, nil
	case string(model.KindTools), string(model.KindPrompts), string(model.KindResources), string(model.KindTemplates):
		return []model.PrimitiveKind{model.PrimitiveKind(kind)}, nil
	default:
		return nil, errclass.Usage{Msg: "unknown inspect kind " + kind}
	}
}

func stepOp(st schema.Step) string {
	switch {
	case st.Inspect != nil:
		return "inspect"
	case st.Call != nil:
		return "call"
	case st.Prompt != nil:
		return "prompt"
	case st.Read != nil:
		return "read"
	case st.Snapshot != nil:
		return "snapshot"
	default:
		return "unknown"
	}
}

func assertionKind(a schema.Assertion) string {
	switch {
	case a.Equal != nil:
		return "equal"
	case a.Contains != nil:
		return "contains"
	case a.JSONPath != nil:
		return "json_path"
	case a.Schema != nil:
		return "schema"
	case a.ContentType != nil:
		return "content_type"
	case a.Latency != nil:
		return "latency"
	case a.ErrorCategory != nil:
		return "error_category"
	default:
		return "unknown"
	}
}

func captureMap(b *bindings) map[string]any {
	out := make(map[string]any, len(b.capture)+len(b.vars))
	for k, v := range b.vars {
		out[k] = v
	}
	maps.Copy(out, b.capture)
	return out
}

func jsonv2Marshal(m map[string]string) ([]byte, error) {
	if m == nil {
		m = map[string]string{}
	}
	b, err := json.Marshal(m, json.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: err}
	}
	return b, nil
}
