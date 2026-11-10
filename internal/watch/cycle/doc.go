// Package cycle probes a negotiated MCP session for watch readiness, captures a
// semantic snapshot, and compares it with the previous ready generation.
// A failed probe never replaces the in-memory comparison baseline.
package cycle
