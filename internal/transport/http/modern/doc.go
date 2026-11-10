// Package modern implements MCP 2026-07-28 Streamable HTTP.
//
// This is the stateless POST-only transport: MCP-Protocol-Version, Mcp-Method,
// and encoded Mcp-Name on every request. There is no protocol session, GET SSE,
// DELETE teardown, or notifications/cancelled. JSON and request-scoped SSE share
// one Exchange type. Closing the HTTP response stream is cancellation.
package modern
