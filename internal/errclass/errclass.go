// Package errclass maps typed operational errors to CLI exit classes.
package errclass

import "errors"

const (
	ExitOK              = 0
	ExitUsage           = 2
	ExitConnection      = 10
	ExitProtocol        = 20
	ExitAssertion       = 30
	ExitInterfaceChange = 40
	ExitIncomplete      = 60
	ExitInternal        = 70
)

type coder interface {
	ExitClass() int
}

// ExitCode returns the CLI class for err. Unknown errors are internal (70).
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var c coder
	if errors.As(err, &c) {
		return c.ExitClass()
	}
	return ExitInternal
}

// Category is the scenario expect_failure / error_category token for err.
func Category(err error) string {
	if err == nil {
		return ""
	}
	if _, ok := errors.AsType[ToolError](err); ok {
		return "tool_error"
	}
	if _, ok := errors.AsType[Cleanup](err); ok {
		return "cleanup"
	}
	switch ExitCode(err) {
	case ExitUsage:
		return "usage"
	case ExitConnection:
		return "connection"
	case ExitProtocol:
		return "protocol"
	case ExitAssertion:
		return "assertion"
	case ExitInterfaceChange:
		return "interface_change"
	case ExitIncomplete:
		return "incomplete"
	default:
		return "internal"
	}
}

// Usage is a caller or flag error (exit 2).
type Usage struct {
	Err error
	Msg string
}

func (e Usage) Error() string {
	if e.Msg != "" && e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "usage error"
}

func (e Usage) Unwrap() error  { return e.Err }
func (e Usage) ExitClass() int { return ExitUsage }

// Connection is a transport or handshake I/O failure (exit 10).
type Connection struct {
	Err error
}

func (e Connection) Error() string {
	if e.Err == nil {
		return "connection error"
	}
	return e.Err.Error()
}

func (e Connection) Unwrap() error  { return e.Err }
func (e Connection) ExitClass() int { return ExitConnection }

// Protocol is a JSON-RPC or era mismatch (exit 20).
type Protocol struct {
	Err error
}

func (e Protocol) Error() string {
	if e.Err == nil {
		return "protocol error"
	}
	return e.Err.Error()
}

func (e Protocol) Unwrap() error  { return e.Err }
func (e Protocol) ExitClass() int { return ExitProtocol }

// Assertion is a failed scenario or call --fail-on-tool-error (exit 30).
type Assertion struct {
	Err error
}

func (e Assertion) Error() string {
	if e.Err == nil {
		return "assertion failed"
	}
	return e.Err.Error()
}

func (e Assertion) Unwrap() error  { return e.Err }
func (e Assertion) ExitClass() int { return ExitAssertion }

// InterfaceChange is a public-interface difference at or above the fail threshold (exit 40).
type InterfaceChange struct {
	Err error
}

func (e InterfaceChange) Error() string {
	if e.Err == nil {
		return "interface changed"
	}
	return e.Err.Error()
}

func (e InterfaceChange) Unwrap() error  { return e.Err }
func (e InterfaceChange) ExitClass() int { return ExitInterfaceChange }

// Incomplete is unfinished collection (exit 60).
type Incomplete struct {
	Err error
}

func (e Incomplete) Error() string {
	if e.Err == nil {
		return "incomplete result"
	}
	return e.Err.Error()
}

func (e Incomplete) Unwrap() error  { return e.Err }
func (e Incomplete) ExitClass() int { return ExitIncomplete }

// Internal is an invariant or unexpected failure (exit 70).
type Internal struct {
	Err error
}

func (e Internal) Error() string {
	if e.Err == nil {
		return "internal error"
	}
	return e.Err.Error()
}

func (e Internal) Unwrap() error  { return e.Err }
func (e Internal) ExitClass() int { return ExitInternal }

// ToolError is a successful tools/call whose payload set isError.
// Scenario expect_failure uses category tool_error; CLI --fail-on-tool-error maps it to assertion.
type ToolError struct {
	Name string
}

func (e ToolError) Error() string {
	if e.Name == "" {
		return "tool returned isError"
	}
	return "tool " + e.Name + " returned isError"
}

func (e ToolError) ExitClass() int { return ExitAssertion }

// Cleanup is a failed always-run hook or session Close after the case finished.
type Cleanup struct {
	Err error
}

func (e Cleanup) Error() string {
	if e.Err == nil {
		return "cleanup failed"
	}
	return e.Err.Error()
}

func (e Cleanup) Unwrap() error  { return e.Err }
func (e Cleanup) ExitClass() int { return ExitInternal }
