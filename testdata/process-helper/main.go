package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
)

func main() {
	switch os.Getenv("MCPDEV_HELPER") {
	case "echo-env":
		key := os.Getenv("MCPDEV_HELPER_KEY")
		fmt.Fprintf(os.Stdout, "{\"env\":%q}\n", os.Getenv(key))
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "cwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "{\"cwd\":%q}\n", wd)
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "stderr-flood":
		chunk := strings.Repeat("x", 1024)
		for range 128 {
			_, _ = io.WriteString(os.Stderr, chunk)
		}
		fmt.Fprint(os.Stdout, "{\"ok\":true}\n")
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "early-exit":
		fmt.Fprint(os.Stderr, "startup failed\n")
		os.Exit(2)
	case "bad-stdout":
		fmt.Fprint(os.Stdout, "not-json\n")
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "ignore-term":
		signal.Ignore(os.Interrupt)
		fmt.Fprint(os.Stdout, "{\"ok\":true}\n")
		select {}
	default:
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			fmt.Fprintln(os.Stdout, sc.Text())
		}
	}
}
