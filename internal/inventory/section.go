package inventory

import (
	"context"
	"slices"

	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

func collectSection(ctx context.Context, reader port.InventoryReader, kind model.PrimitiveKind, advertised bool, limits Limits) model.CatalogSection {
	sec := model.CatalogSection{Kind: kind, Status: model.SectionComplete}
	if !advertised {
		sec.Status = model.SectionUnsupported
		sec.Reasons = []string{model.ReasonCapabilityAbsent}
		return sec
	}
	seen := map[string]struct{}{"": {}}
	cursor := ""
	ids := map[string]struct{}{}
	for {
		if err := ctx.Err(); err != nil {
			return stop(sec, model.SectionIncomplete, model.ReasonCollectionCanceled, cursor)
		}
		page, err := reader.List(ctx, kind, model.Page{Cursor: cursor})
		if err != nil {
			return stop(sec, model.SectionFailed, model.ReasonListFailed, cursor)
		}
		sec.Pages++
		if page.Incomplete {
			sec = note(sec, model.SectionIncomplete, model.ReasonPageIncomplete)
		}
		var hitLimit bool
		sec, hitLimit = appendPage(sec, page, ids, limits.MaxItems)
		if hitLimit {
			return stop(sec, model.SectionIncomplete, model.ReasonItemLimit, cursor)
		}
		if page.NextCursor == "" {
			sec.StoppedCursor = ""
			return sec
		}
		if _, loop := seen[page.NextCursor]; loop {
			return stop(sec, model.SectionIncomplete, model.ReasonCursorLoop, page.NextCursor)
		}
		if sec.Pages >= limits.MaxPages {
			return stop(sec, model.SectionIncomplete, model.ReasonPaginationStopped, page.NextCursor)
		}
		seen[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
	}
}

func appendPage(sec model.CatalogSection, page model.CatalogPage, ids map[string]struct{}, maxItems int) (model.CatalogSection, bool) {
	switch sec.Kind {
	case model.KindTools:
		return appendTools(sec, page.Tools, ids, maxItems)
	case model.KindPrompts:
		return appendPrompts(sec, page.Prompts, ids, maxItems)
	case model.KindResources:
		return appendResources(sec, page.Resources, ids, maxItems)
	default:
		return appendTemplates(sec, page.Templates, ids, maxItems)
	}
}

func appendTools(sec model.CatalogSection, items []model.Tool, ids map[string]struct{}, maxItems int) (model.CatalogSection, bool) {
	for _, item := range items {
		if sec.ItemCount >= maxItems {
			return sec, true
		}
		if !claim(ids, item.Name) {
			sec = note(sec, sec.Status, model.ReasonDuplicateItem)
			continue
		}
		sec.Tools = append(sec.Tools, item)
		sec.ItemCount++
	}
	return sec, false
}

func appendPrompts(sec model.CatalogSection, items []model.Prompt, ids map[string]struct{}, maxItems int) (model.CatalogSection, bool) {
	for _, item := range items {
		if sec.ItemCount >= maxItems {
			return sec, true
		}
		if !claim(ids, item.Name) {
			sec = note(sec, sec.Status, model.ReasonDuplicateItem)
			continue
		}
		sec.Prompts = append(sec.Prompts, item)
		sec.ItemCount++
	}
	return sec, false
}

func appendResources(sec model.CatalogSection, items []model.Resource, ids map[string]struct{}, maxItems int) (model.CatalogSection, bool) {
	for _, item := range items {
		if sec.ItemCount >= maxItems {
			return sec, true
		}
		if !claim(ids, item.URI) {
			sec = note(sec, sec.Status, model.ReasonDuplicateItem)
			continue
		}
		sec.Resources = append(sec.Resources, item)
		sec.ItemCount++
	}
	return sec, false
}

func appendTemplates(sec model.CatalogSection, items []model.ResourceTemplate, ids map[string]struct{}, maxItems int) (model.CatalogSection, bool) {
	for _, item := range items {
		if sec.ItemCount >= maxItems {
			return sec, true
		}
		if !claim(ids, item.URITemplate) {
			sec = note(sec, sec.Status, model.ReasonDuplicateItem)
			continue
		}
		sec.Templates = append(sec.Templates, item)
		sec.ItemCount++
	}
	return sec, false
}

func claim(ids map[string]struct{}, id string) bool {
	if _, ok := ids[id]; ok {
		return false
	}
	ids[id] = struct{}{}
	return true
}

func note(sec model.CatalogSection, status model.SectionStatus, reason string) model.CatalogSection {
	if status != model.SectionComplete {
		sec.Status = status
	}
	if !slices.Contains(sec.Reasons, reason) {
		sec.Reasons = append(sec.Reasons, reason)
	}
	return sec
}

func stop(sec model.CatalogSection, status model.SectionStatus, reason, cursor string) model.CatalogSection {
	sec = note(sec, status, reason)
	sec.Status = status
	sec.StoppedCursor = cursor
	return sec
}
