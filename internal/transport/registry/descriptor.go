package registry

import (
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/extension"
)

// Kind classifies who owns a connector.
type Kind string

const (
	KindBuiltin   Kind = "builtin"
	KindExtension Kind = "extension"
)

// Descriptor is the static registration record for one connector name.
type Descriptor struct {
	Semantics extension.Semantics
	Name      string
	Kind      Kind
	Standard  bool
}

func (d Descriptor) validate() error {
	if d.Name == "" {
		return errclass.Usage{Msg: "connector: name is required"}
	}
	switch d.Kind {
	case KindBuiltin, KindExtension:
	default:
		return errclass.Usage{Msg: "connector: kind must be builtin or extension"}
	}
	if d.Name == NameForbidden && (d.Standard || d.Kind == KindBuiltin) {
		return errclass.Usage{Msg: "connector: grpc is not a standard MCP transport"}
	}
	if d.Kind == KindBuiltin && !d.Standard {
		return errclass.Usage{Msg: "connector: built-ins must be marked standard"}
	}
	if d.Kind == KindExtension && d.Standard {
		return errclass.Usage{Msg: "connector: extensions are not MCP-standard transports"}
	}
	return d.Semantics.Validate()
}
