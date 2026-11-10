package assert

import json "encoding/json/v2"

// Status is one assertion outcome. Skip means the check could not be evaluated.
const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusSkip = "skip"
)

// Result is a typed assertion outcome with expected/actual evidence.
type Result struct {
	Expected any    `json:"expected,omitzero"`
	Actual   any    `json:"actual,omitzero"`
	Kind     string `json:"kind"`
	Path     string `json:"path,omitzero"`
	Status   string `json:"status"`
	Message  string `json:"message,omitzero"`
	Diff     string `json:"diff,omitzero"`
}

func compact(v any) string {
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return ""
	}
	return string(b)
}

func equalJSON(a, b any) bool {
	return compact(a) == compact(b)
}
