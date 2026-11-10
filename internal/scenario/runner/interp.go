package runner

import (
	"bytes"
	"fmt"
	"regexp"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/scenario/schema"
)

var interpRE = regexp.MustCompile(`\$\{([^}]+)\}`)

type bindings struct {
	vars    map[string]string
	secrets map[string]string
	capture map[string]any
}

func (b *bindings) lookup(name string) (string, error) {
	key := name
	secret := false
	if len(name) > 8 && name[:8] == "secrets." {
		secret = true
		key = name[8:]
	}
	if secret {
		v, ok := b.secrets[key]
		if !ok {
			return "", errclass.Usage{Msg: "missing secret " + key}
		}
		return v, nil
	}
	if v, ok := b.vars[key]; ok {
		return v, nil
	}
	if v, ok := b.capture[key]; ok {
		return fmt.Sprint(v), nil
	}
	return "", errclass.Usage{Msg: "unknown interpolation ${" + name + "}"}
}

func (b *bindings) interpString(s string) (string, error) {
	var first error
	out := interpRE.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		v, err := b.lookup(name)
		if err != nil && first == nil {
			first = err
		}
		return v
	})
	return out, first
}

func (b *bindings) interpValue(v any) (any, error) {
	switch t := v.(type) {
	case string:
		return b.interpString(t)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			nv, err := b.interpValue(val)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			nv, err := b.interpValue(val)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

func (b *bindings) interpRaw(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, errclass.Usage{Msg: "arguments JSON", Err: err}
	}
	nv, err := b.interpValue(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(nv, json.Deterministic(true))
}

// interpAssertion resolves ${variable}/${capture}/${secrets.name} inside the
// value-bearing fields of one assertion. Paths and closed-vocabulary fields
// stay literal. equal.value, contains.value, and the string values inside an
// inline schema are interpolated exactly like step arguments.
func (b *bindings) interpAssertion(a schema.Assertion) (schema.Assertion, error) {
	out := a
	if a.Equal != nil {
		v, err := b.interpValue(a.Equal.Value)
		if err != nil {
			return schema.Assertion{}, err
		}
		e := *a.Equal
		e.Value = v
		out.Equal = &e
	}
	if a.Contains != nil {
		v, err := b.interpValue(a.Contains.Value)
		if err != nil {
			return schema.Assertion{}, err
		}
		c := *a.Contains
		c.Value = v
		out.Contains = &c
	}
	if a.Schema != nil {
		s := *a.Schema
		if len(bytes.TrimSpace(s.Schema)) > 0 {
			var v any
			if err := json.Unmarshal(s.Schema, &v); err == nil {
				nv, ierr := b.interpValue(v)
				if ierr != nil {
					return schema.Assertion{}, ierr
				}
				body, merr := json.Marshal(nv, json.Deterministic(true))
				if merr != nil {
					return schema.Assertion{}, errclass.Internal{Err: merr}
				}
				s.Schema = body
			}
			// A malformed schema is left untouched for the schema assertion to report.
		}
		out.Schema = &s
	}
	return out, nil
}

func (b *bindings) interpMap(m map[string]string) (map[string]string, error) {
	if m == nil {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		s, err := b.interpString(v)
		if err != nil {
			return nil, err
		}
		out[k] = s
	}
	return out, nil
}
