package model

import "encoding/json"

// DiffVerdict is the top-level comparison outcome.
// Incomplete inputs are never VerdictEqual.
type DiffVerdict string

const (
	VerdictEqual         DiffVerdict = "equal"
	VerdictChanged       DiffVerdict = "changed"
	VerdictIndeterminate DiffVerdict = "indeterminate"
)

// DiffCategory is what happened at a path.
type DiffCategory string

const (
	CategoryAddition DiffCategory = "addition"
	CategoryRemoval  DiffCategory = "removal"
	CategoryChange   DiffCategory = "change"
)

// DiffSeverity is the default risk class for a change.
type DiffSeverity string

const (
	SeverityBreaking DiffSeverity = "breaking"
	SeverityAdditive DiffSeverity = "additive"
	SeverityMetadata DiffSeverity = "metadata"
	SeverityUnknown  DiffSeverity = "unknown"
)

// Diff is a structured comparison of two public-interface snapshots.
// Changed is true only when Verdict is VerdictChanged (complete inputs with
// classified differences). Indeterminate snapshots never report a clean no-change.
type Diff struct {
	Verdict DiffVerdict `json:"verdict"`
	Changes []Change    `json:"changes"`
	Changed bool        `json:"changed"`
}

// Change is one classified difference at a stable JSON Pointer path.
type Change struct {
	Path      string          `json:"path"`
	Category  DiffCategory    `json:"category"`
	Severity  DiffSeverity    `json:"severity"`
	Rationale string          `json:"rationale"`
	Before    json.RawMessage `json:"before,omitzero"`
	After     json.RawMessage `json:"after,omitzero"`
}

// Rename maps a catalog identity in the before snapshot to one in the after snapshot.
// Kind is tools, prompts, resources, or templates. From/To are the identity keys
// (name, URI, or URI template). Without a mapping, a rename is a removal plus an addition.
type Rename struct {
	Kind PrimitiveKind
	From string
	To   string
}
