package snapshot

import (
	jsonv2 "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/canonical"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// Digests are hex SHA-256 values of canonical semantic encodings.
type Digests struct {
	Snapshot  string
	Tools     string
	Prompts   string
	Resources string
	Templates string
}

// SemanticDigests hashes the whole semantic document and each primitive family.
func SemanticDigests(snap model.Snapshot) (Digests, error) {
	body, err := SemanticBytes(snap)
	if err != nil {
		return Digests{}, err
	}
	out := Digests{Snapshot: canonical.Hex(canonical.Digest(body))}
	out.Tools, err = primitiveDigest(func() (*wireSection, error) { return toolsSection(snap.Tools, false) })
	if err != nil {
		return Digests{}, err
	}
	out.Prompts, err = primitiveDigest(func() (*wireSection, error) { return promptsSection(snap.Prompts, false) })
	if err != nil {
		return Digests{}, err
	}
	out.Resources, err = primitiveDigest(func() (*wireSection, error) { return resourcesSection(snap.Resources, false) })
	if err != nil {
		return Digests{}, err
	}
	out.Templates, err = primitiveDigest(func() (*wireSection, error) { return templatesSection(snap.Templates, false) })
	if err != nil {
		return Digests{}, err
	}
	return out, nil
}

func primitiveDigest(section func() (*wireSection, error)) (string, error) {
	sec, err := section()
	if err != nil {
		return "", err
	}
	body, err := jsonv2.Marshal(sec.Items, jsonv2.Deterministic(true))
	if err != nil {
		return "", err
	}
	return canonical.Hex(canonical.Digest(body)), nil
}
