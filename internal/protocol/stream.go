package protocol

import "io"

// Stream is a newline-delimited JSON-RPC byte stream.
// Reader is server-to-client (stdout). Writer is client-to-server (stdin).
// Callers never pass SDK transport types.
type Stream struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}
