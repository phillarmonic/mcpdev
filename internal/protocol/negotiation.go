package protocol

import (
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/protocol/modern"
)

// Decision records how era negotiation completed. Inventory and diagnostics read these fields.
type Decision struct {
	Requested  string
	Negotiated string
	Reason     string
	Discovered []string
	Fallback   bool
}

// HTTP400 decides whether an HTTP 400 is a modern validation error or a legacy-shaped fallback.
func HTTP400(requested string, status int, body []byte) Decision {
	got := modern.ClassifyHTTP(status, body)
	out := Decision{Requested: requested}
	if got.StayModern() {
		out.Discovered = got.Supported
		out.Negotiated = modern.Select(modern.PreferredVersions, got.Supported)
		out.Reason = string(got.Kind)
		return out
	}
	out.Fallback = true
	out.Reason = modern.ReasonFallback(got)
	return out
}

func applyDecision(info *model.SessionInfo, d Decision) {
	info.RequestedVersion = d.Requested
	info.DiscoveredVersions = d.Discovered
	info.Fallback = d.Fallback
	info.FallbackReason = d.Reason
	if info.ProtocolVersion == "" && d.Negotiated != "" {
		info.ProtocolVersion = d.Negotiated
	}
}
