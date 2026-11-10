package cycle

import (
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/model"
)

// Outcome is the operator-facing interface comparison class for one generation.
type Outcome string

const (
	OutcomeUnchanged     Outcome = "unchanged"
	OutcomeAdditive      Outcome = "additive"
	OutcomeMetadata      Outcome = "metadata"
	OutcomeBreaking      Outcome = "breaking"
	OutcomeIndeterminate Outcome = "indeterminate"
)

// Result is one probe attempt after handshake.
type Result struct {
	Outcome  Outcome
	Message  string
	Diff     model.Diff
	Snapshot model.Snapshot
	Duration time.Duration
	Ready    bool
	Repeated bool
}

// Record is one bounded history row (ready or failed).
type Record struct {
	At       time.Time
	Outcome  Outcome
	Message  string
	Reason   string
	Duration time.Duration
	Ready    bool
}

func rollup(d model.Diff) Outcome {
	if d.Verdict == model.VerdictIndeterminate {
		return OutcomeIndeterminate
	}
	if !d.Changed && len(d.Changes) == 0 {
		return OutcomeUnchanged
	}
	class := OutcomeMetadata
	for _, ch := range d.Changes {
		switch ch.Severity {
		case model.SeverityBreaking, model.SeverityUnknown:
			return OutcomeBreaking
		case model.SeverityAdditive:
			class = OutcomeAdditive
		case model.SeverityMetadata:
			if class != OutcomeAdditive {
				class = OutcomeMetadata
			}
		}
	}
	return class
}

func fingerprint(outcome Outcome, d model.Diff) string {
	var b strings.Builder
	b.WriteString(string(outcome))
	for _, ch := range d.Changes {
		b.WriteByte('|')
		b.WriteString(string(ch.Severity))
		b.WriteByte(':')
		b.WriteString(string(ch.Category))
		b.WriteByte(':')
		b.WriteString(ch.Path)
	}
	return b.String()
}
