package diff

import (
	"github.com/phillarmonic/mcpdev/internal/model"
)

func diffTools(r *report, names map[string]string, from, to model.CatalogSection) {
	fromMap := mapTools(from.Tools)
	toMap := mapTools(to.Tools)
	diffKeyed(names, keysOf(fromMap), keysOf(toMap),
		func(key string) {
			r.add(pointer("tools", key), model.CategoryRemoval, model.SeverityBreaking, toolPayload(fromMap[key]), nil, "Tool was removed.")
		},
		func(key string) {
			r.add(pointer("tools", key), model.CategoryAddition, model.SeverityAdditive, nil, toolPayload(toMap[key]), "Tool was added.")
		},
		func(fromKey, toKey string) {
			compareTool(r, toKey, fromMap[fromKey], toMap[toKey], fromKey != toKey)
		},
	)
}

func diffPrompts(r *report, names map[string]string, from, to model.CatalogSection) {
	fromMap := mapPrompts(from.Prompts)
	toMap := mapPrompts(to.Prompts)
	diffKeyed(names, keysOf(fromMap), keysOf(toMap),
		func(key string) {
			r.add(pointer("prompts", key), model.CategoryRemoval, model.SeverityBreaking, promptPayload(fromMap[key]), nil, "Prompt was removed.")
		},
		func(key string) {
			r.add(pointer("prompts", key), model.CategoryAddition, model.SeverityAdditive, nil, promptPayload(toMap[key]), "Prompt was added.")
		},
		func(fromKey, toKey string) {
			comparePrompt(r, toKey, fromMap[fromKey], toMap[toKey], fromKey != toKey)
		},
	)
}

func diffResources(r *report, names map[string]string, from, to model.CatalogSection) {
	fromMap := mapResources(from.Resources)
	toMap := mapResources(to.Resources)
	diffKeyed(names, keysOf(fromMap), keysOf(toMap),
		func(key string) {
			r.add(pointer("resources", key), model.CategoryRemoval, model.SeverityBreaking, resourcePayload(fromMap[key]), nil, "Resource was removed.")
		},
		func(key string) {
			r.add(pointer("resources", key), model.CategoryAddition, model.SeverityAdditive, nil, resourcePayload(toMap[key]), "Resource was added.")
		},
		func(fromKey, toKey string) {
			compareResource(r, toKey, fromMap[fromKey], toMap[toKey], fromKey != toKey)
		},
	)
}

func diffTemplates(r *report, names map[string]string, from, to model.CatalogSection) {
	fromMap := mapTemplates(from.Templates)
	toMap := mapTemplates(to.Templates)
	diffKeyed(names, keysOf(fromMap), keysOf(toMap),
		func(key string) {
			r.add(pointer("templates", key), model.CategoryRemoval, model.SeverityBreaking, templatePayload(fromMap[key]), nil, "Resource template was removed.")
		},
		func(key string) {
			r.add(pointer("templates", key), model.CategoryAddition, model.SeverityAdditive, nil, templatePayload(toMap[key]), "Resource template was added.")
		},
		func(fromKey, toKey string) {
			compareTemplate(r, toKey, fromMap[fromKey], toMap[toKey], fromKey != toKey)
		},
	)
}

func diffKeyed(rename map[string]string, fromKeys, toKeys map[string]struct{}, onRemove, onAdd func(string), onMatch func(fromKey, toKey string)) {
	seenTo := map[string]struct{}{}
	for _, fromKey := range sortedKeys(fromKeys) {
		toKey := fromKey
		if dest, ok := rename[fromKey]; ok {
			toKey = dest
		}
		if _, ok := toKeys[toKey]; ok {
			onMatch(fromKey, toKey)
			seenTo[toKey] = struct{}{}
			continue
		}
		onRemove(fromKey)
	}
	for _, toKey := range sortedKeys(toKeys) {
		if _, ok := seenTo[toKey]; ok {
			continue
		}
		onAdd(toKey)
	}
}

func compareTool(r *report, key string, from, to model.Tool, renamed bool) {
	if renamed {
		r.add(pointer("tools", key, "name"), model.CategoryChange, model.SeverityBreaking, from.Name, to.Name,
			"Mapped rename; remaining fields are compared as the same tool.")
	}
	diffScalar(r, pointer("tools", key, "title"), from.Title, to.Title, model.SeverityMetadata, "Tool title changed.")
	diffScalar(r, pointer("tools", key, "description"), from.Description, to.Description, model.SeverityMetadata, "Tool description changed.")
	compareSchema(r, pointer("tools", key, "input_schema"), from.InputSchema, to.InputSchema, roleInput)
	compareSchema(r, pointer("tools", key, "output_schema"), from.OutputSchema, to.OutputSchema, roleOutput)
	diffJSON(r, pointer("tools", key, "_meta"), from.Meta, to.Meta, model.SeverityMetadata, "Tool _meta changed.")
	diffJSON(r, pointer("tools", key, "icons"), iconValue(from.Icons), iconValue(to.Icons), model.SeverityMetadata, "Tool icons changed.")
	diffJSON(r, pointer("tools", key, "annotations"), from.Annotations, to.Annotations, model.SeverityMetadata, "Tool annotations changed.")
}

