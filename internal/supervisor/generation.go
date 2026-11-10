package supervisor

import (
	"fmt"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// Generation is a durable snapshot of one rebuild/start attempt.
type Generation struct {
	State           State
	BuildLog        string
	Stderr          string
	TerminateReason string
	Protocol        model.SessionInfo
	ID              int64
	HasProtocol     bool
}

// View is a consistent controller snapshot.
type View struct {
	Current    *Generation
	Incoming   *Generation
	LastFailed *Generation
	State      State
}

// StaleError means a build or start belonged to a superseded generation.
type StaleError struct {
	Generation int64
}

func (e StaleError) Error() string {
	return fmt.Sprintf("supervisor: stale generation %d", e.Generation)
}

func stale(gen int64) error {
	return errclass.Incomplete{Err: StaleError{Generation: gen}}
}
