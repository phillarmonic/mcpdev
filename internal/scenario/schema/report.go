package schema

import (
	jsonv2 "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

// ReportJSON encodes a suite for logs and test reports.
// Secret values cannot appear: Suite only stores from_env names, which are redacted by key shape.
func ReportJSON(s Suite) ([]byte, error) {
	b, err := jsonv2.Marshal(s, jsonv2.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: err}
	}
	var tree any
	if uerr := jsonv2.Unmarshal(b, &tree); uerr != nil {
		return nil, errclass.Internal{Err: uerr}
	}
	out, merr := jsonv2.Marshal(redact.Any(tree, redact.Options{}), jsonv2.Deterministic(true))
	if merr != nil {
		return nil, errclass.Internal{Err: merr}
	}
	return append(out, '\n'), nil
}
