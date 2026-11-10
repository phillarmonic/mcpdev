package watch

import "time"

// Kind is one operator-visible watch transition.
type Kind string

const (
	KindFS        Kind = "fs"
	KindBuild     Kind = "build"
	KindRestart   Kind = "restart"
	KindNegotiate Kind = "negotiate"
	KindProbe     Kind = "probe"
	KindDiff      Kind = "diff"
	KindScenario  Kind = "scenario"
	KindRecovery  Kind = "recovery"
	KindShutdown  Kind = "shutdown"
)

// Note is one append-only watch status record shared by TTY and JSON Lines.
type Note struct {
	At         time.Time `json:"ts"`
	Kind       Kind      `json:"kind"`
	ID         string    `json:"id"`
	State      string    `json:"state"`
	Class      string    `json:"class,omitzero"`
	Outcome    string    `json:"outcome,omitzero"`
	Message    string    `json:"message,omitzero"`
	Path       string    `json:"path,omitzero"`
	Generation int64     `json:"generation,omitzero"`
	Incomplete bool      `json:"incomplete,omitzero"`
}