func comparePrompt(r *report, key string, from, to model.Prompt, renamed bool) {
	if renamed {
		r.add(pointer("prompts", key, "name"), model.CategoryChange, model.SeverityBreaking, from.Name, to.Name,
			"Mapped rename; remaining fields are compared as the same prompt.")
	}
	diffScalar(r, pointer("prompts", key, "title"), from.Title, to.Title, model.SeverityMetadata, "Prompt title changed.")
	diffScalar(r, pointer("prompts", key, "description"), from.Description, to.Description, model.SeverityMetadata, "Prompt description changed.")
	diffJSON(r, pointer("prompts", key, "_meta"), from.Meta, to.Meta, model.SeverityMetadata, "Prompt _meta changed.")
	diffJSON(r, pointer("prompts", key, "icons"), iconValue(from.Icons), iconValue(to.Icons), model.SeverityMetadata, "Prompt icons changed.")
	diffPromptArgs(r, key, from.Arguments, to.Arguments)
}

func compareResource(r *report, key string, from, to model.Resource, renamed bool) {
	if renamed {
		r.add(pointer("resources", key, "uri"), model.CategoryChange, model.SeverityBreaking, from.URI, to.URI,
			"Mapped rename; remaining fields are compared as the same resource.")
	}
	diffScalar(r, pointer("resources", key, "name"), from.Name, to.Name, model.SeverityMetadata, "Resource name changed.")
	diffScalar(r, pointer("resources", key, "title"), from.Title, to.Title, model.SeverityMetadata, "Resource title changed.")
	diffScalar(r, pointer("resources", key, "description"), from.Description, to.Description, model.SeverityMetadata, "Resource description changed.")
	diffScalar(r, pointer("resources", key, "mime_type"), from.MIMEType, to.MIMEType, model.SeverityMetadata, "Resource MIME type changed.")
	diffJSON(r, pointer("resources", key, "_meta"), from.Meta, to.Meta, model.SeverityMetadata, "Resource _meta changed.")
	diffJSON(r, pointer("resources", key, "icons"), iconValue(from.Icons), iconValue(to.Icons), model.SeverityMetadata, "Resource icons changed.")
	diffJSON(r, pointer("resources", key, "annotations"), from.Annotations, to.Annotations, model.SeverityMetadata, "Resource annotations changed.")
}

func compareTemplate(r *report, key string, from, to model.ResourceTemplate, renamed bool) {
	if renamed {
		r.add(pointer("templates", key, "uri_template"), model.CategoryChange, model.SeverityBreaking, from.URITemplate, to.URITemplate,
			"Mapped rename; remaining fields are compared as the same template.")
	}
	diffScalar(r, pointer("templates", key, "name"), from.Name, to.Name, model.SeverityMetadata, "Template name changed.")
	diffScalar(r, pointer("templates", key, "title"), from.Title, to.Title, model.SeverityMetadata, "Template title changed.")
	diffScalar(r, pointer("templates", key, "description"), from.Description, to.Description, model.SeverityMetadata, "Template description changed.")
	diffScalar(r, pointer("templates", key, "mime_type"), from.MIMEType, to.MIMEType, model.SeverityMetadata, "Template MIME type changed.")
	diffJSON(r, pointer("templates", key, "_meta"), from.Meta, to.Meta, model.SeverityMetadata, "Template _meta changed.")
	diffJSON(r, pointer("templates", key, "icons"), iconValue(from.Icons), iconValue(to.Icons), model.SeverityMetadata, "Template icons changed.")
	diffJSON(r, pointer("templates", key, "annotations"), from.Annotations, to.Annotations, model.SeverityMetadata, "Template annotations changed.")
}

func diffPromptArgs(r *report, prompt string, from, to []model.PromptArgument) {
	fromMap := mapArgs(from)
	toMap := mapArgs(to)
	for _, name := range sortedKeys(keysOf(fromMap)) {
		a := fromMap[name]
		b, ok := toMap[name]
		if !ok {
			r.add(pointer("prompts", prompt, "arguments", name), model.CategoryRemoval, model.SeverityBreaking, a, nil, "Prompt argument was removed.")
			continue
		}
		diffScalar(r, pointer("prompts", prompt, "arguments", name, "title"), a.Title, b.Title, model.SeverityMetadata, "Prompt argument title changed.")
		diffScalar(r, pointer("prompts", prompt, "arguments", name, "description"), a.Description, b.Description, model.SeverityMetadata, "Prompt argument description changed.")
		if a.Required != b.Required {
			sev := model.SeverityAdditive
			cat := model.CategoryChange
			rationale := "Prompt argument is no longer required."
			if b.Required {
				sev = model.SeverityBreaking
				rationale = "Prompt argument became required."
			}
			r.add(pointer("prompts", prompt, "arguments", name, "required"), cat, sev, a.Required, b.Required, rationale)
		}
	}
	for _, name := range sortedKeys(keysOf(toMap)) {
		if _, ok := fromMap[name]; ok {
			continue
		}
		b := toMap[name]
		sev := model.SeverityAdditive
		rationale := "Optional prompt argument was added."
		if b.Required {
			sev = model.SeverityBreaking
			rationale = "Required prompt argument was added."
		}
		r.add(pointer("prompts", prompt, "arguments", name), model.CategoryAddition, sev, nil, b, rationale)
	}
}
