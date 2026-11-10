package cycle

import "github.com/phillarmonic/mcpdev/internal/model"

// SetBaseline installs a comparison snapshot before the first ready generation.
func (e *Engine) SetBaseline(snap model.Snapshot) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastReady = snap
	e.hasReady = true
}
