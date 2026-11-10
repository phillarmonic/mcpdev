package snapshot

import (
	"time"

	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

// Meta is capture-time provenance stored on the document and omitted from semantic bytes.
type Meta struct {
	CapturedAt       time.Time
	Headers          map[string]string
	GeneratorVersion string
	Kind             string
	Profile          string
	URL              string
	SessionID        string
	Command          []string
}

func (m Meta) provenance() model.SnapshotProvenance {
	return model.SnapshotProvenance{
		Kind:      m.Kind,
		Profile:   m.Profile,
		URL:       redact.URLString(m.URL, redact.Options{}),
		Command:   redact.Argv(m.Command, redact.Options{}),
		Headers:   redact.MapString(m.Headers, redact.Options{}),
		SessionID: m.SessionID,
	}
}

func generatorVersion(m Meta) string {
	if m.GeneratorVersion != "" {
		return m.GeneratorVersion
	}
	return "0.1.1"
}
