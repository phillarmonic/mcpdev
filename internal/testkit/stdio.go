package testkit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"sync"
)

// StdioLab is an in-process newline-delimited JSON-RPC server.
// ClientIn is the client's write end (server stdin). ClientOut is the
// client's read end (server stdout). Stderr is captured diagnostics.
type StdioLab struct {
	ClientIn  io.WriteCloser
	ClientOut io.ReadCloser
	Stderr    *bytes.Buffer
	outScan   *bufio.Scanner
	cancel    context.CancelFunc
	done      chan error
	Desc      Descriptor
	closeOnce sync.Once
}

// StartStdio runs a fixture on in-memory pipes. The serve loop exits when
// stdin closes or Close cancels the context.
func StartStdio(d Descriptor) (*StdioLab, error) {
	if d.Transport != TransportStdio {
		return nil, fmt.Errorf("testkit: fixture %q is transport %s, not stdio", d.Name, d.Transport)
	}
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithCancel(context.Background())
	sc := bufio.NewScanner(stdoutR)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lab := &StdioLab{
		Desc:      d,
		ClientIn:  stdinW,
		ClientOut: stdoutR,
		Stderr:    stderr,
		outScan:   sc,
		cancel:    cancel,
		done:      make(chan error, 1),
	}
	go func() {
		// Serve loop: owned by StdioLab; joined in Close.
		lab.done <- runStdio(ctx, d, stdinR, stdoutW, stderr)
		_ = stdinR.Close()
		_ = stdoutW.Close()
	}()
	return lab, nil
}

// Cancel stops in-flight delays without closing pipes so the client can still read a response.
func (l *StdioLab) Cancel() {
	l.cancel()
}

// Close cancels the serve loop, closes client pipes, and waits for the goroutine.
func (l *StdioLab) Close() error {
	var err error
	l.closeOnce.Do(func() {
		l.cancel()
		_ = l.ClientIn.Close()
		_ = l.ClientOut.Close()
		err = <-l.done
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			err = nil
		}
	})
	return err
}

func runStdio(ctx context.Context, d Descriptor, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	_, _ = fmt.Fprintf(stderr, "fixture: started name=%s era=%s\n", d.Name, d.Era)
	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if d.Malformed == MalformedJSON {
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return err
			}
			return io.EOF
		}
		_, err := io.WriteString(stdout, "not-json\n")
		return err
	}
	state := &SessionState{}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var req Request
		err := readJSONLine(sc, &req)
		if err != nil {
			if d.shutdownMode() == ShutdownWaitCancel && errors.Is(err, io.EOF) {
				<-ctx.Done()
				return ctx.Err()
			}
			return err
		}
		if d.Malformed == MalformedExitOnFirstRPC {
			return io.ErrClosedPipe
		}
		resp := handle(ctx, d, state, req)
		if req.Method == "notifications/initialized" && d.NotifyListChanged {
			if err := writeJSONLine(stdout, map[string]any{
				"jsonrpc": jsonRPCVersion,
				"method":  "notifications/tools/list_changed",
			}); err != nil {
				return err
			}
		}
		if req.ID == nil && resp.Error == nil {
			continue
		}
		if err := writeJSONLine(stdout, resp); err != nil {
			return err
		}
	}
}

// CallStdio writes one request and reads one response.
func CallStdio(lab *StdioLab, req Request) (Response, error) {
	if err := writeJSONLine(lab.ClientIn, req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := readJSONLine(lab.outScan, &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		panic(err)
	}
	return b
}
