package trace

import "time"

// Kind is a stable trace event token.
type Kind string

const (
	KindFrame    Kind = "frame"
	KindHTTPReq  Kind = "http_request"
	KindHTTPResp Kind = "http_response"
	KindSSE      Kind = "sse"
	KindRetry    Kind = "retry"
	KindFallback Kind = "fallback"
	KindCancel   Kind = "cancel"
	KindStderr   Kind = "stderr"
)

// Direction is who sent the bytes. Correlation events use DirInternal.
type Direction string

const (
	DirClient   Direction = "client"
	DirServer   Direction = "server"
	DirInternal Direction = "internal"
)

// Format selects stderr rendering.
type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
)

// DefaultPreviewLimit matches diagnostic previews.
const DefaultPreviewLimit = 512

// unlimitedPreview keeps the full redacted payload (--verbose).
const unlimitedPreview = -1

// Event is one ordered, redacted exchange or correlation record.
type Event struct {
	Headers      map[string]string
	Kind         Kind
	Direction    Direction
	Transport    string
	Method       string
	RequestID    string
	Note         string
	Payload      string
	Seq          int
	HTTPStatus   int
	Attempt      int
	PayloadBytes int
	Elapsed      time.Duration
	Truncated    bool
}
