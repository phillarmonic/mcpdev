// Package testkit is the dual-era MCP fixture laboratory.
//
// Production packages must not import this package. Tests of transport,
// protocol, inspect, diff, scenario, and watch code share these fixtures
// instead of inventing ad hoc servers. The laboratory does not import the
// official Go MCP SDK; that import stays in internal/protocol.
package testkit
