package snapshot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/inventory"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Store implements port.SnapshotStore.
type Store struct {
	Clock   port.Clock
	Version string
	Target  port.Target
	Limits  inventory.Limits
}

var _ port.SnapshotStore = Store{}

// Capture walks the session inventory into a snapshot document.
func (s Store) Capture(ctx context.Context, sess port.Session, kinds []model.PrimitiveKind) (model.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.Snapshot{}, errclass.Incomplete{Err: err}
	}
	if sess == nil {
		return model.Snapshot{}, errclass.Internal{Err: errors.New("snapshot: nil session")}
	}
	info := sess.Info()
	inv := inventory.Collect(ctx, info, sess.Inventory(), s.Limits, kinds...)
	return FromInventory(inv, Meta{
		CapturedAt:       s.now(),
		GeneratorVersion: s.generator(),
		Kind:             s.Target.Kind,
		Profile:          s.Target.Profile,
		URL:              s.Target.URL,
		Command:          s.Target.Command,
		Headers:          s.Target.Headers,
		SessionID:        info.SessionID,
	})
}

// Write encodes snap to path. Incomplete documents are written as incomplete.
func (s Store) Write(ctx context.Context, path string, snap model.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return errclass.Internal{Err: err}
	}
	clean, err := validatedPath(path)
	if err != nil {
		return err
	}
	body, err := Encode(snap)
	if err != nil {
		return err
	}
	dir := filepath.Dir(clean)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, ".mcpdev-snapshot-*.tmp")
	if err != nil {
		return errclass.Internal{Err: fmt.Errorf("snapshot: temp %s: %w", dir, err)}
	}
	tmpName := tmp.Name()
	replaced := false
	defer func() {
		_ = tmp.Close()
		if !replaced {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return errclass.Internal{Err: fmt.Errorf("snapshot: chmod %s: %w", tmpName, err)}
	}
	if _, err := tmp.Write(body); err != nil {
		return errclass.Internal{Err: fmt.Errorf("snapshot: write %s: %w", clean, err)}
	}
	if err := tmp.Sync(); err != nil {
		return errclass.Internal{Err: fmt.Errorf("snapshot: sync %s: %w", clean, err)}
	}
	if err := tmp.Close(); err != nil {
		return errclass.Internal{Err: fmt.Errorf("snapshot: close %s: %w", clean, err)}
	}
	if err := os.Rename(tmpName, clean); err != nil {
		return errclass.Internal{Err: fmt.Errorf("snapshot: replace %s: %w", clean, err)}
	}
	replaced = true
	return nil
}

// Read loads and validates a snapshot file.
func (s Store) Read(ctx context.Context, path string) (model.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.Snapshot{}, errclass.Internal{Err: err}
	}
	clean, err := validatedPath(path)
	if err != nil {
		return model.Snapshot{}, err
	}
	body, err := os.ReadFile(clean) //nolint:gosec // G304: snapshot path is operator-supplied after validation
	if err != nil {
		return model.Snapshot{}, errclass.Internal{Err: fmt.Errorf("snapshot: read %s: %w", clean, err)}
	}
	return Decode(body)
}

func (s Store) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Store) generator() string {
	if s.Version != "" {
		return s.Version
	}
	return "0.1.1"
}

func validatedPath(path string) (string, error) {
	if path == "" || path == "-" {
		return "", errclass.Usage{Msg: "snapshot: path is required"}
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err == nil && info.IsDir() {
		return "", errclass.Usage{Msg: "snapshot: path is a directory"}
	}
	return clean, nil
}
