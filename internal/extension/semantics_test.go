package extension

import "testing"

func TestSemanticsValidate(t *testing.T) {
	ok := Semantics{
		Cancellation: CancelRequestContext,
		Streaming:    StreamCustom,
		Metadata:     MetaCustom,
		Credentials:  CredNone,
	}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	var empty Semantics
	if err := empty.Validate(); err == nil {
		t.Fatal("expected missing semantics to fail")
	}
}

func TestExperimentalGRPCUnspecified(t *testing.T) {
	var g ExperimentalGRPC
	if g.Ready() {
		t.Fatal("empty mapping must not be ready")
	}
	g.Service = "example.v1.Unknown"
	g.Framing = "bytes-as-jsonrpc"
	if !g.Ready() {
		t.Fatal("expected ready after both fields")
	}
}
