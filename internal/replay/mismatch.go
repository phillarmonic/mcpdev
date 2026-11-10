package replay

import (
	"fmt"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Mismatch is the first request that did not match the script.
type Mismatch struct {
	Expected string
	Actual   string
	Method   string
	Path     string
	Field    string
	Step     int
}

func (m Mismatch) Error() string {
	loc := fmt.Sprintf("step %d", m.Step)
	if m.Method != "" {
		loc += " method " + m.Method
	}
	switch m.Field {
	case "method":
		return fmt.Sprintf("replay: %s: expected method %s, got %s", loc, m.Expected, m.Actual)
	case "metadata":
		return fmt.Sprintf("replay: %s: metadata mismatch at %s: expected %s, got %s", loc, pointer(m.Path), m.Expected, m.Actual)
	default:
		return fmt.Sprintf("replay: %s: payload mismatch at %s: expected %s, got %s", loc, pointer(m.Path), m.Expected, m.Actual)
	}
}

func (m Mismatch) ExitClass() int { return errclass.ExitProtocol }

func pointer(p string) string {
	if p == "" {
		return "/"
	}
	return p
}
