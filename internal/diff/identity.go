package diff

import "github.com/phillarmonic/mcpdev/internal/model"

func diffIdentity(r *report, from, to model.Snapshot) {
	diffScalar(r, pointer("protocol", "era"), string(from.Protocol.Era), string(to.Protocol.Era),
		model.SeverityBreaking, "Negotiated protocol era changed.")
	diffScalar(r, pointer("protocol", "version"), from.Protocol.Version, to.Protocol.Version,
		model.SeverityBreaking, "Negotiated protocol version changed.")
	diffScalar(r, pointer("protocol", "handshake"), string(from.Protocol.Handshake), string(to.Protocol.Handshake),
		model.SeverityBreaking, "Handshake kind changed.")
	diffScalar(r, pointer("protocol", "discover"), string(from.Protocol.Discover), string(to.Protocol.Discover),
		model.SeverityMetadata, "Discover state changed.")
	diffScalar(r, pointer("server", "name"), from.Server.Name, to.Server.Name,
		model.SeverityBreaking, "Server name changed.")
	diffScalar(r, pointer("server", "title"), from.Server.Title, to.Server.Title,
		model.SeverityMetadata, "Server title changed.")
	diffScalar(r, pointer("server", "version"), from.Server.Version, to.Server.Version,
		model.SeverityMetadata, "Server version string changed.")
	diffScalar(r, pointer("server", "instructions"), from.Server.Instructions, to.Server.Instructions,
		model.SeverityMetadata, "Server instructions changed.")
}

func diffCapabilities(r *report, from, to model.ServerCapabilities) {
	diffFlag(r, pointer("capabilities", "tools"), from.Tools, to.Tools, "tools")
	diffFlag(r, pointer("capabilities", "prompts"), from.Prompts, to.Prompts, "prompts")
	diffFlag(r, pointer("capabilities", "resources"), from.Resources, to.Resources, "resources")
	diffFlag(r, pointer("capabilities", "completions"), from.Completions, to.Completions, "completions")
	diffFlag(r, pointer("capabilities", "logging"), from.Logging, to.Logging, "logging")
	diffFlag(r, pointer("capabilities", "tools_list_changed"), from.ToolsListChanged, to.ToolsListChanged, "tools listChanged")
	diffFlag(r, pointer("capabilities", "prompts_list_changed"), from.PromptsListChanged, to.PromptsListChanged, "prompts listChanged")
	diffFlag(r, pointer("capabilities", "resources_list_changed"), from.ResourcesListChanged, to.ResourcesListChanged, "resources listChanged")
	diffFlag(r, pointer("capabilities", "resource_subscribe"), from.ResourceSubscribe, to.ResourceSubscribe, "resource subscribe")
	diffNameSet(r, pointer("capabilities", "experimental"), from.Experimental, to.Experimental, "experimental capability")
	diffNameSet(r, pointer("capabilities", "extensions"), from.Extensions, to.Extensions, "extension")
}

func diffFlag(r *report, path string, from, to bool, label string) {
	if from == to {
		return
	}
	if from && !to {
		r.add(path, model.CategoryRemoval, model.SeverityBreaking, from, to, "Removed "+label+" capability.")
		return
	}
	r.add(path, model.CategoryAddition, model.SeverityAdditive, from, to, "Added "+label+" capability.")
}

func diffNameSet(r *report, base string, from, to []string, label string) {
	fromSet := nameSet(from)
	toSet := nameSet(to)
	for _, name := range sortedKeys(fromSet) {
		if _, ok := toSet[name]; !ok {
			r.add(pointerJoin(base, name), model.CategoryRemoval, model.SeverityBreaking, name, nil, "Removed "+label+".")
		}
	}
	for _, name := range sortedKeys(toSet) {
		if _, ok := fromSet[name]; !ok {
			r.add(pointerJoin(base, name), model.CategoryAddition, model.SeverityAdditive, nil, name, "Added "+label+".")
		}
	}
}

func diffScalar(r *report, path, from, to string, sev model.DiffSeverity, rationale string) {
	if from == to {
		return
	}
	r.add(path, model.CategoryChange, sev, omitEmpty(from), omitEmpty(to), rationale)
}

func omitEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func pointerJoin(base, extra string) string {
	return base + "/" + escapePointer(extra)
}
