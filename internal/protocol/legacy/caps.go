package legacy

import (
	"fmt"
	"maps"
	"slices"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// DecodeCaps converts an initialize-result capabilities value into model flags.
// A missing object is empty capabilities. A non-object is a protocol error.
func DecodeCaps(v any) (model.ServerCapabilities, error) {
	if v == nil {
		return model.ServerCapabilities{}, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return model.ServerCapabilities{}, errclass.Protocol{
			Err: fmt.Errorf("protocol: capabilities must be an object, got %T", v),
		}
	}
	out := model.ServerCapabilities{}
	tools, hasTools, err := featureObject(m, "tools")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	if hasTools {
		out.Tools = true
		out.ToolsListChanged, err = boolField(tools, "listChanged")
		if err != nil {
			return model.ServerCapabilities{}, err
		}
	}
	prompts, hasPrompts, err := featureObject(m, "prompts")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	if hasPrompts {
		out.Prompts = true
		out.PromptsListChanged, err = boolField(prompts, "listChanged")
		if err != nil {
			return model.ServerCapabilities{}, err
		}
	}
	resources, hasResources, err := featureObject(m, "resources")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	if hasResources {
		out.Resources = true
		out.ResourcesListChanged, err = boolField(resources, "listChanged")
		if err != nil {
			return model.ServerCapabilities{}, err
		}
		out.ResourceSubscribe, err = boolField(resources, "subscribe")
		if err != nil {
			return model.ServerCapabilities{}, err
		}
	}
	_, hasCompletions, err := featureObject(m, "completions")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	out.Completions = hasCompletions
	_, hasLogging, err := featureObject(m, "logging")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	out.Logging = hasLogging
	exp, err := optionalStringMap(m, "experimental")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	ext, err := optionalStringMap(m, "extensions")
	if err != nil {
		return model.ServerCapabilities{}, err
	}
	out.Experimental = exp
	out.Extensions = ext
	return out, nil
}

func optionalStringMap(m map[string]any, key string) ([]string, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return nil, nil
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, errclass.Protocol{
			Err: fmt.Errorf("protocol: capabilities.%s must be an object, got %T", key, v),
		}
	}
	keys := slices.Collect(maps.Keys(obj))
	slices.Sort(keys)
	return keys, nil
}

func featureObject(m map[string]any, key string) (map[string]any, bool, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return map[string]any{}, false, nil
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false, errclass.Protocol{
			Err: fmt.Errorf("protocol: capabilities.%s must be an object, got %T", key, v),
		}
	}
	return obj, true, nil
}

func boolField(m map[string]any, key string) (bool, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, errclass.Protocol{
			Err: fmt.Errorf("protocol: capabilities field %s must be a boolean, got %T", key, v),
		}
	}
	return b, nil
}
