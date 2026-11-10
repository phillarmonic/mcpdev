package snapshot

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"time"

	"github.com/phillarmonic/mcpdev/internal/canonical"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

const capturedAtLayout = time.RFC3339Nano

// Encode writes the full snapshot document, including provenance.
func Encode(snap model.Snapshot) ([]byte, error) {
	if err := validateCollection(snap); err != nil {
		return nil, err
	}
	doc, err := toWire(snap, true)
	if err != nil {
		return nil, err
	}
	b, err := jsonv2.Marshal(doc, jsonv2.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: fmt.Errorf("snapshot: encode: %w", err)}
	}
	return append(b, '\n'), nil
}

// SemanticBytes encodes only fields that describe the public interface.
// Generator version, capture time, connection provenance, stopped cursors, and page counts are omitted.
func SemanticBytes(snap model.Snapshot) ([]byte, error) {
	if err := validateCollection(snap); err != nil {
		return nil, err
	}
	doc, err := toWire(snap, false)
	if err != nil {
		return nil, err
	}
	doc.Generator = wireGenerator{}
	doc.CapturedAt = ""
	doc.Provenance = wireProvenance{}
	b, err := jsonv2.Marshal(doc, jsonv2.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: fmt.Errorf("snapshot: semantic encode: %w", err)}
	}
	return b, nil
}

// Decode reads a snapshot file body. Unknown future schema versions are rejected.
func Decode(raw []byte) (model.Snapshot, error) {
	var head struct {
		SchemaVersion *int `json:"schema_version"`
	}
	if err := jsonv2.Unmarshal(raw, &head); err != nil {
		return model.Snapshot{}, errclass.Usage{Msg: "snapshot: corrupt file", Err: err}
	}
	if head.SchemaVersion == nil {
		return model.Snapshot{}, errclass.Usage{Msg: "snapshot: missing schema_version"}
	}
	switch v := *head.SchemaVersion; {
	case v == model.SnapshotSchemaVersion:
		// current
	case v > model.SnapshotSchemaVersion:
		return model.Snapshot{}, errclass.Usage{Msg: fmt.Sprintf(
			"snapshot schema version %d is newer than this mcpdev (%d); upgrade mcpdev", v, model.SnapshotSchemaVersion)}
	default:
		return model.Snapshot{}, errclass.Usage{Msg: fmt.Sprintf(
			"snapshot schema version %d is not supported (current is %d)", v, model.SnapshotSchemaVersion)}
	}
	var doc wireDoc
	if err := jsonv2.Unmarshal(raw, &doc); err != nil {
		return model.Snapshot{}, errclass.Usage{Msg: "snapshot: corrupt file", Err: err}
	}
	snap, err := fromWire(doc)
	if err != nil {
		return model.Snapshot{}, err
	}
	if err := validateCollection(snap); err != nil {
		return model.Snapshot{}, err
	}
	return snap, nil
}

func toWire(snap model.Snapshot, includeDiag bool) (wireDoc, error) {
	version := snap.SchemaVersion
	if version == 0 {
		version = model.SnapshotSchemaVersion
	}
	doc := wireDoc{
		SchemaVersion: version,
		Protocol: wireProtocol{
			Era:       string(snap.Protocol.Era),
			Version:   snap.Protocol.Version,
			Handshake: string(snap.Protocol.Handshake),
			Discover:  string(snap.Protocol.Discover),
		},
		Server: wireServer{
			Name:         snap.Server.Name,
			Title:        snap.Server.Title,
			Version:      snap.Server.Version,
			Instructions: snap.Server.Instructions,
		},
		Capabilities:      wireCapsFrom(snap.Capabilities),
		Incomplete:        snap.Incomplete,
		IncompleteReasons: emptyStrings(snap.IncompleteReasons),
	}
	if includeDiag {
		doc.Generator = wireGenerator{Name: model.SnapshotGeneratorName, Version: snap.GeneratorVersion}
		if !snap.CapturedAt.IsZero() {
			doc.CapturedAt = snap.CapturedAt.UTC().Format(capturedAtLayout)
		}
		doc.Provenance = wireProvenance{
			Kind:      snap.Provenance.Kind,
			Profile:   snap.Provenance.Profile,
			URL:       snap.Provenance.URL,
			Command:   snap.Provenance.Command,
			Headers:   snap.Provenance.Headers,
			SessionID: snap.Provenance.SessionID,
		}
	}
	var err error
	if snap.Tools.Kind != "" {
		if doc.Tools, err = toolsSection(snap.Tools, includeDiag); err != nil {
			return wireDoc{}, err
		}
	}
	if snap.Prompts.Kind != "" {
		if doc.Prompts, err = promptsSection(snap.Prompts, includeDiag); err != nil {
			return wireDoc{}, err
		}
	}
	if snap.Resources.Kind != "" {
		if doc.Resources, err = resourcesSection(snap.Resources, includeDiag); err != nil {
			return wireDoc{}, err
		}
	}
	if snap.Templates.Kind != "" {
		if doc.Templates, err = templatesSection(snap.Templates, includeDiag); err != nil {
			return wireDoc{}, err
		}
	}
	return doc, nil
}

