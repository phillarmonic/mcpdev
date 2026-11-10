package stdio

import (
	"io"
	"os"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }
func (r errorReader) Close() error             { return nil }

type bytesThenError struct {
	err  error
	data []byte
}

func (r *bytesThenError) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func (r *bytesThenError) Close() error { return nil }

// cmd.Wait closes a StdoutPipe as soon as the child exits, so an early exit can
// arrive as a closed pipe instead of EOF. Both are connection failures.
func TestFirstJSONClosedPipeIsConnection(t *testing.T) {
	names := []string{"eof", "file closed", "pipe closed"}
	errors := []error{io.EOF, os.ErrClosed, io.ErrClosedPipe}
	for i, readErr := range errors {
		t.Run(names[i], func(t *testing.T) {
			conn := &firstJSON{r: errorReader{err: readErr}}
			_, err := conn.Read(make([]byte, 8))
			if got := errclass.ExitCode(err); got != errclass.ExitConnection {
				t.Fatalf("exit %d, want %d: %v", got, errclass.ExitConnection, err)
			}
		})
	}
}

func TestFirstJSONWhitespaceThenCloseIsConnection(t *testing.T) {
	conn := &firstJSON{r: &bytesThenError{data: []byte("  \n"), err: os.ErrClosed}}
	_, err := conn.Read(make([]byte, 8))
	if got := errclass.ExitCode(err); got != errclass.ExitConnection {
		t.Fatalf("exit %d, want %d: %v", got, errclass.ExitConnection, err)
	}
}
