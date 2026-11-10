package cycle

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/port"
)

// Readiness is how a generation becomes eligible for promotion after handshake.
type Readiness string

const (
	// ReadinessInventory requires a complete public-interface capture.
	ReadinessInventory Readiness = "inventory"
	// ReadinessScenario requires named smoke scenarios, then still captures a snapshot.
	ReadinessScenario Readiness = "scenario"
)

const defaultHistoryBound = 16

// Config is the operator-facing readiness and reporting policy.
type Config struct {
	Readiness          Readiness
	ReadinessScenarios []string
	Smoke              []string
	Ignore             []string
	HistoryBound       int
	AllowIncomplete    bool // recorded; incomplete captures still fail readiness
}

func (c Config) resolved() Config {
	if c.Readiness == "" {
		c.Readiness = ReadinessInventory
	}
	if c.HistoryBound <= 0 {
		c.HistoryBound = defaultHistoryBound
	}
	return c
}

// SmokeRunner runs named scenarios against the generation already under probe.
// Implementations must not spawn a second child for the same generation.
type SmokeRunner interface {
	Run(ctx context.Context, sess port.Session, names []string) error
}

// Handshake turns a live stdio child into a negotiated session.
type Handshake func(ctx context.Context, proc port.Proc) (port.Session, error)
