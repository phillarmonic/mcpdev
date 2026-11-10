package diff

import (
	"bytes"
	jsonv2 "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/canonical"
	"github.com/phillarmonic/mcpdev/internal/model"
)

type schemaRole int

const (
	roleInput schemaRole = iota
	roleOutput
)

var metadataSchemaKeys = map[string]struct{}{
	"title": {}, "description": {}, "$comment": {}, "default": {}, "examples": {},
}

func compareSchema(r *report, path string, from, to []byte, role schemaRole) {
	fromC, fromErr := canonical.Schema(from)
	toC, toErr := canonical.Schema(to)
	if fromErr != nil || toErr != nil {
		if bytes.Equal(bytes.TrimSpace(from), bytes.TrimSpace(to)) {
			return
		}
		r.add(path, model.CategoryChange, model.SeverityUnknown, jsonValue(from), jsonValue(to),
			"Schema could not be classified; treating the difference as unknown risk.")
		return
	}
	if bytes.Equal(fromC, toC) {
		return
	}
	fromEmpty := len(bytes.TrimSpace(fromC)) == 0
	toEmpty := len(bytes.TrimSpace(toC)) == 0
	switch {
	case fromEmpty && !toEmpty:
		sev := model.SeverityAdditive
		if role == roleOutput {
			sev = model.SeverityUnknown
		}
		r.add(path, model.CategoryAddition, sev, nil, jsonValue(toC), schemaAddRationale(role))
		return
	case !fromEmpty && toEmpty:
		r.add(path, model.CategoryRemoval, model.SeverityBreaking, jsonValue(fromC), nil, schemaRemoveRationale(role))
		return
	}
	var fromV, toV any
	if err := jsonv2.Unmarshal(fromC, &fromV); err != nil {
		r.add(path, model.CategoryChange, model.SeverityUnknown, jsonValue(fromC), jsonValue(toC),
			"Schema difference could not be walked.")
		return
	}
	if err := jsonv2.Unmarshal(toC, &toV); err != nil {
		r.add(path, model.CategoryChange, model.SeverityUnknown, jsonValue(fromC), jsonValue(toC),
			"Schema difference could not be walked.")
		return
	}
	walkSchema(r, path, fromV, toV, role)
}

func schemaAddRationale(role schemaRole) string {
	if role == roleOutput {
		return "Output schema appeared; incompatibility is not classified beyond unknown risk."
	}
	return "Input schema was added."
}

func schemaRemoveRationale(role schemaRole) string {
	if role == roleOutput {
		return "Output schema was removed."
	}
	return "Input schema was removed."
}

func walkSchema(r *report, path string, from, to any, role schemaRole) {
	fromObj, fromIsObj := from.(map[string]any)
	toObj, toIsObj := to.(map[string]any)
	if !fromIsObj || !toIsObj {
		if equalJSON(from, to) {
			return
		}
		r.add(path, model.CategoryChange, model.SeverityUnknown, from, to, "Schema value changed in a form that is not classified.")
		return
	}
	compareType(r, path, fromObj["type"], toObj["type"])
	compareEnum(r, path, fromObj["enum"], toObj["enum"])
	compareConst(r, path, fromObj["const"], toObj["const"])
	compareRequired(r, path, fromObj, toObj, role)
	compareProperties(r, path, fromObj, toObj, role)
	compareAdditional(r, path, fromObj["additionalProperties"], toObj["additionalProperties"], role)
	compareComposition(r, path, "anyOf", fromObj["anyOf"], toObj["anyOf"], role, model.SeverityAdditive, model.SeverityBreaking)
	compareComposition(r, path, "allOf", fromObj["allOf"], toObj["allOf"], role, model.SeverityBreaking, model.SeverityAdditive)
	compareComposition(r, path, "oneOf", fromObj["oneOf"], toObj["oneOf"], role, model.SeverityUnknown, model.SeverityUnknown)
	compareBounds(r, path, fromObj, toObj)
	walkKnownNested(r, path, fromObj, toObj, role)
	compareRemaining(r, path, fromObj, toObj)
}

func walkKnownNested(r *report, path string, from, to map[string]any, role schemaRole) {
	for _, key := range []string{"items", "not", "if", "then", "else", "contains", "propertyNames", "contentSchema"} {
		f, fOk := from[key]
		t, tOk := to[key]
		if !fOk && !tOk {
			continue
		}
		if _, isList := f.([]any); isList || looksLikeList(t) {
			compareComposition(r, path, key, f, t, role, model.SeverityUnknown, model.SeverityUnknown)
			continue
		}
		switch {
		case !fOk:
			r.add(pointerJoin(path, key), model.CategoryAddition, model.SeverityUnknown, nil, t, "Nested schema keyword was added.")
		case !tOk:
			r.add(pointerJoin(path, key), model.CategoryRemoval, model.SeverityBreaking, f, nil, "Nested schema keyword was removed.")
		default:
			walkSchema(r, pointerJoin(path, key), f, t, role)
		}
	}
	for _, key := range []string{"$defs", "definitions", "patternProperties", "dependentSchemas"} {
		compareSchemaMap(r, pointerJoin(path, key), from[key], to[key], role)
	}
}

func looksLikeList(v any) bool {
	_, ok := v.([]any)
	return ok
}
