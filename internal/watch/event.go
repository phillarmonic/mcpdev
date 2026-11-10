package watch

import "strings"

// Op is a normalized, platform-independent file change.
type Op uint32

const (
	OpCreate Op = 1 << iota
	OpWrite
	OpRemove
	OpRename
	OpChmod
)

// Event is one normalized path change relative to a watch root.
type Event struct {
	Root string
	Path string
	Op   Op
}

func (o Op) String() string {
	var parts []string
	if o&OpCreate != 0 {
		parts = append(parts, "create")
	}
	if o&OpWrite != 0 {
		parts = append(parts, "write")
	}
	if o&OpRemove != 0 {
		parts = append(parts, "remove")
	}
	if o&OpRename != 0 {
		parts = append(parts, "rename")
	}
	if o&OpChmod != 0 {
		parts = append(parts, "chmod")
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "|")
}
