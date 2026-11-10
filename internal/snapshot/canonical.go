package snapshot

import (
	"bytes"
	jsonv2 "encoding/json/v2"
	"fmt"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

// CanonicalJSON re-encodes nested JSON with sorted object keys.
// JSON Schema documents use canonical.Schema instead; this helper is for _meta and other trees.
// Number values are encoded by encoding/json/v2; string URIs are not rewritten.
func CanonicalJSON(raw []byte) ([]byte, error) {
	return marshalTree(raw, false)
}

// CanonicalMeta sorts keys and redacts credential-shaped object keys in _meta.
func CanonicalMeta(raw []byte) ([]byte, error) {
	return marshalTree(raw, true)
}

func marshalTree(raw []byte, redactMeta bool) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var tree any
	if err := jsonv2.Unmarshal(raw, &tree); err != nil {
		return nil, errclass.Internal{Err: fmt.Errorf("snapshot: decode nested JSON: %w", err)}
	}
	if redactMeta {
		tree = redact.Any(tree, redact.Options{})
	}
	out, err := jsonv2.Marshal(tree, jsonv2.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: fmt.Errorf("snapshot: marshal nested JSON: %w", err)}
	}
	return out, nil
}
