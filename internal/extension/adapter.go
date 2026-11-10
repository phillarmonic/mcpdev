package extension

import (
	"context"
	"io"
)

// Config is operator input for a custom channel. Values are untrusted except
// when the host process supplied them (headers, endpoint).
type Config struct {
	Headers       map[string]string
	Extra         map[string]string
	Endpoint      string
	AllowInsecure bool
}

// Conn is a live custom channel. Close owns teardown. Diagnostics must already
// be redacted: never echo Authorization, tokens, or raw header values.
type Conn interface {
	Reader() io.ReadCloser
	Writer() io.WriteCloser
	Close(ctx context.Context) error
	Diagnostics() map[string]string
}

// Adapter constructs a custom JSON-RPC channel. Implementations live outside
// internal/protocol and must not import it.
type Adapter interface {
	Name() string
	Semantics() Semantics
	Open(ctx context.Context, cfg Config) (Conn, error)
}
