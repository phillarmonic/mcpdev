package legacy

import (
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func TestDecodeCapsObject(t *testing.T) {
	caps, err := DecodeCaps(map[string]any{
		"tools":        map[string]any{"listChanged": true},
		"prompts":      map[string]any{},
		"resources":    map[string]any{"subscribe": true},
		"experimental": map[string]any{"vendor.flag": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !caps.Tools || !caps.Prompts || !caps.Resources {
		t.Fatalf("presence: %+v", caps)
	}
	if !caps.ToolsListChanged || !caps.ResourceSubscribe {
		t.Fatalf("flags: %+v", caps)
	}
	if len(caps.Experimental) != 1 || caps.Experimental[0] != "vendor.flag" {
		t.Fatalf("experimental: %v", caps.Experimental)
	}
}

func TestDecodeCapsMalformed(t *testing.T) {
	cases := []any{
		[]any{"tools"},
		"tools",
		map[string]any{"tools": true},
		map[string]any{"tools": map[string]any{"listChanged": "yes"}},
		map[string]any{"experimental": []any{}},
	}
	for _, v := range cases {
		_, err := DecodeCaps(v)
		if errclass.ExitCode(err) != errclass.ExitProtocol {
			t.Fatalf("%v: class %d err=%v", v, errclass.ExitCode(err), err)
		}
	}
}

func TestDecodeCapsNilIsEmpty(t *testing.T) {
	caps, err := DecodeCaps(nil)
	if err != nil {
		t.Fatal(err)
	}
	if caps.Tools {
		t.Fatalf("empty: %+v", caps)
	}
}
