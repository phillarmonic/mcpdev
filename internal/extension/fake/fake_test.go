package fake

import (
	"testing"

	"github.com/phillarmonic/mcpdev/internal/extension"
)

func TestOpenAndClose(t *testing.T) {
	c := Connector{
		Sem: extension.Semantics{
			Cancellation: extension.CancelNone,
			Streaming:    extension.StreamNewlineJSON,
			Metadata:     extension.MetaCustom,
			Credentials:  extension.CredNone,
		},
	}
	conn, err := c.Open(t.Context(), extension.Config{Endpoint: "mem"})
	if err != nil {
		t.Fatal(err)
	}
	if conn.Diagnostics()["closed"] != "false" {
		t.Fatalf("%v", conn.Diagnostics())
	}
	if err := conn.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if conn.Diagnostics()["closed"] != "true" {
		t.Fatal("expected closed")
	}
}
