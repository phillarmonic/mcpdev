package testkit

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestRunHTTPServesMCPPathAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- RunHTTP(ctx, "modern-http", "127.0.0.1:0", func(url string) {
			ready <- url
		})
	}()
	var url string
	select {
	case url = <-ready:
	case err := <-done:
		t.Fatalf("server exited before ready: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for listen")
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := waitDo(t, req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET %s: got %d, want 405 for 2026-07-28", url, resp.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for shutdown")
	}
}

func TestRunHTTPRejectsStdioFixture(t *testing.T) {
	err := RunHTTP(t.Context(), "legacy-stdio", "127.0.0.1:0", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func waitDo(t *testing.T, req *http.Request) (*http.Response, error) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		attempt, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(attempt)
		if err == nil {
			return resp, nil
		}
		last = err
		select {
		case <-t.Context().Done():
			return nil, t.Context().Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	return nil, last
}
