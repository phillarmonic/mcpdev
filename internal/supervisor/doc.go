// Package supervisor swaps rebuilt stdio MCP processes across generations.
//
// It tracks idle, building, starting, ready, stopping, failed, and closed
// states, rejects stale callbacks, and keeps at most one promoted generation.
package supervisor
