package schema

import _ "embed"

// FormalJSON is the published JSON Schema for suite documents (schema_version 1).
//
//go:embed suite.v1.json
var FormalJSON []byte
