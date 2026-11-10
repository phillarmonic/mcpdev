// Package process supervises stdio MCP child processes.
//
// It owns argv construction, environment merge, stdin/stdout pipes, bounded
// stderr capture, and shutdown escalation. It never parses stdout as JSON-RPC.
package process
