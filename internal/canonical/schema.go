package canonical

import (
	"bytes"
	jsonv2 "encoding/json/v2"
	"fmt"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Schema re-encodes a JSON Schema document with representation-only noise removed.
// Empty input stays empty. Object keys are sorted at marshal time.
func Schema(raw []byte) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var tree any
	if err := jsonv2.Unmarshal(raw, &tree); err != nil {
		return nil, errclass.Internal{Err: fmt.Errorf("canonical: decode schema: %w", err)}
	}
	out, err := jsonv2.Marshal(normalizeSchema(tree), jsonv2.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: fmt.Errorf("canonical: marshal schema: %w", err)}
	}
	return out, nil
}
