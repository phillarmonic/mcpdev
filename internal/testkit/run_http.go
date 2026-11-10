package testkit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const httpShutdownTimeout = 5 * time.Second

// RunHTTP serves a named HTTP catalog fixture until ctx is cancelled.
// The MCP endpoint is always {base}/mcp. ready is called after the listener is bound.
func RunHTTP(ctx context.Context, name, addr string, ready func(url string)) error {
	d, err := Named(name)
	if err != nil {
		return err
	}
	if d.Transport != TransportHTTP {
		return fmt.Errorf("testkit: fixture %q is transport %s, not http", name, d.Transport)
	}
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	lab := &HTTPLab{Desc: d, state: &SessionState{}}
	mux := http.NewServeMux()
	mux.Handle("/mcp", lab)
	mux.Handle("/mcp/", lab)
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String() + "/mcp"
	errCh := make(chan error, 1)
	go func() {
		serveErr := srv.Serve(ln)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()
	if ready != nil {
		ready(url)
	}
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpShutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		return <-errCh
	case serveErr := <-errCh:
		return serveErr
	}
}
