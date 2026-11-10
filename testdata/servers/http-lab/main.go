package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/phillarmonic/mcpdev/internal/testkit"
)

const readyName = "ready.url"

func main() {
	name := os.Getenv("MCPDEV_FIXTURE")
	if name == "" {
		name = "legacy-http"
	}
	addr := os.Getenv("MCPDEV_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	readyFile := filepath.Join(".", readyName)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := testkit.RunHTTP(ctx, name, addr, func(url string) {
		fmt.Fprintln(os.Stderr, "fixture: listening", url)
		if writeErr := os.WriteFile(readyFile, []byte(url+"\n"), 0o600); writeErr != nil {
			fmt.Fprintln(os.Stderr, writeErr)
			os.Exit(1)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
