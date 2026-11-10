package inspect

import (
	"slices"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func filterNames(inv model.Inventory, names []string) model.Inventory {
	if len(names) == 0 {
		return inv
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	inv.Tools.Tools = slices.DeleteFunc(inv.Tools.Tools, func(t model.Tool) bool {
		_, ok := want[t.Name]
		return !ok
	})
	inv.Prompts.Prompts = slices.DeleteFunc(inv.Prompts.Prompts, func(p model.Prompt) bool {
		_, ok := want[p.Name]
		return !ok
	})
	inv.Resources.Resources = slices.DeleteFunc(inv.Resources.Resources, func(r model.Resource) bool {
		_, ok := want[r.URI]
		if ok {
			return false
		}
		_, ok = want[r.Name]
		return !ok
	})
	inv.Templates.Templates = slices.DeleteFunc(inv.Templates.Templates, func(r model.ResourceTemplate) bool {
		_, ok := want[r.URITemplate]
		if ok {
			return false
		}
		_, ok = want[r.Name]
		return !ok
	})
	return inv
}
