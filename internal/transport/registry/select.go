package registry

import (
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// Select maps Request to a registered connector without constructing it.
func (r *Registry) Select(req Request) (Selection, error) {
	name, err := inferName(req)
	if err != nil {
		return Selection{}, err
	}
	if name == NameForbidden {
		return Selection{}, errclass.Usage{Msg: "connector: grpc is not a standard MCP transport"}
	}
	proto, err := normalizeProtocol(req.Protocol)
	if err != nil {
		return Selection{}, err
	}
	d, ok := r.Lookup(name)
	if !ok {
		return Selection{}, errclass.Usage{Msg: "connector: unknown name " + name}
	}
	sel := Selection{Descriptor: d, EraPolicy: proto, Constructed: name}
	switch name {
	case NameStdio:
		if req.URL != "" {
			return Selection{}, errclass.Usage{Msg: "connector: stdio does not take --url"}
		}
		if len(req.Command) == 0 {
			return Selection{}, errclass.Usage{Msg: "connector: stdio requires --command"}
		}
	case NameHTTP, NameHTTP2025, NameHTTP2026:
		if len(req.Command) > 0 {
			return Selection{}, errclass.Usage{Msg: "connector: http does not take --command"}
		}
		if req.URL == "" {
			return Selection{}, errclass.Usage{Msg: "connector: http requires --url"}
		}
		constructed, err := httpConstructed(name, proto)
		if err != nil {
			return Selection{}, err
		}
		sel.Constructed = constructed
	default:
		// extension: command/url rules are adapter-specific
	}
	return sel, nil
}

func inferName(req Request) (string, error) {
	name := req.Connector
	hasCmd := len(req.Command) > 0
	hasURL := req.URL != ""
	if name == "" {
		switch {
		case hasCmd && hasURL:
			return "", errclass.Usage{Msg: "connector: --command and --url are mutually exclusive"}
		case hasCmd:
			return NameStdio, nil
		case hasURL:
			return NameHTTP, nil
		default:
			return "", errclass.Usage{Msg: "connector: --command or --url is required"}
		}
	}
	return name, nil
}

func normalizeProtocol(p string) (string, error) {
	switch p {
	case "", ProtocolAuto:
		return ProtocolAuto, nil
	case string(model.Era20251125), string(model.Era20260728):
		return p, nil
	default:
		return "", errclass.Usage{Msg: "connector: unknown protocol " + p}
	}
}

func httpConstructed(name, proto string) (string, error) {
	switch name {
	case NameHTTP2025:
		if proto == string(model.Era20260728) {
			return "", errclass.Usage{Msg: "connector: http-2025-11-25 cannot speak 2026-07-28"}
		}
		return NameHTTP2025, nil
	case NameHTTP2026:
		if proto == string(model.Era20251125) {
			return "", errclass.Usage{Msg: "connector: http-2026-07-28 cannot speak 2025-11-25"}
		}
		return NameHTTP2026, nil
	case NameHTTP:
		switch proto {
		case string(model.Era20251125):
			return NameHTTP2025, nil
		case string(model.Era20260728), ProtocolAuto:
			return NameHTTP2026, nil
		default:
			return "", errclass.Usage{Msg: "connector: unknown protocol " + proto}
		}
	default:
		return "", errclass.Internal{Err: errNotHTTP}
	}
}
