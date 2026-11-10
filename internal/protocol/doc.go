// Package protocol wraps the official MCP Go SDK behind mcpdev session and result types.
//
// github.com/modelcontextprotocol/go-sdk is imported only from this package.
// Handshake version and capability-object checks live in protocol/legacy and stay SDK-free.
// 2026-07-28 discover, _meta, and modern error classification live in protocol/modern.
package protocol
