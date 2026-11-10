package diff

import (
	"slices"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func compareType(r *report, path string, from, to any) {
	if equalJSON(from, to) {
		return
	}
	fromSet, fromOK := typeSet(from)
	toSet, toOK := typeSet(to)
	if !fromOK || !toOK {
		r.add(pointerJoin(path, "type"), model.CategoryChange, model.SeverityUnknown, from, to, "type keyword changed in an unclassified form.")
		return
	}
	removed := missingFrom(fromSet, toSet)
	added := missingFrom(toSet, fromSet)
	switch {
	case len(fromSet) == 0 && len(toSet) > 0:
		r.add(pointerJoin(path, "type"), model.CategoryChange, model.SeverityBreaking, from, to, "Type constraint was added to an unconstrained schema.")
	case len(toSet) == 0 && len(fromSet) > 0:
		r.add(pointerJoin(path, "type"), model.CategoryChange, model.SeverityAdditive, from, to, "Type constraint was removed.")
	case len(removed) > 0 && len(added) == 0:
		r.add(pointerJoin(path, "type"), model.CategoryChange, model.SeverityBreaking, from, to, "Type union was narrowed.")
	case len(added) > 0 && len(removed) == 0:
		r.add(pointerJoin(path, "type"), model.CategoryChange, model.SeverityAdditive, from, to, "Type union was widened.")
	default:
		r.add(pointerJoin(path, "type"), model.CategoryChange, model.SeverityBreaking, from, to, "Instance type changed incompatibly.")
	}
}

func typeSet(v any) ([]string, bool) {
	if v == nil {
		return nil, true
	}
	if s, ok := v.(string); ok {
		return []string{s}, true
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	slices.Sort(out)
	return out, true
}

func compareEnum(r *report, path string, from, to any) {
	if equalJSON(from, to) {
		return
	}
	fromSet, fromOK := enumSet(from)
	toSet, toOK := enumSet(to)
	if !fromOK || !toOK {
		r.add(pointerJoin(path, "enum"), model.CategoryChange, model.SeverityUnknown, from, to, "enum keyword changed in an unclassified form.")
		return
	}
	switch {
	case len(fromSet) == 0 && len(toSet) > 0:
		r.add(pointerJoin(path, "enum"), model.CategoryAddition, model.SeverityBreaking, nil, to, "enum constraint was added (value set narrowed).")
		return
	case len(fromSet) > 0 && len(toSet) == 0:
		r.add(pointerJoin(path, "enum"), model.CategoryRemoval, model.SeverityAdditive, from, nil, "enum constraint was removed (value set widened).")
		return
	}
	removed := missingFrom(fromSet, toSet)
	added := missingFrom(toSet, fromSet)
	switch {
	case len(removed) > 0 && len(added) == 0:
		r.add(pointerJoin(path, "enum"), model.CategoryChange, model.SeverityBreaking, from, to, "enum was narrowed.")
	case len(added) > 0 && len(removed) == 0:
		r.add(pointerJoin(path, "enum"), model.CategoryChange, model.SeverityAdditive, from, to, "enum was widened.")
	default:
		r.add(pointerJoin(path, "enum"), model.CategoryChange, model.SeverityBreaking, from, to, "enum membership changed incompatibly.")
	}
}

func enumSet(v any) ([]string, bool) {
	if v == nil {
		return nil, true
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		out = append(out, string(marshalValue(item)))
	}
	slices.Sort(out)
	return out, true
}

func compareConst(r *report, path string, from, to any) {
	if equalJSON(from, to) {
		return
	}
	r.add(pointerJoin(path, "const"), model.CategoryChange, model.SeverityBreaking, from, to, "const value changed.")
}

func missingFrom(have, in []string) []string {
	set := nameSet(in)
	var out []string
	for _, item := range have {
		if _, ok := set[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}
