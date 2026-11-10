package schema

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/model"
)

var identRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var interpRE = regexp.MustCompile(`\$\{([^}]+)\}`)

var allowedProtocol = map[string]bool{
	"":           true,
	"auto":       true,
	"2025-11-25": true,
	"2026-07-28": true,
}

var allowedInspectKind = map[string]bool{
	"":                          true,
	"all":                       true,
	string(model.KindTools):     true,
	string(model.KindPrompts):   true,
	string(model.KindResources): true,
	string(model.KindTemplates): true,
}

var allowedFailOn = map[string]bool{
	"":         true,
	"metadata": true,
	"additive": true,
	"breaking": true,
}

var allowedErrorCategory = map[string]bool{
	"usage":            true,
	"connection":       true,
	"protocol":         true,
	"assertion":        true,
	"interface_change": true,
	"incomplete":       true,
	"internal":         true,
	"tool_error":       true,
}

func validateSuite(s Suite) []Issue {
	var issues []Issue
	switch {
	case s.SchemaVersion == 0:
		issues = append(issues, Issue{Path: "/schema_version", Message: "required (must be 1)"})
	case s.SchemaVersion > SchemaVersion:
		issues = append(issues, Issue{Path: "/schema_version", Message: fmt.Sprintf(
			"version %d is newer than this mcpdev (%d); upgrade mcpdev", s.SchemaVersion, SchemaVersion)})
	default:
		if s.SchemaVersion != SchemaVersion {
			issues = append(issues, Issue{Path: "/schema_version", Message: fmt.Sprintf(
				"version %d is not supported (current is %d)", s.SchemaVersion, SchemaVersion)})
		}
	}
	if strings.TrimSpace(s.Name) == "" {
		issues = append(issues, Issue{Path: "/name", Message: "required"})
	}
	if !allowedProtocol[s.Protocol] {
		issues = append(issues, Issue{Path: "/protocol", Message: "must be auto, 2025-11-25, or 2026-07-28"})
	}
	issues = append(issues, validateTarget(s.Target)...)
	issues = append(issues, validateDuration("/defaults/timeout", s.Defaults.Timeout)...)
	issues = append(issues, validateDuration("/defaults/connect_timeout", s.Defaults.ConnectTimeout)...)
	issues = append(issues, validateVariables(s.Variables)...)
	issues = append(issues, validateSecrets(s.Secrets)...)
	for i, h := range s.Setup {
		issues = append(issues, validateHook(fmt.Sprintf("/setup/%d", i), h, true)...)
	}
	for i, h := range s.Cleanup {
		issues = append(issues, validateHook(fmt.Sprintf("/cleanup/%d", i), h, false)...)
	}
	if len(s.Cases) == 0 {
		issues = append(issues, Issue{Path: "/cases", Message: "at least one case is required"})
	}
	seenCase := map[string]int{}
	for i, c := range s.Cases {
		base := fmt.Sprintf("/cases/%d", i)
		issues = append(issues, validateCase(base, c, s, seenCase)...)
	}
	return issues
}

func validateTarget(t Target) []Issue {
	var n int
	if strings.TrimSpace(t.Profile) != "" {
		n++
	}
	if len(t.Command) > 0 {
		n++
		for i, a := range t.Command {
			if strings.TrimSpace(a) == "" {
				return []Issue{{Path: fmt.Sprintf("/target/command/%d", i), Message: "argv entries must be non-empty"}}
			}
		}
	}
	if strings.TrimSpace(t.URL) != "" {
		n++
	}
	switch n {
	case 0:
		return []Issue{{Path: "/target", Message: "exactly one of profile, command, or url is required"}}
	case 1:
		return nil
	default:
		return []Issue{{Path: "/target", Message: "exactly one of profile, command, or url is allowed"}}
	}
}

func validateVariables(vars map[string]string) []Issue {
	var issues []Issue
	for k, v := range vars {
		p := joinPointer("/variables", k)
		if !identRE.MatchString(k) {
			issues = append(issues, Issue{Path: p, Message: "name must be a simple identifier"})
		}
		issues = append(issues, checkInterp(p, v, Suite{}, nil, true)...)
	}
	return issues
}

func validateSecrets(secrets map[string]Secret) []Issue {
	var issues []Issue
	for k, s := range secrets {
		p := joinPointer("/secrets", k)
		if !identRE.MatchString(k) {
			issues = append(issues, Issue{Path: p, Message: "name must be a simple identifier"})
		}
		if strings.TrimSpace(s.FromEnv) == "" {
			issues = append(issues, Issue{Path: p + "/from_env", Message: "required environment variable name"})
		}
	}
	return issues
}

func validateHook(path string, h Hook, allowSnapshot bool) []Issue {
	n := 0
	var issues []Issue
	if h.Snapshot != nil {
		n++
		if !allowSnapshot {
			issues = append(issues, Issue{Path: path + "/snapshot", Message: "snapshot hooks belong in setup, not cleanup"})
		}
		if strings.TrimSpace(h.Snapshot.Path) == "" {
			issues = append(issues, Issue{Path: path + "/snapshot/path", Message: "required"})
		}
		if h.Snapshot.As != "" && !identRE.MatchString(h.Snapshot.As) {
			issues = append(issues, Issue{Path: path + "/snapshot/as", Message: "must be a simple identifier"})
		}
	}
	if h.Exec != nil {
		n++
		if len(h.Exec.Argv) == 0 {
			issues = append(issues, Issue{Path: path + "/exec/argv", Message: "required argv array"})
		}
		for i, a := range h.Exec.Argv {
			if strings.TrimSpace(a) == "" {
				issues = append(issues, Issue{Path: fmt.Sprintf("%s/exec/argv/%d", path, i), Message: "argv entries must be non-empty"})
			}
		}
	}
	if n != 1 {
		issues = append(issues, Issue{Path: path, Message: "exactly one of snapshot or exec is required"})
	}
	return issues
}

