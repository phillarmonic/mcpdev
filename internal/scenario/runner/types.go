package runner

import (
	"github.com/phillarmonic/mcpdev/internal/assert"
	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Deps are the same ports one-shot commands use.
type Deps struct {
	Connector port.Connector
	Store     port.SnapshotStore
	Differ    port.Differ
	Clock     port.Clock
}

func (d Deps) clock() port.Clock {
	if d.Clock != nil {
		return d.Clock
	}
	return clock.Wall{}
}

// Request is one loaded suite plus a resolved connection target.
type Request struct {
	Env     map[string]string
	File    string
	WorkDir string
	Target  port.Target
	Jobs    int
}

// Report is one suite execution.
type Report struct {
	File       string       `json:"file"`
	Name       string       `json:"name"`
	Status     string       `json:"status"`
	Category   string       `json:"category,omitzero"`
	Message    string       `json:"message,omitzero"`
	Cases      []CaseReport `json:"cases"`
	DurationMS int64        `json:"duration_ms"`
}

// CaseReport is one case with isolated session state.
type CaseReport struct {
	Name       string       `json:"name"`
	Status     string       `json:"status"`
	Category   string       `json:"category,omitzero"`
	Message    string       `json:"message,omitzero"`
	Steps      []StepReport `json:"steps"`
	DurationMS int64        `json:"duration_ms"`
}

// StepReport is one operation plus assertion evidence.
type StepReport struct {
	ID         string          `json:"id,omitzero"`
	Op         string          `json:"op"`
	Status     string          `json:"status"`
	Category   string          `json:"category,omitzero"`
	Message    string          `json:"message,omitzero"`
	Assertions []assert.Result `json:"assertions,omitzero"`
	DurationMS int64           `json:"duration_ms"`
}

const (
	statusPass  = "pass"
	statusFail  = "fail"
	statusSkip  = "skip"
	statusError = "error"
)
