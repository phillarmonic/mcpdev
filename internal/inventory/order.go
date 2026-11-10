package inventory

import (
	"cmp"
	"slices"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func sortSection(sec *model.CatalogSection) {
	slices.SortFunc(sec.Tools, func(a, b model.Tool) int { return cmp.Compare(a.Name, b.Name) })
	slices.SortFunc(sec.Prompts, func(a, b model.Prompt) int { return cmp.Compare(a.Name, b.Name) })
	slices.SortFunc(sec.Resources, func(a, b model.Resource) int {
		if n := cmp.Compare(a.URI, b.URI); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	})
	slices.SortFunc(sec.Templates, func(a, b model.ResourceTemplate) int {
		if n := cmp.Compare(a.URITemplate, b.URITemplate); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	})
	slices.Sort(sec.Reasons)
}