func toolsSection(sec model.CatalogSection, diag bool) (*wireSection, error) {
	items := make([]wireItem, 0, len(sec.Tools))
	for _, t := range sec.Tools {
		item := wireItem{
			Name:        t.Name,
			Title:       t.Title,
			Description: t.Description,
			Icons:       wireIcons(t.Icons),
			Annotations: wireToolAnns(t.Annotations),
		}
		var err error
		if item.InputSchema, err = asSchema(t.InputSchema); err != nil {
			return nil, err
		}
		if item.OutputSchema, err = asSchema(t.OutputSchema); err != nil {
			return nil, err
		}
		if item.Meta, err = asValue(t.Meta); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return packSection(sec, items, diag), nil
}

func promptsSection(sec model.CatalogSection, diag bool) (*wireSection, error) {
	items := make([]wireItem, 0, len(sec.Prompts))
	for _, p := range sec.Prompts {
		item := wireItem{
			Name:        p.Name,
			Title:       p.Title,
			Description: p.Description,
			Icons:       wireIcons(p.Icons),
			Arguments:   wireArgs(p.Arguments),
		}
		var err error
		if item.Meta, err = asValue(p.Meta); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return packSection(sec, items, diag), nil
}

func resourcesSection(sec model.CatalogSection, diag bool) (*wireSection, error) {
	items := make([]wireItem, 0, len(sec.Resources))
	for _, r := range sec.Resources {
		item := wireItem{
			URI:         r.URI,
			Name:        r.Name,
			Title:       r.Title,
			Description: r.Description,
			MIMEType:    r.MIMEType,
			Size:        r.Size,
			Icons:       wireIcons(r.Icons),
			Hints:       wireResAnns(r.Annotations),
		}
		var err error
		if item.Meta, err = asValue(r.Meta); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return packSection(sec, items, diag), nil
}

func templatesSection(sec model.CatalogSection, diag bool) (*wireSection, error) {
	items := make([]wireItem, 0, len(sec.Templates))
	for _, r := range sec.Templates {
		item := wireItem{
			URITemplate: r.URITemplate,
			Name:        r.Name,
			Title:       r.Title,
			Description: r.Description,
			MIMEType:    r.MIMEType,
			Icons:       wireIcons(r.Icons),
			Hints:       wireResAnns(r.Annotations),
		}
		var err error
		if item.Meta, err = asValue(r.Meta); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return packSection(sec, items, diag), nil
}

func packSection(sec model.CatalogSection, items []wireItem, diag bool) *wireSection {
	out := &wireSection{
		Status:  string(sec.Status),
		Reasons: emptyStrings(sec.Reasons),
		Items:   items,
		Count:   sec.ItemCount,
	}
	if diag {
		out.StoppedCursor = sec.StoppedCursor
		out.Pages = sec.Pages
	}
	return out
}

func asValue(raw []byte) (jsontext.Value, error) {
	return asEncoded(raw, CanonicalJSON)
}

func asSchema(raw []byte) (jsontext.Value, error) {
	return asEncoded(raw, canonical.Schema)
}

func asEncoded(raw []byte, canon func([]byte) ([]byte, error)) (jsontext.Value, error) {
	if len(raw) == 0 {
		var none jsontext.Value
		return none, nil
	}
	b, err := canon(raw)
	if err != nil {
		return nil, err
	}
	return jsontext.Value(b), nil
}

func wireIcons(in []model.Icon) []wireIcon {
	if len(in) == 0 {
		return nil
	}
	out := make([]wireIcon, 0, len(in))
	for _, icon := range in {
		out = append(out, wireIcon{Source: icon.Source, MIMEType: icon.MIMEType, Theme: icon.Theme, Sizes: icon.Sizes})
	}
	return out
}

func wireToolAnns(a *model.ToolAnnotations) *wireToolAnn {
	if a == nil {
		return nil
	}
	return &wireToolAnn{
		Title:           a.Title,
		ReadOnlyHint:    a.ReadOnlyHint,
		DestructiveHint: a.DestructiveHint,
		IdempotentHint:  a.IdempotentHint,
		OpenWorldHint:   a.OpenWorldHint,
	}
}

func wireResAnns(a *model.Annotations) *wireResAnn {
	if a == nil {
		return nil
	}
	return &wireResAnn{LastModified: a.LastModified, Audience: a.Audience, Priority: a.Priority}
}

func wireArgs(in []model.PromptArgument) []wireArg {
	if len(in) == 0 {
		return nil
	}
	out := make([]wireArg, 0, len(in))
	for _, a := range in {
		out = append(out, wireArg{Name: a.Name, Title: a.Title, Description: a.Description, Required: a.Required})
	}
	return out
}
