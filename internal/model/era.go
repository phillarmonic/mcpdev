// Package model holds SDK-free domain types shared by ports and adapters.
package model

// ProtocolEra is a first-class MCP specification date.
// 2025-11-25 and 2026-07-28 are never collapsed into one value.
type ProtocolEra string

const (
	Era20251125 ProtocolEra = "2025-11-25"
	Era20260728 ProtocolEra = "2026-07-28"
)

// DiscoverState records how server/discover completed for a session.
type DiscoverState string

const (
	DiscoverOK            DiscoverState = "ok"
	DiscoverSkipped       DiscoverState = "skipped"
	DiscoverFailed        DiscoverState = "failed"
	DiscoverNotApplicable DiscoverState = "not_applicable"
)
