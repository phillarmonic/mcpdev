package diagnostic

import (
	"context"
	"errors"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// Severity is a stable diagnostic level.
type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

// Code is a stable machine token. New codes extend this list; they do not reuse names.
type Code string

const (
	CodeUsage           Code = "usage_error"
	CodeConnection      Code = "connection_failed"
	CodeProtocol        Code = "protocol_error"
	CodeAssertion       Code = "assertion_failed"
	CodeInterfaceChange Code = "interface_changed"
	CodeIncomplete      Code = "incomplete"
	CodeInternal        Code = "internal_error"
	CodeChildStderr     Code = "child_stderr"
)

// DefaultPreviewLimit is the maximum byte length of string previews in renderers.
const DefaultPreviewLimit = 512

// Record is the shared diagnostic used by one-shot commands, watch, traces, and scenario reports.
type Record struct {
	Cause        error
	Details      map[string]any
	Code         Code
	Severity     Severity
	Message      string
	Operation    string
	Transport    string
	Era          model.ProtocolEra
	ChildStderr  string
	Elapsed      time.Duration
	PreviewLimit int
	Retryable    bool
}

func (r Record) Error() string {
	if r.Message != "" {
		if r.Code != "" {
			return string(r.Code) + ": " + r.Message
		}
		return r.Message
	}
	if r.Cause != nil {
		return r.Cause.Error()
	}
	if r.Code != "" {
		return string(r.Code)
	}
	return "diagnostic"
}

func (r Record) Unwrap() error { return r.Cause }

func (r Record) ExitClass() int {
	switch r.Code {
	case CodeUsage:
		return errclass.ExitUsage
	case CodeConnection:
		return errclass.ExitConnection
	case CodeProtocol:
		return errclass.ExitProtocol
	case CodeAssertion:
		return errclass.ExitAssertion
	case CodeInterfaceChange:
		return errclass.ExitInterfaceChange
	case CodeIncomplete:
		return errclass.ExitIncomplete
	case CodeChildStderr:
		return errclass.ExitOK
	case CodeInternal:
		return errclass.ExitInternal
	default:
		if r.Cause != nil {
			return errclass.ExitCode(r.Cause)
		}
		return errclass.ExitInternal
	}
}

// ProtocolFailure reports a protocol-class diagnostic. Child stderr never sets this by itself.
func (r Record) ProtocolFailure() bool {
	return r.Code == CodeProtocol
}

// FromError fills Code, Severity, and Message from err when those fields are empty.
func FromError(err error, rec Record) Record {
	if err == nil {
		return rec
	}
	if existing, ok := errors.AsType[Record](err); ok {
		if rec.Operation == "" {
			rec.Operation = existing.Operation
		}
		if rec.Transport == "" {
			rec.Transport = existing.Transport
		}
		if rec.Era == "" {
			rec.Era = existing.Era
		}
		if rec.ChildStderr == "" {
			rec.ChildStderr = existing.ChildStderr
		}
	}
	rec.Cause = err
	if rec.Code == "" {
		rec.Code = codeFromExit(errclass.ExitCode(err))
	}
	if rec.Severity == "" {
		rec.Severity = SeverityError
		if rec.Code == CodeChildStderr {
			rec.Severity = SeverityInfo
		}
	}
	if rec.Message == "" {
		rec.Message = err.Error()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		rec.Retryable = true
	}
	return rec
}

func codeFromExit(code int) Code {
	switch code {
	case errclass.ExitUsage:
		return CodeUsage
	case errclass.ExitConnection:
		return CodeConnection
	case errclass.ExitProtocol:
		return CodeProtocol
	case errclass.ExitAssertion:
		return CodeAssertion
	case errclass.ExitInterfaceChange:
		return CodeInterfaceChange
	case errclass.ExitIncomplete:
		return CodeIncomplete
	default:
		return CodeInternal
	}
}
