package watch

import (
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/watch/cycle"
)

// Deps are ports the watch probe needs after a supervisor handshake.
type Deps struct {
	Store  port.SnapshotStore
	Differ port.Differ
	Clock  port.Clock
	Smoke  cycle.SmokeRunner
}

// Request is readiness, smoke, and history policy for one watch session.
type Request struct {
	Config cycle.Config
}

// Engine constructs a probe/diff engine with no comparison baseline.
func Engine(deps Deps, req Request, onReport func(cycle.Result)) *cycle.Engine {
	return cycle.New(cycle.Options{
		Store:    deps.Store,
		Differ:   deps.Differ,
		Clock:    deps.Clock,
		Smoke:    deps.Smoke,
		Config:   req.Config,
		OnReport: onReport,
	})
}

// Probe wraps a stdio handshake so supervisor.Swap only promotes a ready generation.
func Probe(eng *cycle.Engine, handshake cycle.Handshake) cycle.Handshake {
	return eng.Wrap(handshake)
}