func validateCase(base string, c Case, s Suite, seen map[string]int) []Issue {
	var issues []Issue
	if strings.TrimSpace(c.Name) == "" {
		issues = append(issues, Issue{Path: base + "/name", Message: "required"})
	} else if prev, ok := seen[c.Name]; ok {
		issues = append(issues, Issue{Path: base + "/name", Message: fmt.Sprintf("duplicate case name %q (also /cases/%d)", c.Name, prev)})
	} else {
		seen[c.Name] = indexFrom(base)
	}
	issues = append(issues, validateDuration(base+"/timeout", c.Timeout)...)
	if len(c.Steps) == 0 {
		issues = append(issues, Issue{Path: base + "/steps", Message: "at least one step is required"})
	}
	captured := map[string]string{}
	seenStep := map[string]int{}
	for i, st := range c.Steps {
		sp := fmt.Sprintf("%s/steps/%d", base, i)
		issues = append(issues, validateStep(sp, st, s, captured, seenStep)...)
	}
	return issues
}

func indexFrom(base string) int {
	_, last, ok := strings.CutLast(base, "/")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
}

func validateStep(path string, st Step, s Suite, captured map[string]string, seenID map[string]int) []Issue {
	var issues []Issue
	if st.ID != "" {
		if !identRE.MatchString(st.ID) {
			issues = append(issues, Issue{Path: path + "/id", Message: "must be a simple identifier"})
		} else if prev, ok := seenID[st.ID]; ok {
			issues = append(issues, Issue{Path: path + "/id", Message: fmt.Sprintf("duplicate step id %q (also index %d)", st.ID, prev)})
		} else {
			seenID[st.ID] = indexFrom(path)
		}
	}
	ops := 0
	if st.Inspect != nil {
		ops++
		if !allowedInspectKind[st.Inspect.Kind] {
			issues = append(issues, Issue{Path: path + "/inspect/kind", Message: "must be tools, prompts, resources, templates, all, or omitted"})
		}
	}
	if st.Call != nil {
		ops++
		if strings.TrimSpace(st.Call.Tool) == "" {
			issues = append(issues, Issue{Path: path + "/call/tool", Message: "required"})
		}
		issues = append(issues, checkInterp(path+"/call/tool", st.Call.Tool, s, captured, false)...)
		issues = append(issues, checkInterp(path+"/call/arguments", string(st.Call.Arguments), s, captured, false)...)
	}
	if st.Prompt != nil {
		ops++
		if strings.TrimSpace(st.Prompt.Name) == "" {
			issues = append(issues, Issue{Path: path + "/prompt/name", Message: "required"})
		}
		for k, v := range st.Prompt.Arguments {
			issues = append(issues, checkInterp(joinPointer(path+"/prompt/arguments", k), v, s, captured, false)...)
		}
	}
	if st.Read != nil {
		ops++
		hasURI := strings.TrimSpace(st.Read.URI) != ""
		hasTpl := strings.TrimSpace(st.Read.Template) != ""
		if hasURI == hasTpl {
			issues = append(issues, Issue{Path: path + "/read", Message: "exactly one of uri or template is required"})
		}
		issues = append(issues, checkInterp(path+"/read/uri", st.Read.URI, s, captured, false)...)
		issues = append(issues, checkInterp(path+"/read/template", st.Read.Template, s, captured, false)...)
		for k, v := range st.Read.Arguments {
			issues = append(issues, checkInterp(joinPointer(path+"/read/arguments", k), v, s, captured, false)...)
		}
	}
	if st.Snapshot != nil {
		ops++
		if strings.TrimSpace(st.Snapshot.Compare) == "" {
			issues = append(issues, Issue{Path: path + "/snapshot/compare", Message: "required baseline path"})
		}
		if !allowedFailOn[st.Snapshot.FailOn] {
			issues = append(issues, Issue{Path: path + "/snapshot/fail_on", Message: "must be metadata, additive, or breaking"})
		}
	}
	if ops != 1 {
		issues = append(issues, Issue{Path: path, Message: "exactly one of inspect, call, prompt, read, or snapshot is required"})
	}
	if st.ExpectFailure != nil {
		if !allowedErrorCategory[st.ExpectFailure.Category] {
			issues = append(issues, Issue{Path: path + "/expect_failure/category", Message: "unknown error category"})
		}
	}
	for i, capture := range st.Capture {
		cp := fmt.Sprintf("%s/capture/%d", path, i)
		if !identRE.MatchString(capture.Name) {
			issues = append(issues, Issue{Path: cp + "/name", Message: "must be a simple identifier"})
		} else if _, exists := s.Variables[capture.Name]; exists {
			issues = append(issues, Issue{Path: cp + "/name", Message: "collides with a suite variable"})
		} else if _, exists := s.Secrets[capture.Name]; exists {
			issues = append(issues, Issue{Path: cp + "/name", Message: "collides with a secret name"})
		} else if prev, ok := captured[capture.Name]; ok {
			issues = append(issues, Issue{Path: cp + "/name", Message: "already captured at " + prev})
		} else {
			captured[capture.Name] = cp
		}
		issues = append(issues, validateJSONPath(cp+"/path", capture.Path)...)
	}
	for i, a := range st.Assert {
		issues = append(issues, validateAssertion(fmt.Sprintf("%s/assert/%d", path, i), a, s, captured)...)
	}
	return issues
}
