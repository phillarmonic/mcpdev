package buildrun

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandNotFound(t *testing.T) {
	got := Run(t.Context(), Spec{Argv: []string{"mcpdev-missing-build-binary"}})
	if got.Class != ClassNotFound {
		t.Fatalf("class %s err %v", got.Class, got.Err)
	}
}

func TestNonzero(t *testing.T) {
	got := Run(t.Context(), Spec{
		Argv: []string{os.Args[0], "-test.run=^TestHelperExit2$", "-test.v=false"},
		Env:  map[string]string{"MCPDEV_BUILDRUN_HELPER": "exit2"},
	})
	if got.Class != ClassNonzero || got.ExitCode != 2 {
		t.Fatalf("class %s code %d err %v", got.Class, got.ExitCode, got.Err)
	}
}

func TestHelperExit2(t *testing.T) {
	if os.Getenv("MCPDEV_BUILDRUN_HELPER") != "exit2" {
		return
	}
	os.Exit(2)
}

func TestTimeoutAndCancel(t *testing.T) {
	timeout := Run(t.Context(), Spec{
		Argv: []string{"tool"},
		LookPath: func(string) (string, error) {
			return "/tool", nil
		},
		Run: func(context.Context, []string, string, []string, io.Writer, io.Writer) error {
			return context.DeadlineExceeded
		},
	})
	if timeout.Class != ClassTimeout {
		t.Fatalf("timeout class %s", timeout.Class)
	}
	canceled := Run(t.Context(), Spec{
		Argv: []string{"tool"},
		LookPath: func(string) (string, error) {
			return "/tool", nil
		},
		Run: func(context.Context, []string, string, []string, io.Writer, io.Writer) error {
			return context.Canceled
		},
	})
	if canceled.Class != ClassCanceled {
		t.Fatalf("cancel class %s", canceled.Class)
	}
}

func TestBoundedOutput(t *testing.T) {
	got := Run(t.Context(), Spec{
		Argv:        []string{"tool"},
		OutputLimit: 4,
		LookPath: func(string) (string, error) {
			return "/tool", nil
		},
		Run: func(_ context.Context, _ []string, _ string, _ []string, stdout, stderr io.Writer) error {
			_, _ = io.WriteString(stdout, "abcdefgh")
			_, _ = io.WriteString(stderr, "ijklmnop")
			return nil
		},
	})
	if got.Stdout != "abcd" || got.StdoutDropped != 4 {
		t.Fatalf("stdout %q dropped %d", got.Stdout, got.StdoutDropped)
	}
	if got.Stderr != "ijkl" || got.StderrDropped != 4 {
		t.Fatalf("stderr %q dropped %d", got.Stderr, got.StderrDropped)
	}
}

func TestLastGoodSurvivesFailure(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "app")
	staging := filepath.Join(dir, "app.partial")
	if err := os.WriteFile(final, []byte("good"), 0o600); err != nil {
		t.Fatal(err)
	}
	fail := Run(t.Context(), Spec{
		Argv:            []string{"tool"},
		ArtifactStaging: staging,
		ArtifactFinal:   final,
		PreviousGood:    final,
		LookPath: func(string) (string, error) {
			return "/tool", nil
		},
		Run: func(_ context.Context, _ []string, _ string, _ []string, _, _ io.Writer) error {
			if err := os.WriteFile(staging, []byte("bad"), 0o600); err != nil {
				return err
			}
			return errors.New("compile failed")
		},
	})
	if fail.Class == ClassOK || fail.Replaced {
		t.Fatalf("fail %+v", fail)
	}
	if fail.GoodPath != final {
		t.Fatalf("good path %s", fail.GoodPath)
	}
	body, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "good" {
		t.Fatalf("final %s", body)
	}
	if _, statErr := os.Stat(staging); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("staging still present: %v", statErr)
	}
	ok := Run(t.Context(), Spec{
		Argv:            []string{"tool"},
		ArtifactStaging: staging,
		ArtifactFinal:   final,
		PreviousGood:    final,
		LookPath: func(string) (string, error) {
			return "/tool", nil
		},
		Run: func(_ context.Context, _ []string, _ string, _ []string, _, _ io.Writer) error {
			return os.WriteFile(staging, []byte("newer"), 0o600)
		},
	})
	if !ok.Replaced || ok.GoodPath != final {
		t.Fatalf("ok %+v", ok)
	}
	body, err = os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "newer" {
		t.Fatalf("final %s", body)
	}
}

func TestUsageEmptyArgv(t *testing.T) {
	got := Run(t.Context(), Spec{})
	if got.Class != ClassUsage {
		t.Fatalf("class %s", got.Class)
	}
}
