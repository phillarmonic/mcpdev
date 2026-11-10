package diff

import (
	jsonv2 "encoding/json/v2"
	"strconv"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func compareRequired(r *report, path string, from, to map[string]any, role schemaRole) {
	fromReq := stringSet(from["required"])
	toReq := stringSet(to["required"])
	fromProps := objectKeys(from["properties"])
	toProps := objectKeys(to["properties"])
	for _, name := range sortedKeys(toReq) {
		if _, existed := fromReq[name]; existed {
			continue
		}
		_, newProp := toProps[name]
		_, oldProp := fromProps[name]
		if newProp && !oldProp {
			continue // classified with the new property
		}
		sev := model.SeverityBreaking
		rationale := "Required input was added."
		if role == roleOutput {
			sev = model.SeverityAdditive
			rationale = "Required output property was added."
		}
		r.add(pointerJoin(path, "required")+"/"+escapePointer(name), model.CategoryAddition, sev, nil, name, rationale)
	}
	for _, name := range sortedKeys(fromReq) {
		if _, ok := toReq[name]; ok {
			continue
		}
		sev := model.SeverityAdditive
		rationale := "Required input was removed."
		if role == roleOutput {
			sev = model.SeverityBreaking
			rationale = "Required output property was removed."
		}
		r.add(pointerJoin(path, "required")+"/"+escapePointer(name), model.CategoryRemoval, sev, name, nil, rationale)
	}
}

func compareProperties(r *report, path string, from, to map[string]any, role schemaRole) {
	fromProps := asObject(from["properties"])
	toProps := asObject(to["properties"])
	fromReq := stringSet(from["required"])
	toReq := stringSet(to["required"])
	base := pointerJoin(path, "properties")
	for _, name := range sortedKeys(keysOf(fromProps)) {
		before := fromProps[name]
		after, ok := toProps[name]
		if !ok {
			r.add(pointerJoin(base, name), model.CategoryRemoval, model.SeverityBreaking, before, nil, propertyRemoveRationale(role))
			continue
		}
		walkSchema(r, pointerJoin(base, name), before, after, role)
	}
	for _, name := range sortedKeys(keysOf(toProps)) {
		if _, ok := fromProps[name]; ok {
			continue
		}
		after := toProps[name]
		_, required := toReq[name]
		_, wasRequired := fromReq[name]
		sev := model.SeverityAdditive
		rationale := "Optional property was added."
		cat := model.CategoryAddition
		if required && !wasRequired && role == roleInput {
			sev = model.SeverityBreaking
			rationale = "Required input property was added."
		}
		if required && role == roleOutput {
			rationale = "Required output property was added."
		}
		r.add(pointerJoin(base, name), cat, sev, nil, after, rationale)
	}
}

func propertyRemoveRationale(role schemaRole) string {
	if role == roleOutput {
		return "Output property was removed."
	}
	return "Input property was removed."
}

func compareAdditional(r *report, path string, from, to any, role schemaRole) {
	if equalJSON(from, to) {
		return
	}
	fromB, fromBool := from.(bool)
	toB, toBool := to.(bool)
	switch {
	case from == nil || to == nil:
		r.add(pointerJoin(path, "additionalProperties"), model.CategoryChange, model.SeverityUnknown, from, to,
			"additionalProperties presence changed; missing vs true vs false stay distinct.")
	case fromBool && toBool && fromB && !toB:
		r.add(pointerJoin(path, "additionalProperties"), model.CategoryChange, model.SeverityBreaking, from, to,
			"additionalProperties was tightened to false.")
	case fromBool && toBool && !fromB && toB:
		r.add(pointerJoin(path, "additionalProperties"), model.CategoryChange, model.SeverityAdditive, from, to,
			"additionalProperties was relaxed to true.")
	default:
		walkSchema(r, pointerJoin(path, "additionalProperties"), from, to, role)
	}
}

func compareComposition(r *report, path, key string, from, to any, role schemaRole, addSev, removeSev model.DiffSeverity) {
	if equalJSON(from, to) {
		return
	}
	fromArr, fromOK := from.([]any)
	toArr, toOK := to.([]any)
	if !fromOK || !toOK {
		if from == nil && toOK {
			r.add(pointerJoin(path, key), model.CategoryAddition, addSev, nil, to, key+" combinator was added.")
			return
		}
		if to == nil && fromOK {
			r.add(pointerJoin(path, key), model.CategoryRemoval, removeSev, from, nil, key+" combinator was removed.")
			return
		}
		r.add(pointerJoin(path, key), model.CategoryChange, model.SeverityUnknown, from, to, key+" changed in an unclassified form.")
		return
	}
	n := min(len(fromArr), len(toArr))
	for i := range n {
		walkSchema(r, pointerJoin(path, key)+"/"+strconv.Itoa(i), fromArr[i], toArr[i], role)
	}
	for i, branch := range toArr[n:] {
		r.add(pointerJoin(path, key)+"/"+strconv.Itoa(n+i), model.CategoryAddition, addSev, nil, branch, key+" branch was added.")
	}
	for i, branch := range fromArr[n:] {
		r.add(pointerJoin(path, key)+"/"+strconv.Itoa(n+i), model.CategoryRemoval, removeSev, branch, nil, key+" branch was removed.")
	}
}

func compareSchemaMap(r *report, path string, from, to any, role schemaRole) {
	if equalJSON(from, to) {
		return
	}
	fromObj := asObject(from)
	toObj := asObject(to)
	for _, name := range sortedKeys(keysOf(fromObj)) {
		after, ok := toObj[name]
		if !ok {
			r.add(pointerJoin(path, name), model.CategoryRemoval, model.SeverityUnknown, fromObj[name], nil, "Named nested schema was removed.")
			continue
		}
		walkSchema(r, pointerJoin(path, name), fromObj[name], after, role)
	}
	for _, name := range sortedKeys(keysOf(toObj)) {
		if _, ok := fromObj[name]; ok {
			continue
		}
		r.add(pointerJoin(path, name), model.CategoryAddition, model.SeverityUnknown, nil, toObj[name], "Named nested schema was added.")
	}
}

func compareBounds(r *report, path string, from, to map[string]any) {
	for _, key := range []string{"minimum", "exclusiveMinimum", "minLength", "minItems", "minProperties", "multipleOf"} {
		compareBound(r, path, key, from[key], to[key], true)
	}
	for _, key := range []string{"maximum", "exclusiveMaximum", "maxLength", "maxItems", "maxProperties"} {
		compareBound(r, path, key, from[key], to[key], false)
	}
}

func compareBound(r *report, path, key string, from, to any, lower bool) {
	if equalJSON(from, to) {
		return
	}
	fromN, fromOK := asFloat(from)
	toN, toOK := asFloat(to)
	if !fromOK || !toOK {
		r.add(pointerJoin(path, key), model.CategoryChange, model.SeverityUnknown, from, to, key+" changed in an unclassified form.")
		return
	}
	narrowed := (lower && toN > fromN) || (!lower && toN < fromN)
	sev := model.SeverityAdditive
	rationale := key + " was relaxed."
	if narrowed {
		sev = model.SeverityBreaking
		rationale = key + " was tightened."
	}
	r.add(pointerJoin(path, key), model.CategoryChange, sev, from, to, rationale)
}

func compareRemaining(r *report, path string, from, to map[string]any) {
	handled := map[string]struct{}{
		"type": {}, "enum": {}, "const": {}, "required": {}, "properties": {},
		"additionalProperties": {}, "anyOf": {}, "allOf": {}, "oneOf": {},
		"items": {}, "not": {}, "if": {}, "then": {}, "else": {}, "contains": {},
		"propertyNames": {}, "contentSchema": {}, "$defs": {}, "definitions": {},
		"patternProperties": {}, "dependentSchemas": {},
		"minimum": {}, "exclusiveMinimum": {}, "maximum": {}, "exclusiveMaximum": {},
		"minLength": {}, "maxLength": {}, "minItems": {}, "maxItems": {},
		"minProperties": {}, "maxProperties": {}, "multipleOf": {},
	}
	keys := map[string]struct{}{}
	for key := range from {
		keys[key] = struct{}{}
	}
	for key := range to {
		keys[key] = struct{}{}
	}
	for _, key := range sortedKeys(keys) {
		if _, ok := handled[key]; ok {
			continue
		}
		f, fOk := from[key]
		t, tOk := to[key]
		if equalJSON(f, t) {
			continue
		}
		sev := model.SeverityUnknown
		rationale := "Schema keyword " + key + " changed with unknown compatibility impact."
		if _, meta := metadataSchemaKeys[key]; meta {
			sev = model.SeverityMetadata
			rationale = "Schema annotation " + key + " changed."
		}
		switch {
		case !fOk:
			r.add(pointerJoin(path, key), model.CategoryAddition, sev, nil, t, rationale)
		case !tOk:
			if sev == model.SeverityUnknown {
				sev = model.SeverityUnknown
			}
			r.add(pointerJoin(path, key), model.CategoryRemoval, sev, f, nil, rationale)
		default:
			r.add(pointerJoin(path, key), model.CategoryChange, sev, f, t, rationale)
		}
	}
}

func stringSet(v any) map[string]struct{} {
	arr, ok := v.([]any)
	if !ok {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			continue
		}
		out[s] = struct{}{}
	}
	return out
}

func objectKeys(v any) map[string]struct{} {
	return keysOf(asObject(v))
}

func asObject(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func equalJSON(a, b any) bool {
	left, err := jsonv2.Marshal(a, jsonv2.Deterministic(true))
	if err != nil {
		return false
	}
	right, err := jsonv2.Marshal(b, jsonv2.Deterministic(true))
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
