package stdio

import (
	"bytes"
	"errors"
	"io"
	"os"
	"unicode"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// firstJSON guards stdout so a non-JSON first byte is a protocol error, not a
// mixed diagnostic stream. Child stderr is never inspected here.
type firstJSON struct {
	r       io.ReadCloser
	err     error
	buf     bytes.Buffer
	checked bool
}

func (r *firstJSON) Read(p []byte) (int, error) {
	if !r.checked {
		if err := r.peek(); err != nil {
			return 0, err
		}
	}
	if r.buf.Len() > 0 {
		n, _ := r.buf.Read(p)
		return n, nil
	}
	return r.r.Read(p)
}

func (r *firstJSON) peek() error {
	r.checked = true
	tmp := make([]byte, 1)
	for {
		n, err := r.r.Read(tmp)
		if n == 1 {
			c := rune(tmp[0])
			if unicode.IsSpace(c) {
				r.buf.WriteByte(tmp[0])
				continue
			}
			r.buf.WriteByte(tmp[0])
			if tmp[0] != '{' && tmp[0] != '[' {
				r.err = errclass.Protocol{Err: errors.New("stdio: first stdout byte is not JSON")}
				return r.err
			}
			return nil
		}
		if err != nil {
			// cmd.Wait closes a StdoutPipe as soon as the child exits, so an
			// early exit can surface as a closed pipe instead of EOF. Both mean
			// the child stopped before writing JSON-RPC, which is a connection
			// failure rather than an internal one.
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
				r.err = errclass.Connection{Err: errors.New("stdio: child exited before JSON-RPC stdout")}
				return r.err
			}
			r.err = err
			return err
		}
	}
}

func (r *firstJSON) Close() error {
	return r.r.Close()
}
