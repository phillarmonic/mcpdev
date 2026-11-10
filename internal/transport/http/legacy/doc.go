// Package legacy implements MCP 2025-11-25 Streamable HTTP.
//
// This is the stateful transport: MCP-Protocol-Version, optional Mcp-Session-Id,
// POST JSON or SSE, GET server-initiated SSE with bounded reconnect, and DELETE teardown.
// It is not HTTP+SSE (2024-11-05) and not the 2026-07-28 POST-only shape.
package legacy
