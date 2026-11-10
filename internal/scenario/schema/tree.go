package schema

import (
	"strconv"
	"strings"
)

var forbiddenKeys = map[string]string{
	"shell":      "arbitrary shell execution is not allowed",
	"sh":         "arbitrary shell execution is not allowed",
	"bash":       "arbitrary shell execution is not allowed",
	"zsh":        "arbitrary shell execution is not allowed",
	"fish":       "arbitrary shell execution is not allowed",
	"powershell": "arbitrary shell execution is not allowed",
	"pwsh":       "arbitrary shell execution is not allowed",
	"cmd":        "arbitrary shell execution is not allowed",
	"script":     "arbitrary shell execution is not allowed",
	"run":        "use inspect, call, prompt, read, or snapshot; run is not a step kind",
	"system":     "arbitrary process strings are not allowed",
	"entrypoint": "arbitrary process strings are not allowed",
	"exec_shell": "arbitrary shell execution is not allowed",
}

var allowedSuite = map[string]bool{
	"schema_version": true,
	"name":           true,
	"description":    true,
	"target":         true,
	"protocol":       true,
	"defaults":       true,
	"variables":      true,
	"secrets":        true,
	"setup":          true,
	"cleanup":        true,
	"cases":          true,
}

var allowedTarget = map[string]bool{"profile": true, "command": true, "url": true}

var allowedDefaults = map[string]bool{"timeout": true, "connect_timeout": true}

var allowedSecret = map[string]bool{"from_env": true}

var allowedHook = map[string]bool{"snapshot": true, "exec": true}

var allowedHookSnapshot = map[string]bool{"path": true, "as": true}

var allowedHookExec = map[string]bool{"argv": true, "cwd": true}

var allowedCase = map[string]bool{"name": true, "timeout": true, "steps": true}

var allowedStep = map[string]bool{
	"id": true, "inspect": true, "call": true, "prompt": true, "read": true,
	"snapshot": true, "expect_failure": true, "capture": true, "assert": true,
}

var allowedInspect = map[string]bool{"kind": true}

var allowedCall = map[string]bool{"tool": true, "arguments": true}

var allowedPrompt = map[string]bool{"name": true, "arguments": true}

var allowedRead = map[string]bool{"uri": true, "template": true, "arguments": true}

var allowedSnapshot = map[string]bool{"compare": true, "fail_on": true, "ignore": true, "rename": true}

var allowedExpect = map[string]bool{"category": true}

var allowedCapture = map[string]bool{"name": true, "path": true}

var allowedAssert = map[string]bool{
	"equal": true, "contains": true, "json_path": true, "schema": true,
	"content_type": true, "latency": true, "error_category": true,
}

var allowedEqual = map[string]bool{"path": true, "value": true, "variable": true}

var allowedContains = map[string]bool{"path": true, "value": true}

var allowedJSONPath = map[string]bool{"path": true, "exists": true, "count": true}

var allowedSchema = map[string]bool{"path": true, "schema": true, "file": true}

var allowedContentType = map[string]bool{"equals": true}

var allowedLatency = map[string]bool{"max": true}

var allowedErrorCat = map[string]bool{"equals": true}

func walkForbidden(path string, v any, issues *[]Issue) {
	switch t := v.(type) {
	case map[string]any:
		if opaquePayload(path) {
			return
		}
		for k, val := range t {
			child := joinPointer(path, k)
			if msg, bad := forbiddenKeys[k]; bad {
				*issues = append(*issues, Issue{Path: child, Message: msg})
			}
			if k == "command" {
				if path != "/target" {
					*issues = append(*issues, Issue{Path: child, Message: "command is only allowed on target as an argv array"})
				} else if _, isArr := val.([]any); !isArr && val != nil {
					*issues = append(*issues, Issue{Path: child, Message: "must be an argv array, not a shell string"})
				}
			}
			walkForbidden(child, val, issues)
		}
	case []any:
		if opaquePayload(path) {
			return
		}
		for i, val := range t {
			walkForbidden(joinPointer(path, strconv.Itoa(i)), val, issues)
		}
	}
}

