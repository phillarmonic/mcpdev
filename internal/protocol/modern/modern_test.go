package modern

import (
	"net/http"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestSelectHighestMutual(t *testing.T) {
	got := Select(PreferredVersions, []string{"2025-11-25", Era})
	if got != Era {
		t.Fatalf("got %q", got)
	}
	if Select(PreferredVersions, []string{"2025-11-25"}) != "" {
		t.Fatal("legacy-only list must not select a modern version")
	}
}

func TestRequestMetaKeys(t *testing.T) {
	meta := RequestMeta(model.Identity{Name: "mcpdev-test", Version: "dev", Title: "Lab"}, Era)
	if meta[MetaProtocolVersion] != Era {
		t.Fatalf("version: %v", meta[MetaProtocolVersion])
	}
	info, _ := meta[MetaClientInfo].(map[string]any)
	if info["name"] != "mcpdev-test" || info["title"] != "Lab" {
		t.Fatalf("clientInfo: %v", info)
	}
	if _, ok := meta[MetaClientCapabilities].(map[string]any); !ok {
		t.Fatalf("clientCapabilities: %T", meta[MetaClientCapabilities])
	}
}

func TestClassifyRPCRecognizedStayModern(t *testing.T) {
	miss := ClassifyRPC(CodeMissingCapability, nil)
	if !miss.StayModern() || miss.Kind != KindMissingCapability {
		t.Fatalf("missing cap: %+v", miss)
	}
	header := ClassifyRPC(CodeHeaderMismatch, nil)
	if !header.StayModern() || header.Kind != KindHeaderMismatch {
		t.Fatalf("header: %+v", header)
	}
	unsup := ClassifyRPC(CodeUnsupportedVersion, map[string]any{
		"supported": []any{Era, "2025-11-25"},
		"requested": "1999-01-01",
	})
	if !unsup.StayModern() || Select(PreferredVersions, unsup.Supported) != Era {
		t.Fatalf("unsupported retry: %+v", unsup)
	}
	if ClassifyRPC(-32601, nil).StayModern() {
		t.Fatal("method not found must allow fallback")
	}
}

func TestClassifyHTTPEmptyAndUnrecognizedFallback(t *testing.T) {
	empty := ClassifyHTTP(http.StatusBadRequest, nil)
	if empty.StayModern() || empty.Kind != KindEmptyBody {
		t.Fatalf("empty: %+v", empty)
	}
	unrec := ClassifyHTTP(http.StatusBadRequest, []byte(`{"jsonrpc":"2.0","error":{"code":-1,"message":"nope"}}`))
	if unrec.StayModern() || unrec.Kind != KindUnrecognized {
		t.Fatalf("unrecognized: %+v", unrec)
	}
	mismatch := ClassifyHTTP(http.StatusBadRequest, []byte(`{"jsonrpc":"2.0","error":{"code":-32020,"message":"HeaderMismatch"}}`))
	if !mismatch.StayModern() {
		t.Fatalf("header mismatch must stay modern: %+v", mismatch)
	}
}
