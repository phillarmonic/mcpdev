package diff

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/model"
)

// Engine implements port.Differ.
// Renames, when set, pair catalog identities so they are compared as one item
// instead of a removal plus an addition.
type Engine struct {
	Renames []model.Rename
}

// Diff classifies additions, removals, and field changes between snapshots.
func (e Engine) Diff(ctx context.Context, from, to model.Snapshot) (model.Diff, error) {
	if err := ctx.Err(); err != nil {
		return model.Diff{}, err
	}
	var r report
	indeterminate := unread(from) || unread(to)
	if indeterminate {
		r.add(
			pointer("incomplete"),
			model.CategoryChange,
			model.SeverityUnknown,
			incompletePayload(from),
			incompletePayload(to),
			"One or both snapshots are incomplete, so equality cannot be proven.",
		)
	}
	diffIdentity(&r, from, to)
	diffCapabilities(&r, from.Capabilities, to.Capabilities)
	diffTools(&r, e.renames(model.KindTools), from.Tools, to.Tools)
	diffPrompts(&r, e.renames(model.KindPrompts), from.Prompts, to.Prompts)
	diffResources(&r, e.renames(model.KindResources), from.Resources, to.Resources)
	diffTemplates(&r, e.renames(model.KindTemplates), from.Templates, to.Templates)
	r.sort()
	out := model.Diff{Changes: r.changes, Verdict: model.VerdictEqual}
	if out.Changes == nil {
		out.Changes = []model.Change{}
	}
	switch {
	case indeterminate:
		out.Verdict = model.VerdictIndeterminate
	case len(out.Changes) > 0:
		out.Verdict = model.VerdictChanged
		out.Changed = true
	}
	return out, nil
}

func (e Engine) renames(kind model.PrimitiveKind) map[string]string {
	out := make(map[string]string)
	for _, item := range e.Renames {
		if item.Kind != kind || item.From == "" || item.To == "" {
			continue
		}
		out[item.From] = item.To
	}
	return out
}

func unread(snap model.Snapshot) bool {
	if snap.Incomplete {
		return true
	}
	return unreadSection(snap.Tools) || unreadSection(snap.Prompts) ||
		unreadSection(snap.Resources) || unreadSection(snap.Templates)
}

func unreadSection(sec model.CatalogSection) bool {
	return sec.Status == model.SectionFailed || sec.Status == model.SectionIncomplete
}

func incompletePayload(snap model.Snapshot) map[string]any {
	return map[string]any{
		"incomplete":         snap.Incomplete,
		"incomplete_reasons": cloneStrings(snap.IncompleteReasons),
		"tools":              string(snap.Tools.Status),
		"prompts":            string(snap.Prompts.Status),
		"resources":          string(snap.Resources.Status),
		"templates":          string(snap.Templates.Status),
	}
}

func cloneStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
