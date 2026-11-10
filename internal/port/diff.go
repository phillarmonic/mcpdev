package port

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/model"
)

// Differ compares two public-interface snapshots.
// Incomplete documents must not be reported as equal.
type Differ interface {
	Diff(ctx context.Context, from, to model.Snapshot) (model.Diff, error)
}
