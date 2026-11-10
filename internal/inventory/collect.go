package inventory

import (
	"context"
	"slices"

	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Collect reads advertised catalog families through reader.
// kinds empty means all four families. Section failures stay on the document.
func Collect(ctx context.Context, info model.SessionInfo, reader port.InventoryReader, limits Limits, kinds ...model.PrimitiveKind) model.Inventory {
	limits = limits.resolved()
	inv := model.Inventory{
		Instructions:    info.Instructions,
		ServerName:      info.ServerName,
		ServerTitle:     info.ServerTitle,
		ServerVersion:   info.ServerVersion,
		ProtocolVersion: info.ProtocolVersion,
		Handshake:       info.Handshake,
		DiscoverState:   info.DiscoverState,
		Era:             info.Era,
		Capabilities:    info.Capabilities,
	}
	if wantKind(kinds, model.KindTools) {
		inv.Tools = collectSection(ctx, reader, model.KindTools, info.Capabilities.Advertises(model.KindTools), limits)
		sortSection(&inv.Tools)
	}
	if wantKind(kinds, model.KindPrompts) {
		inv.Prompts = collectSection(ctx, reader, model.KindPrompts, info.Capabilities.Advertises(model.KindPrompts), limits)
		sortSection(&inv.Prompts)
	}
	if wantKind(kinds, model.KindResources) {
		inv.Resources = collectSection(ctx, reader, model.KindResources, info.Capabilities.Advertises(model.KindResources), limits)
		sortSection(&inv.Resources)
	}
	if wantKind(kinds, model.KindTemplates) {
		inv.Templates = collectSection(ctx, reader, model.KindTemplates, info.Capabilities.Advertises(model.KindTemplates), limits)
		sortSection(&inv.Templates)
	}
	for _, sec := range []model.CatalogSection{inv.Tools, inv.Prompts, inv.Resources, inv.Templates} {
		switch sec.Status {
		case model.SectionFailed, model.SectionIncomplete:
			inv.Incomplete = true
			inv.Reasons = appendReasons(inv.Reasons, sec.Reasons)
		}
	}
	return inv
}

func wantKind(kinds []model.PrimitiveKind, kind model.PrimitiveKind) bool {
	if len(kinds) == 0 {
		return true
	}
	return slices.Contains(kinds, kind)
}

func appendReasons(dst, extra []string) []string {
	for _, reason := range extra {
		if reason == model.ReasonDuplicateItem {
			continue
		}
		if slices.Contains(dst, reason) {
			continue
		}
		dst = append(dst, reason)
	}
	slices.Sort(dst)
	return dst
}
