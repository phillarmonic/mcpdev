package snapshot

import (
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

func fromWire(doc wireDoc) (model.Snapshot, error) {
	snap := model.Snapshot{
		SchemaVersion:     doc.SchemaVersion,
		GeneratorVersion:  doc.Generator.Version,
		Incomplete:        doc.Incomplete,
		IncompleteReasons: emptyStrings(doc.IncompleteReasons),
		Protocol: model.SnapshotProtocol{
			Era:       model.ProtocolEra(doc.Protocol.Era),
			Version:   doc.Protocol.Version,
			Handshake: model.HandshakeKind(doc.Protocol.Handshake),
			Discover:  model.DiscoverState(doc.Protocol.Discover),
		},
		Server: model.SnapshotServer{
			Name:         doc.Server.Name,
			Title:        doc.Server.Title,
			Version:      doc.Server.Version,
			Instructions: doc.Server.Instructions,
		},
		Capabilities: model.ServerCapabilities{
			Experimental:         doc.Capabilities.Experimental,
			Extensions:           doc.Capabilities.Extensions,
			Tools:                doc.Capabilities.Tools,
			Prompts:              doc.Capabilities.Prompts,
			Resources:            doc.Capabilities.Resources,
			Completions:          doc.Capabilities.Completions,
			Logging:              doc.Capabilities.Logging,
			ToolsListChanged:     doc.Capabilities.ToolsListChanged,
			PromptsListChanged:   doc.Capabilities.PromptsListChanged,
			ResourcesListChanged: doc.Capabilities.ResourcesListChanged,
			ResourceSubscribe:    doc.Capabilities.ResourceSubscribe,
		},
		Provenance: model.SnapshotProvenance{
			Kind:      doc.Provenance.Kind,
			Profile:   doc.Provenance.Profile,
			URL:       doc.Provenance.URL,
			Command:   doc.Provenance.Command,
			Headers:   doc.Provenance.Headers,
			SessionID: doc.Provenance.SessionID,
		},
	}
	if doc.CapturedAt != "" {
		t, err := time.Parse(capturedAtLayout, doc.CapturedAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, doc.CapturedAt)
		}
		if err != nil {
			return model.Snapshot{}, errclass.Usage{Msg: "snapshot: invalid captured_at", Err: err}
		}
		snap.CapturedAt = t.UTC()
	}
	var err error
	if snap.Tools, err = toolsFrom(doc.Tools); err != nil {
		return model.Snapshot{}, err
	}
	if snap.Prompts, err = promptsFrom(doc.Prompts); err != nil {
		return model.Snapshot{}, err
	}
	if snap.Resources, err = resourcesFrom(doc.Resources); err != nil {
		return model.Snapshot{}, err
	}
	if snap.Templates, err = templatesFrom(doc.Templates); err != nil {
		return model.Snapshot{}, err
	}
	return snap, nil
}

func toolsFrom(sec *wireSection) (model.CatalogSection, error) {
	if sec == nil {
		return model.CatalogSection{}, nil
	}
	out := baseSection(model.KindTools, sec)
	for _, item := range sec.Items {
		out.Tools = append(out.Tools, model.Tool{
			Name:         item.Name,
			Title:        item.Title,
			Description:  item.Description,
			InputSchema:  rawJSON(item.InputSchema),
			OutputSchema: rawJSON(item.OutputSchema),
			Meta:         rawJSON(item.Meta),
			Icons:        iconsFrom(item.Icons),
			Annotations:  toolAnnsFrom(item.Annotations),
		})
	}
	return out, nil
}

func promptsFrom(sec *wireSection) (model.CatalogSection, error) {
	if sec == nil {
		return model.CatalogSection{}, nil
	}
	out := baseSection(model.KindPrompts, sec)
	for _, item := range sec.Items {
		p := model.Prompt{
			Name:        item.Name,
			Title:       item.Title,
			Description: item.Description,
			Meta:        rawJSON(item.Meta),
			Icons:       iconsFrom(item.Icons),
		}
		for _, a := range item.Arguments {
			p.Arguments = append(p.Arguments, model.PromptArgument{
				Name: a.Name, Title: a.Title, Description: a.Description, Required: a.Required,
			})
		}
		out.Prompts = append(out.Prompts, p)
	}
	return out, nil
}

