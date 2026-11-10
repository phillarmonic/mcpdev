package assert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

// Context is the step document plus bindings used by variable equals.
type Context struct {
	Doc      any
	Bindings map[string]any
	Category string
	MIMEType string
	WorkDir  string
	Duration time.Duration
}

// Eval runs one closed assertion. Missing paths are skip (unevaluable), never pass.
func Eval(a schema.Assertion, ctx Context) Result {
	switch {
	case a.Equal != nil:
		return evalEqual(*a.Equal, ctx)
	case a.Contains != nil:
		return evalContains(*a.Contains, ctx)
	case a.JSONPath != nil:
		return evalJSONPath(*a.JSONPath, ctx)
	case a.Schema != nil:
		return evalSchema(*a.Schema, ctx)
	case a.ContentType != nil:
		return evalContentType(*a.ContentType, ctx)
	case a.Latency != nil:
		return evalLatency(*a.Latency, ctx)
	case a.ErrorCategory != nil:
		return evalErrorCategory(*a.ErrorCategory, ctx)
	default:
		return Result{Kind: "unknown", Status: StatusSkip, Message: "unavailable assertion kind"}
	}
}

func evalEqual(a schema.EqualAssert, ctx Context) Result {
	r := Result{Kind: "equal", Path: a.Path}
	actual, err := one(ctx.Doc, a.Path)
	if err != nil {
		r.Status = StatusSkip
		r.Message = err.Error()
		return r
	}
	r.Actual = actual
	expected := a.Value
	if a.Variable != "" {
		v, ok := ctx.Bindings[a.Variable]
		if !ok {
			r.Status = StatusSkip
			r.Message = "unknown variable " + a.Variable
			return r
		}
		expected = v
	}
	r.Expected = expected
	if equalJSON(actual, expected) {
		r.Status = StatusPass
		return r
	}
	r.Status = StatusFail
	r.Diff = fmt.Sprintf("expected %s, actual %s", compact(expected), compact(actual))
	return r
}

func evalContains(a schema.ContainsAssert, ctx Context) Result {
	r := Result{Kind: "contains", Path: a.Path, Expected: a.Value}
	actual, err := one(ctx.Doc, a.Path)
	if err != nil {
		matches, selErr := Select(ctx.Doc, a.Path)
		if selErr != nil || len(matches) == 0 {
			r.Status = StatusSkip
			r.Message = err.Error()
			return r
		}
		actual = matches
	}
	r.Actual = actual
	if containsValue(actual, a.Value) {
		r.Status = StatusPass
		return r
	}
	r.Status = StatusFail
	r.Diff = fmt.Sprintf("%s does not contain %s", compact(actual), compact(a.Value))
	return r
}

func evalJSONPath(a schema.JSONPathAssert, ctx Context) Result {
	r := Result{Kind: "json_path", Path: a.Path}
	matches, err := Select(ctx.Doc, a.Path)
	if err != nil {
		r.Status = StatusSkip
		r.Message = err.Error()
		return r
	}
	r.Actual = len(matches)
	if a.Exists != nil {
		r.Expected = *a.Exists
		got := len(matches) > 0
		if got != *a.Exists {
			r.Status = StatusFail
			r.Diff = fmt.Sprintf("exists=%v count=%d", got, len(matches))
			return r
		}
	}
	if a.Count != nil {
		r.Expected = *a.Count
		if len(matches) != *a.Count {
			r.Status = StatusFail
			r.Diff = fmt.Sprintf("count expected %d actual %d", *a.Count, len(matches))
			return r
		}
	}
	r.Status = StatusPass
	return r
}

func evalSchema(a schema.SchemaAssert, ctx Context) Result {
	r := Result{Kind: "schema", Path: a.Path}
	body := a.Schema
	if a.File != "" {
		p := a.File
		if ctx.WorkDir != "" && !filepath.IsAbs(p) {
			p = filepath.Join(ctx.WorkDir, a.File)
		}
		p = filepath.Clean(p)
		raw, err := os.ReadFile(p)
		if err != nil {
			r.Status = StatusSkip
			r.Message = "schema file: " + err.Error()
			return r
		}
		body = raw
	}
	if len(body) == 0 {
		r.Status = StatusSkip
		r.Message = "schema is empty"
		return r
	}
	var sch any
	if err := json.Unmarshal(body, &sch); err != nil {
		r.Status = StatusSkip
		r.Message = "schema JSON: " + err.Error()
		return r
	}
	actual, err := one(ctx.Doc, a.Path)
	if err != nil {
		r.Status = StatusSkip
		r.Message = err.Error()
		return r
	}
	r.Actual = actual
	r.Expected = sch
	if msg := matchSchema(actual, sch); msg != "" {
		r.Status = StatusFail
		r.Diff = msg
		return r
	}
	r.Status = StatusPass
	return r
}

func evalContentType(a schema.ContentTypeAssert, ctx Context) Result {
	r := Result{Kind: "content_type", Expected: a.Equals, Actual: ctx.MIMEType}
	if ctx.MIMEType == "" {
		r.Status = StatusSkip
		r.Message = "no content type on this step result"
		return r
	}
	if ctx.MIMEType == a.Equals {
		r.Status = StatusPass
		return r
	}
	r.Status = StatusFail
	r.Diff = fmt.Sprintf("expected %s actual %s", a.Equals, ctx.MIMEType)
	return r
}

func evalLatency(a schema.LatencyAssert, ctx Context) Result {
	r := Result{Kind: "latency", Expected: a.Max, Actual: ctx.Duration.String()}
	limit, err := time.ParseDuration(a.Max)
	if err != nil {
		r.Status = StatusSkip
		r.Message = "invalid latency max"
		return r
	}
	if ctx.Duration <= limit {
		r.Status = StatusPass
		return r
	}
	r.Status = StatusFail
	r.Diff = fmt.Sprintf("%s exceeded %s", ctx.Duration, limit)
	return r
}

func evalErrorCategory(a schema.ErrorCategoryAssert, ctx Context) Result {
	r := Result{Kind: "error_category", Expected: a.Equals, Actual: ctx.Category}
	if a.Equals == ctx.Category {
		r.Status = StatusPass
		return r
	}
	r.Status = StatusFail
	r.Diff = fmt.Sprintf("expected %s actual %s", a.Equals, ctx.Category)
	return r
}

func one(doc any, path string) (any, error) {
	if strings.TrimSpace(path) == "" {
		return doc, nil
	}
	matches, err := Select(doc, path)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("missing %s", path)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return matches, nil
}

func containsValue(hay any, needle any) bool {
	switch t := hay.(type) {
	case string:
		return strings.Contains(t, fmt.Sprint(needle))
	case []any:
		for _, item := range t {
			if equalJSON(item, needle) || containsValue(item, needle) {
				return true
			}
			if s, ok := item.(string); ok && strings.Contains(s, fmt.Sprint(needle)) {
				return true
			}
		}
	default:
		if s, ok := hay.(string); ok {
			return strings.Contains(s, fmt.Sprint(needle))
		}
	}
	return false
}
