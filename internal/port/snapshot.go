package port

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/model"
)

// SnapshotStore canonicalizes, persists, and reloads public-interface snapshots.
// Capture uses Session.Inventory(); it does not import the SDK.
type SnapshotStore interface {
	Capture(ctx context.Context, sess Session, kinds []model.PrimitiveKind) (model.Snapshot, error)
	Write(ctx context.Context, path string, snap model.Snapshot) error
	Read(ctx context.Context, path string) (model.Snapshot, error)
}