func resourcesFrom(sec *wireSection) (model.CatalogSection, error) {
	if sec == nil {
		return model.CatalogSection{}, nil
	}
	out := baseSection(model.KindResources, sec)
	for _, item := range sec.Items {
		out.Resources = append(out.Resources, model.Resource{
			URI:         item.URI,
			Name:        item.Name,
			Title:       item.Title,
			Description: item.Description,
			MIMEType:    item.MIMEType,
			Size:        item.Size,
			Meta:        rawJSON(item.Meta),
			Icons:       iconsFrom(item.Icons),
			Annotations: resAnnsFrom(item.Hints),
		})
	}
	return out, nil
}

func templatesFrom(sec *wireSection) (model.CatalogSection, error) {
	if sec == nil {
		return model.CatalogSection{}, nil
	}
	out := baseSection(model.KindTemplates, sec)
	for _, item := range sec.Items {
		out.Templates = append(out.Templates, model.ResourceTemplate{
			URITemplate: item.URITemplate,
			Name:        item.Name,
			Title:       item.Title,
			Description: item.Description,
			MIMEType:    item.MIMEType,
			Meta:        rawJSON(item.Meta),
			Icons:       iconsFrom(item.Icons),
			Annotations: resAnnsFrom(item.Hints),
		})
	}
	return out, nil
}

func baseSection(kind model.PrimitiveKind, sec *wireSection) model.CatalogSection {
	count := sec.Count
	if count == 0 {
		count = len(sec.Items)
	}
	return model.CatalogSection{
		Kind:          kind,
		Status:        model.SectionStatus(sec.Status),
		Reasons:       emptyStrings(sec.Reasons),
		StoppedCursor: sec.StoppedCursor,
		Pages:         sec.Pages,
		ItemCount:     count,
	}
}

func iconsFrom(in []wireIcon) []model.Icon {
	if len(in) == 0 {
		return nil
	}
	out := make([]model.Icon, 0, len(in))
	for _, icon := range in {
		out = append(out, model.Icon{Source: icon.Source, MIMEType: icon.MIMEType, Theme: icon.Theme, Sizes: icon.Sizes})
	}
	return out
}

func toolAnnsFrom(a *wireToolAnn) *model.ToolAnnotations {
	if a == nil {
		return nil
	}
	return &model.ToolAnnotations{
		Title:           a.Title,
		ReadOnlyHint:    a.ReadOnlyHint,
		DestructiveHint: a.DestructiveHint,
		IdempotentHint:  a.IdempotentHint,
		OpenWorldHint:   a.OpenWorldHint,
	}
}

func resAnnsFrom(a *wireResAnn) *model.Annotations {
	if a == nil {
		return nil
	}
	return &model.Annotations{LastModified: a.LastModified, Audience: a.Audience, Priority: a.Priority}
}

func rawJSON(v jsontext.Value) json.RawMessage {
	if len(v) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), v...)
}

func validateCollection(snap model.Snapshot) error {
	if snap.SchemaVersion != 0 && snap.SchemaVersion != model.SnapshotSchemaVersion {
		return errclass.Usage{Msg: fmt.Sprintf("snapshot schema version %d is not current (%d)", snap.SchemaVersion, model.SnapshotSchemaVersion)}
	}
	computed := sectionIncomplete(snap.Tools) || sectionIncomplete(snap.Prompts) || sectionIncomplete(snap.Resources) || sectionIncomplete(snap.Templates)
	if computed && !snap.Incomplete {
		return errclass.Usage{Msg: "snapshot: incomplete collection cannot be stored as complete"}
	}
	if snap.Incomplete && !computed && len(snap.IncompleteReasons) == 0 {
		return errclass.Usage{Msg: "snapshot: incomplete is true without section or reason evidence"}
	}
	return nil
}

func sectionIncomplete(sec model.CatalogSection) bool {
	switch sec.Status {
	case model.SectionFailed, model.SectionIncomplete:
		return true
	default:
		return false
	}
}
