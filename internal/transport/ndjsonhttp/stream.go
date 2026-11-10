package ndjsonhttp

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"slices"
)

// Poster sends one JSON-RPC message and returns the response body (possibly empty).
type Poster func(ctx context.Context, payload []byte) ([]byte, error)

// Bind pumps newline-delimited JSON-RPC over POST.
// The returned writer is the client-to-server side; the reader is server-to-client.
func Bind(ctx context.Context, post Poster) (io.ReadCloser, io.WriteCloser) {
	respR, respW := io.Pipe()
	reqR, reqW := io.Pipe()
	go run(ctx, post, reqR, respW)
	return respR, reqW
}

func run(ctx context.Context, post Poster, req io.ReadCloser, resp *io.PipeWriter) {
	defer func() { _ = req.Close() }()
	sc := bufio.NewScanner(req)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			_ = resp.CloseWithError(err)
			return
		}
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		body, err := post(ctx, slices.Clone(line))
		if err != nil {
			_ = resp.CloseWithError(err)
			return
		}
		if len(body) == 0 {
			continue
		}
		if _, err := resp.Write(body); err != nil {
			return
		}
		if body[len(body)-1] != '\n' {
			if _, err := resp.Write([]byte{'\n'}); err != nil {
				return
			}
		}
	}
	if err := sc.Err(); err != nil {
		_ = resp.CloseWithError(err)
		return
	}
	_ = resp.Close()
}
