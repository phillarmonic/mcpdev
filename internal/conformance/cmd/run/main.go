package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/conformance"
	"github.com/phillarmonic/mcpdev/internal/platform"
)

func main() {
	revision := flag.String("revision", "", "single requirement set (default: every pin entry, separately)")
	outDir := flag.String("out", "artifacts/conformance", "directory for per-revision runner output")
	flag.Parse()
	root := moduleRoot()
	pin, err := conformance.LoadPin(root)
	if err != nil {
		fatal(err)
	}
	revs := pin.Revisions
	if *revision != "" {
		rev, err := pin.RevisionByRequirements(*revision)
		if err != nil {
			fatal(err)
		}
		revs = []conformance.Revision{rev}
	}
	var failed []string
	for _, rev := range revs {
		fmt.Fprintf(os.Stderr, "conformance: scoring %s on fixture %s (package %s)\n", rev.Requirements, rev.Fixture, pin.NPXPackage())
		if err := runRevision(root, pin, rev, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "conformance: %s: %v\n", rev.Requirements, err)
			failed = append(failed, rev.Requirements)
			continue
		}
		fmt.Fprintf(os.Stderr, "conformance: %s finished\n", rev.Requirements)
	}
	if len(failed) > 0 {
		fatal(fmt.Errorf("failed revisions: %s", strings.Join(failed, ", ")))
	}
}

func runRevision(root string, pin conformance.Pin, rev conformance.Revision, outDir string) error {
	revOut := filepath.Join(root, outDir, rev.Requirements)
	if err := os.MkdirAll(revOut, 0o750); err != nil {
		return err
	}
	labBin := platform.BuildPath(revOut, "http-lab")
	build := exec.Command("go", "build", "-o", labBin, ".") //nolint:gosec // G204: fixed testdata/servers/http-lab
	build.Dir = filepath.Join(root, "testdata", "servers", "http-lab")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build http-lab: %w", err)
	}
	readyFile := filepath.Join(revOut, "ready.url")
	_ = os.Remove(readyFile)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lab := exec.CommandContext(ctx, labBin) //nolint:gosec // G204: binary is go-build output
	lab.Dir = revOut
	lab.Env = append(os.Environ(), "MCPDEV_FIXTURE="+rev.Fixture)
	lab.Stdout = os.Stderr
	lab.Stderr = os.Stderr
	if err := lab.Start(); err != nil {
		return fmt.Errorf("start http-lab: %w", err)
	}
	defer func() {
		cancel()
		_ = lab.Wait()
	}()
	url, err := waitReady(readyFile, 15*time.Second)
	if err != nil {
		return err
	}
	baseline := filepath.Join(root, rev.Baseline)
	args := append([]string{"--yes", pin.NPXPackage()}, conformance.ServerArgs(url, rev.Requirements, baseline, revOut)...)
	npx := exec.Command("npx", args...) //nolint:gosec // G204: package@version from pin.json, args are flags
	npx.Dir = revOut
	npx.Stdout = os.Stdout
	npx.Stderr = os.Stderr
	if err := npx.Run(); err != nil {
		return fmt.Errorf("npx %s: %w", pin.NPXPackage(), err)
	}
	return nil
}

func waitReady(path string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path) //nolint:gosec // G304: ready.url is created in the revision output dir
		if err == nil {
			url := strings.TrimSpace(string(raw))
			if url != "" {
				return url, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return "", fmt.Errorf("timeout waiting for %s", path)
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "conformance: runtime.Caller failed")
		os.Exit(2)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