// opaquePayload is MCP JSON the suite must not interpret as mcpdev keys.
// tools/call arguments, prompt/read argument maps, and assertion literals may
// legally contain names such as command or shell.
func opaquePayload(path string) bool {
	for _, suffix := range []string{
		"/call/arguments",
		"/prompt/arguments",
		"/read/arguments",
		"/equal/value",
		"/contains/value",
		"/schema/schema",
	} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func walkUnknown(path string, v any, allowed map[string]bool, issues *[]Issue) {
	obj, ok := v.(map[string]any)
	if !ok {
		if path == "" || path == "/" {
			*issues = append(*issues, Issue{Path: "/", Message: "document must be an object"})
		}
		return
	}
	for k, val := range obj {
		if forbiddenKeys[k] != "" {
			continue
		}
		if !allowed[k] {
			*issues = append(*issues, Issue{Path: joinPointer(path, k), Message: "unknown field"})
			continue
		}
		descendKnown(joinPointer(path, k), k, val, issues)
	}
}

func descendKnown(path, key string, val any, issues *[]Issue) {
	switch key {
	case "target":
		walkUnknown(path, val, allowedTarget, issues)
	case "defaults":
		walkUnknown(path, val, allowedDefaults, issues)
	case "secrets":
		walkMapObjects(path, val, allowedSecret, issues)
	case "setup", "cleanup":
		walkList(path, val, func(p string, item any) {
			walkUnknown(p, item, allowedHook, issues)
			hookDescend(p, item, issues)
		}, issues)
	case "cases":
		walkList(path, val, func(p string, item any) {
			walkUnknown(p, item, allowedCase, issues)
			caseDescend(p, item, issues)
		}, issues)
	}
}

func hookDescend(path string, item any, issues *[]Issue) {
	obj, _ := item.(map[string]any)
	if snap, ok := obj["snapshot"]; ok {
		walkUnknown(joinPointer(path, "snapshot"), snap, allowedHookSnapshot, issues)
	}
	if ex, ok := obj["exec"]; ok {
		walkUnknown(joinPointer(path, "exec"), ex, allowedHookExec, issues)
	}
}

func caseDescend(path string, item any, issues *[]Issue) {
	obj, _ := item.(map[string]any)
	steps, ok := obj["steps"]
	if !ok {
		return
	}
	walkList(joinPointer(path, "steps"), steps, func(p string, step any) {
		walkUnknown(p, step, allowedStep, issues)
		stepDescend(p, step, issues)
	}, issues)
}

func stepDescend(path string, item any, issues *[]Issue) {
	obj, _ := item.(map[string]any)
	if v, ok := obj["inspect"]; ok {
		walkUnknown(joinPointer(path, "inspect"), v, allowedInspect, issues)
	}
	if v, ok := obj["call"]; ok {
		walkUnknown(joinPointer(path, "call"), v, allowedCall, issues)
	}
	if v, ok := obj["prompt"]; ok {
		walkUnknown(joinPointer(path, "prompt"), v, allowedPrompt, issues)
	}
	if v, ok := obj["read"]; ok {
		walkUnknown(joinPointer(path, "read"), v, allowedRead, issues)
	}
	if v, ok := obj["snapshot"]; ok {
		walkUnknown(joinPointer(path, "snapshot"), v, allowedSnapshot, issues)
	}
	if v, ok := obj["expect_failure"]; ok {
		walkUnknown(joinPointer(path, "expect_failure"), v, allowedExpect, issues)
	}
	if v, ok := obj["capture"]; ok {
		walkList(joinPointer(path, "capture"), v, func(p string, item any) {
			walkUnknown(p, item, allowedCapture, issues)
		}, issues)
	}
	if v, ok := obj["assert"]; ok {
		walkList(joinPointer(path, "assert"), v, func(p string, a any) {
			walkUnknown(p, a, allowedAssert, issues)
			assertDescend(p, a, issues)
		}, issues)
	}
}

func assertDescend(path string, item any, issues *[]Issue) {
	obj, _ := item.(map[string]any)
	if v, ok := obj["equal"]; ok {
		walkUnknown(joinPointer(path, "equal"), v, allowedEqual, issues)
	}
	if v, ok := obj["contains"]; ok {
		walkUnknown(joinPointer(path, "contains"), v, allowedContains, issues)
	}
	if v, ok := obj["json_path"]; ok {
		walkUnknown(joinPointer(path, "json_path"), v, allowedJSONPath, issues)
	}
	if v, ok := obj["schema"]; ok {
		walkUnknown(joinPointer(path, "schema"), v, allowedSchema, issues)
	}
	if v, ok := obj["content_type"]; ok {
		walkUnknown(joinPointer(path, "content_type"), v, allowedContentType, issues)
	}
	if v, ok := obj["latency"]; ok {
		walkUnknown(joinPointer(path, "latency"), v, allowedLatency, issues)
	}
	if v, ok := obj["error_category"]; ok {
		walkUnknown(joinPointer(path, "error_category"), v, allowedErrorCat, issues)
	}
}

func walkList(path string, v any, each func(string, any), issues *[]Issue) {
	arr, ok := v.([]any)
	if !ok {
		*issues = append(*issues, Issue{Path: path, Message: "must be an array"})
		return
	}
	for i, item := range arr {
		each(joinPointer(path, strconv.Itoa(i)), item)
	}
}

func walkMapObjects(path string, v any, allowed map[string]bool, issues *[]Issue) {
	obj, ok := v.(map[string]any)
	if !ok {
		*issues = append(*issues, Issue{Path: path, Message: "must be an object"})
		return
	}
	for k, val := range obj {
		walkUnknown(joinPointer(path, k), val, allowed, issues)
	}
}
