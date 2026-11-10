package schema

import (
	"fmt"
	"strings"
)

// Issue is one document problem at a JSON Pointer.
type Issue struct {
	Path    string
	Message string
}

func (i Issue) String() string {
	if i.Path == "" || i.Path == "/" {
		return i.Message
	}
	return i.Path + ": " + i.Message
}

// Error lists every useful validation problem. Invalid suites must not be executed.
type Error struct {
	File   string
	Issues []Issue
}

func (e Error) Error() string {
	if len(e.Issues) == 0 {
		if e.File != "" {
			return e.File + ": invalid scenario suite"
		}
		return "invalid scenario suite"
	}
	parts := make([]string, 0, len(e.Issues))
	for _, is := range e.Issues {
		parts = append(parts, is.String())
	}
	joined := strings.Join(parts, "; ")
	if e.File != "" {
		return fmt.Sprintf("%s: %s", e.File, joined)
	}
	return joined
}
