package errclass

import (
	"context"
	"errors"
	"testing"
)

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != ExitOK {
		t.Fatal("nil")
	}
	if ExitCode(Usage{Msg: "bad flag"}) != ExitUsage {
		t.Fatal("usage")
	}
	if ExitCode(Connection{Err: context.Canceled}) != ExitConnection {
		t.Fatal("connection")
	}
	if !errors.Is(Connection{Err: context.Canceled}, context.Canceled) {
		t.Fatal("unwrap canceled")
	}
	if ExitCode(errors.New("mystery")) != ExitInternal {
		t.Fatal("unknown")
	}
	if ExitCode(Assertion{Err: errors.New("isError")}) != ExitAssertion {
		t.Fatal("assertion")
	}
	if ExitCode(InterfaceChange{Err: errors.New("changed")}) != ExitInterfaceChange {
		t.Fatal("interface change")
	}
	if Category(ToolError{Name: "echo"}) != "tool_error" {
		t.Fatal("tool_error")
	}
	if Category(Cleanup{Err: errors.New("hook")}) != "cleanup" {
		t.Fatal("cleanup")
	}
}
