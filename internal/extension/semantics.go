package extension

import "github.com/phillarmonic/mcpdev/internal/errclass"

// Cancellation names how the custom channel aborts in-flight JSON-RPC.
type Cancellation string

const (
	// CancelSharedNotification posts notifications/cancelled on the shared stream (stdio-shaped).
	CancelSharedNotification Cancellation = "shared-notifications-cancelled"
	// CancelRequestContext cancels the per-request I/O context (HTTP-shaped).
	CancelRequestContext Cancellation = "request-context"
	// CancelNone means the channel cannot cancel in-flight work; callers must treat that as explicit.
	CancelNone Cancellation = "none"
)

// Streaming names how JSON-RPC messages move on the channel.
type Streaming string

const (
	// StreamNewlineJSON is newline-delimited JSON-RPC (stdio framing).
	StreamNewlineJSON Streaming = "newline-jsonrpc"
	// StreamRequestSSE is request-scoped SSE wrapping JSON-RPC bodies.
	StreamRequestSSE Streaming = "request-scoped-sse"
	// StreamCustom is a declared non-stdio, non-HTTP framing. The adapter docs must define it.
	StreamCustom Streaming = "custom"
)

// Metadata names how protocol version and client/server info are carried.
type Metadata string

const (
	// MetaInitialize uses the 2025-11-25 handshake fields.
	MetaInitialize Metadata = "initialize-handshake"
	// MetaPerRequest uses 2026-07-28 per-request _meta (and headers, if any).
	MetaPerRequest Metadata = "per-request-meta"
	// MetaCustom is a declared mapping. The adapter docs must define it.
	MetaCustom Metadata = "custom"
)

// Credentials names where secrets live and that they must be redacted in diagnostics.
type Credentials string

const (
	// CredNone means the channel carries no operator secrets.
	CredNone Credentials = "none"
	// CredHeaders means operator-supplied HTTP-style headers (redact Authorization and kin).
	CredHeaders Credentials = "headers"
	// CredEnv means child environment variables, never a shell string from the server.
	CredEnv Credentials = "env"
	// CredCustom is a declared secret channel. Diagnostics must still redact values.
	CredCustom Credentials = "custom"
)

// Semantics is the required capability declaration for a custom transport.
// Every field must be a known non-empty token. Incomplete declarations are rejected.
type Semantics struct {
	Cancellation Cancellation
	Streaming    Streaming
	Metadata     Metadata
	Credentials  Credentials
}

// Validate reports a usage error when any required semantic is missing or unknown.
func (s Semantics) Validate() error {
	switch s.Cancellation {
	case CancelSharedNotification, CancelRequestContext, CancelNone:
	default:
		return errclass.Usage{Msg: "extension: cancellation semantics are required"}
	}
	switch s.Streaming {
	case StreamNewlineJSON, StreamRequestSSE, StreamCustom:
	default:
		return errclass.Usage{Msg: "extension: streaming semantics are required"}
	}
	switch s.Metadata {
	case MetaInitialize, MetaPerRequest, MetaCustom:
	default:
		return errclass.Usage{Msg: "extension: metadata semantics are required"}
	}
	switch s.Credentials {
	case CredNone, CredHeaders, CredEnv, CredCustom:
	default:
		return errclass.Usage{Msg: "extension: credential semantics are required"}
	}
	return nil
}
