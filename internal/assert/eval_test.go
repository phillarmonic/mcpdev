package assert

import (
	"testing"
	"time"

	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

func TestEqualAndContains(t *testing.T) {
	doc := map[string]any{
		"ok": true,
		"result": map[string]any{
			"tools": []any{
				map[string]any{"name": "list_files"},
				map[string]any{"name": "read_file"},
			},
			"text": "module github.com/example",
		},
	}
	ctx := Context{Doc: doc, Bindings: map[string]any{"first_tool": "list_files"}, Duration: time.Millisecond}
	pass := Eval(schema.Assertion{Equal: &schema.EqualAssert{Path: "$.ok", Value: true}}, ctx)
	if pass.Status != StatusPass {
		t.Fatalf("%+v", pass)
	}
	names := Eval(schema.Assertion{Contains: &schema.ContainsAssert{Path: "$.result.tools[*].name", Value: "list_files"}}, ctx)
	if names.Status != StatusPass {
		t.Fatalf("%+v", names)
	}
	miss := Eval(schema.Assertion{Equal: &schema.EqualAssert{Path: "$.missing", Value: 1}}, ctx)
	if miss.Status != StatusSkip {
		t.Fatalf("missing path must skip: %+v", miss)
	}
	bind := Eval(schema.Assertion{Equal: &schema.EqualAssert{Path: "$.result.tools[0].name", Variable: "first_tool"}}, ctx)
	if bind.Status != StatusPass {
		t.Fatalf("%+v", bind)
	}
}

func TestJSONPathCountAndSchema(t *testing.T) {
	doc := map[string]any{"result": map[string]any{"messages": []any{map[string]any{"role": "user"}}}}
	one := 1
	exists := true
	r := Eval(schema.Assertion{JSONPath: &schema.JSONPathAssert{Path: "$.result.messages", Exists: &exists, Count: &one}}, Context{Doc: doc})
	if r.Status != StatusPass {
		t.Fatalf("%+v", r)
	}
	sch := Eval(schema.Assertion{Schema: &schema.SchemaAssert{Path: "$.result", Schema: []byte(`{"type":"object"}`)}}, Context{Doc: doc})
	if sch.Status != StatusPass {
		t.Fatalf("%+v", sch)
	}
}

func TestLatencyAndErrorCategory(t *testing.T) {
	ctx := Context{Doc: map[string]any{}, Duration: 2 * time.Second, Category: "tool_error", MIMEType: "text/markdown"}
	lat := Eval(schema.Assertion{Latency: &schema.LatencyAssert{Max: "1s"}}, ctx)
	if lat.Status != StatusFail {
		t.Fatalf("%+v", lat)
	}
	cat := Eval(schema.Assertion{ErrorCategory: &schema.ErrorCategoryAssert{Equals: "tool_error"}}, ctx)
	if cat.Status != StatusPass {
		t.Fatalf("%+v", cat)
	}
	mime := Eval(schema.Assertion{ContentType: &schema.ContentTypeAssert{Equals: "text/markdown"}}, ctx)
	if mime.Status != StatusPass {
		t.Fatalf("%+v", mime)
	}
}
