package main

import (
	"fmt"
	"os"

	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func main() {
	name := os.Getenv("MCPDEV_FIXTURE")
	if name == "" {
		name = "legacy-stdio"
	}
	if err := testkit.RunStdio(os.Stdin, os.Stdout, os.Stderr, name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
