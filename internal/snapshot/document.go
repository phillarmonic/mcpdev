package snapshot

import (
	"encoding/json"

	"github.com/phillarmonic/mcpdev/internal/canonical"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// FromInventory builds a snapshot document from a collected inventory.
// Incomplete inventories stay incomplete; they cannot be stored as complete.
func FromInventory(inv model.Inventory, meta Meta) (model.Snapshot, error) {
	snap := model.Snapshot{
		SchemaVersion:    model.SnapshotSchemaVersion,
		GeneratorVersion: generatorVersion(meta),
		CapturedAt:       meta.CapturedAt.UTC(),
		Provenance:       meta.provenance(),
		Protocol: model.SnapshotProtocol{
			Era:       inv.Era,
			Version:   inv.ProtocolVersion,
			Handshake: inv.Handshake,
			Discover:  inv.DiscoverState,
		},
		Server: model.SnapshotServer{
			Name:         inv.ServerName,
			Title:        inv.ServerTitle,
			Version:      inv.ServerVersion,
			Instructions: inv.Instructions,
		},
		Capabilities:      inv.Capabilities,
		Incomplete:        inv.Incomplete,
		IncompleteReasons: cloneStrings(inv.Reasons),
		Tools:             inv.Tools,
		Prompts:           inv.Prompts,
		Resources:         inv.Resources,
		Templates:         inv.Templates,
	}
	if err := canonicalizeSnapshot(&snap); err != nil {
		return model.Snapshot{}, err
	}
	if err := validateCollection(snap); err != nil {
		return model.Snapshot{}, err
	}
	return snap, nil
}

func canonicalizeSnapshot(snap *model.Snapshot) error {
	var err error
	if snap.Tools.Tools, err = mapTools(snap.Tools.Tools); err != nil {
		return err
	}
	if snap.Prompts.Prompts, err = mapPrompts(snap.Prompts.Prompts); err != nil {
		return err
	}
	if snap.Resources.Resources, err = mapResources(snap.Resources.Resources); err != nil {
		return err
	}
	if snap.Templates.Templates, err = mapTemplates(snap.Templates.Templates); err != nil {
		return err
	}
	return nil
}

func mapTools(in []model.Tool) ([]model.Tool, error) {
	out := make([]model.Tool, len(in))
	for i, t := range in {
		inSchema, err := canonical.Schema(t.InputSchema)
		if err != nil {
			return nil, err
		}
		outSchema, err := canonical.Schema(t.OutputSchema)
		if err != nil {
			return nil, err
		}
		meta, err := CanonicalMeta(t.Meta)
		if err != nil {
			return nil, err
		}
		t.InputSchema = json.RawMessage(inSchema)
		t.OutputSchema = json.RawMessage(outSchema)
		t.Meta = json.RawMessage(meta)
		out[i] = t
	}
	return out, nil
}

func mapPrompts(in []model.Prompt) ([]model.Prompt, error) {
	out := make([]model.Prompt, len(in))
	for i, p := range in {
		meta, err := CanonicalMeta(p.Meta)
		if err != nil {
			return nil, err
		}
		p.Meta = json.RawMessage(meta)
		out[i] = p
	}
	return out, nil
}

func mapResources(in []model.Resource) ([]model.Resource, error) {
	out := make([]model.Resource, len(in))
	for i, r := range in {
		meta, err := CanonicalMeta(r.Meta)
		if err != nil {
			return nil, err
		}
		r.Meta = json.RawMessage(meta)
		out[i] = r
	}
	return out, nil
}

func mapTemplates(in []model.ResourceTemplate) ([]model.ResourceTemplate, error) {
	out := make([]model.ResourceTemplate, len(in))
	for i, r := range in {
		meta, err := CanonicalMeta(r.Meta)
		if err != nil {
			return nil, err
		}
		r.Meta = json.RawMessage(meta)
		out[i] = r
	}
	return out, nil
}

func cloneStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
