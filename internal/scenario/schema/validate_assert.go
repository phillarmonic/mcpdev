package schema

import (
	"encoding/json"
	"strings"
	"time"
)

func validateAssertion(path string, a Assertion, s Suite, captured map[string]string) []Issue {
	n := 0
	var issues []Issue
	if a.Equal != nil {
		n++
		if strings.TrimSpace(a.Equal.Path) == "" {
			issues = append(issues, Issue{Path: path + "/equal/path", Message: "required"})
		}
		if a.Equal.Path != "" {
			issues = append(issues, validateJSONPath(path+"/equal/path", a.Equal.Path)...)
		}
		hasVal := a.Equal.Value != nil
		hasVar := a.Equal.Variable != ""
		if hasVal == hasVar {
			issues = append(issues, Issue{Path: path + "/equal", Message: "exactly one of value or variable is required"})
		}
		if hasVar {
			issues = append(issues, lookupVar(path+"/equal/variable", a.Equal.Variable, s, captured)...)
		}
	}
	if a.Contains != nil {
		n++
		issues = append(issues, validateJSONPath(path+"/contains/path", a.Contains.Path)...)
		if a.Contains.Value == nil {
			issues = append(issues, Issue{Path: path + "/contains/value", Message: "required"})
		}
	}
	if a.JSONPath != nil {
		n++
		issues = append(issues, validateJSONPath(path+"/json_path/path", a.JSONPath.Path)...)
		if a.JSONPath.Exists == nil && a.JSONPath.Count == nil {
			issues = append(issues, Issue{Path: path + "/json_path", Message: "exists or count is required"})
		}
	}
	if a.Schema != nil {
		n++
		hasInline := len(bytesTrim(a.Schema.Schema)) > 0
		hasFile := strings.TrimSpace(a.Schema.File) != ""
		if hasInline == hasFile {
			issues = append(issues, Issue{Path: path + "/schema", Message: "exactly one of schema or file is required"})
		}
		if a.Schema.Path != "" {
			issues = append(issues, validateJSONPath(path+"/schema/path", a.Schema.Path)...)
		}
	}
	if a.ContentType != nil {
		n++
		if strings.TrimSpace(a.ContentType.Equals) == "" {
			issues = append(issues, Issue{Path: path + "/content_type/equals", Message: "required"})
		}
	}
	if a.Latency != nil {
		n++
		issues = append(issues, validateDuration(path+"/latency/max", a.Latency.Max)...)
		if strings.TrimSpace(a.Latency.Max) == "" {
			issues = append(issues, Issue{Path: path + "/latency/max", Message: "required"})
		}
	}
	if a.ErrorCategory != nil {
		n++
		if !allowedErrorCategory[a.ErrorCategory.Equals] {
			issues = append(issues, Issue{Path: path + "/error_category/equals", Message: "unknown error category"})
		}
	}
	if n != 1 {
		issues = append(issues, Issue{Path: path, Message: "exactly one assertion kind is required"})
	}
	issues = append(issues, validateAssertionInterp(path, a, s, captured)...)
	return issues
}

// validateAssertionInterp checks that references inside interpolated assertion
// value fields resolve, mirroring the load-time checks step arguments receive.
func validateAssertionInterp(path string, a Assertion, s Suite, captured map[string]string) []Issue {
	var issues []Issue
	if a.Equal != nil {
		issues = append(issues, checkInterpValue(path+"/equal/value", a.Equal.Value, s, captured)...)
	}
	if a.Contains != nil {
		issues = append(issues, checkInterpValue(path+"/contains/value", a.Contains.Value, s, captured)...)
	}
	if a.Schema != nil && len(bytesTrim(a.Schema.Schema)) > 0 {
		issues = append(issues, checkInterpJSONValues(path+"/schema/schema", a.Schema.Schema, s, captured)...)
	}
	return issues
}

func checkInterpValue(path string, v any, s Suite, captured map[string]string) []Issue {
	switch t := v.(type) {
	case string:
		return checkInterp(path, t, s, captured, false)
	case map[string]any:
		var issues []Issue
		for _, val := range t {
			issues = append(issues, checkInterpValue(path, val, s, captured)...)
		}
		return issues
	case []any:
		var issues []Issue
		for _, val := range t {
			issues = append(issues, checkInterpValue(path, val, s, captured)...)
		}
		return issues
	default:
		return nil
	}
}

func checkInterpJSONValues(path string, raw []byte, s Suite, captured map[string]string) []Issue {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return checkInterpValue(path, v, s, captured)
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func validateDuration(path, s string) []Issue {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return []Issue{{Path: path, Message: "must be a Go duration such as 10s"}}
	}
	if d < 0 {
		return []Issue{{Path: path, Message: "must be non-negative"}}
	}
	return nil
}

func validateJSONPath(path, expr string) []Issue {
	if strings.TrimSpace(expr) == "" {
		return []Issue{{Path: path, Message: "required JSON Path"}}
	}
	if !strings.HasPrefix(expr, "$") && !strings.HasPrefix(expr, "/") {
		return []Issue{{Path: path, Message: "must start with $ (JSON Path) or / (JSON Pointer)"}}
	}
	return nil
}

func lookupVar(path, name string, s Suite, captured map[string]string) []Issue {
	if _, ok := s.Variables[name]; ok {
		return nil
	}
	if _, ok := captured[name]; ok {
		return nil
	}
	return []Issue{{Path: path, Message: "unknown variable or capture " + name}}
}

func checkInterp(path, text string, s Suite, captured map[string]string, varsOnly bool) []Issue {
	if text == "" {
		return nil
	}
	var issues []Issue
	for _, m := range interpRE.FindAllStringSubmatch(text, -1) {
		ref := m[1]
		switch {
		case strings.HasPrefix(ref, "secrets."):
			if varsOnly {
				issues = append(issues, Issue{Path: path, Message: "suite variables cannot interpolate secrets"})
				continue
			}
			name := strings.TrimPrefix(ref, "secrets.")
			if !identRE.MatchString(name) {
				issues = append(issues, Issue{Path: path, Message: "invalid secret reference ${" + ref + "}"})
				continue
			}
			if _, ok := s.Secrets[name]; !ok {
				issues = append(issues, Issue{Path: path, Message: "unknown secret " + name + " (missing-value at run time would fail closed)"})
			}
		case identRE.MatchString(ref):
			if varsOnly {
				issues = append(issues, Issue{Path: path, Message: "suite variables cannot interpolate other interpolations"})
				continue
			}
			if _, ok := s.Variables[ref]; ok {
				continue
			}
			if _, ok := captured[ref]; ok {
				continue
			}
			issues = append(issues, Issue{Path: path, Message: "unknown variable or capture ${" + ref + "} (missing-value errors are explicit)"})
		default:
			issues = append(issues, Issue{Path: path, Message: "invalid interpolation ${" + ref + "}"})
		}
	}
	return issues
}
