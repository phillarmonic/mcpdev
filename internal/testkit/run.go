package testkit

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// RunStdio serves a named catalog fixture on stdio until stdin EOF or error.
// Protocol JSON-RPC is written only to stdout. Logs go to stderr.
func RunStdio(stdin io.Reader, stdout io.Writer, stderr io.Writer, name string) error {
	d, err := Named(name)
	if err != nil {
		return err
	}
	d.Transport = TransportStdio
	err = runStdio(context.Background(), d, stdin, stdout, stderr)
	if errors.Is(err, io.EOF) {
		_, _ = fmt.Fprintf(stderr, "fixture: shutdown name=%s\n", d.Name)
		return nil
	}
	return err
}
