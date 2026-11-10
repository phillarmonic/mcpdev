package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/release"
	"github.com/phillarmonic/mcpdev/internal/version"
)

func main() {
	outDir := flag.String("out", "dist", "output directory")
	noSmoke := flag.Bool("no-smoke", false, "skip host archive inspect")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		fatal(fmt.Errorf("pack: run from the module root: %w", err))
	}
	opt := stamp(root)
	opt.Root = root
	opt.OutDir = filepath.Join(root, *outDir)
	opt.Smoke = !*noSmoke
	if err := release.Pack(opt); err != nil {
		fatal(err)
	}
}

func stamp(root string) release.Options {
	opt := release.Options{
		Version:   version.Version,
		Commit:    "unknown",
		BuildTime: "unknown",
		Policy:    version.PolicyUnknown,
		Epoch:     0,
	}
	if v := strings.TrimSpace(os.Getenv("MCPDEV_VERSION")); v != "" {
		opt.Version = strings.TrimPrefix(v, "v")
	} else if out, err := git(root, "describe", "--tags", "--always", "--dirty"); err == nil {
		opt.Version = strings.TrimPrefix(out, "v")
	}
	if out, err := git(root, "rev-parse", "HEAD"); err == nil {
		opt.Commit = out
	}
	if s := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			fatal(fmt.Errorf("pack: SOURCE_DATE_EPOCH: %w", err))
		}
		opt.Epoch = n
		opt.BuildTime = time.Unix(n, 0).UTC().Format(time.RFC3339)
		opt.Policy = version.PolicySourceDateEpoch
		return opt
	}
	if out, err := git(root, "log", "-1", "--format=%ct"); err == nil {
		n, err := strconv.ParseInt(out, 10, 64)
		if err == nil {
			opt.Epoch = n
			opt.BuildTime = time.Unix(n, 0).UTC().Format(time.RFC3339)
			opt.Policy = version.PolicyGitCommit
		}
	}
	return opt
}

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...) //nolint:gosec // G204: git subcommands are a fixed allowlist
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
