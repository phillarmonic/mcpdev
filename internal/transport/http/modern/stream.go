package modern

import (
	"context"
	"io"

	"github.com/phillarmonic/mcpdev/internal/transport/ndjsonhttp"
)

// ByteStream presents POST JSON-RPC as a newline-delimited pipe for protocol.Connect.
func (c *Conn) ByteStream(ctx context.Context) (io.ReadCloser, io.WriteCloser) {
	return ndjsonhttp.Bind(ctx, func(ctx context.Context, payload []byte) ([]byte, error) {
		ex, err := c.Post(ctx, payload)
		if err != nil {
			return nil, err
		}
		return ex.Body, nil
	})
}
