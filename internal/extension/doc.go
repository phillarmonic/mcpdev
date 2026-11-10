// Package extension is the experimental custom-transport boundary.
//
// Adapters here carry MCP JSON-RPC framing over a non-standard channel.
// They must not import the official Go SDK or internal/protocol: protocol
// owns JSON-RPC session semantics; this package only owns the byte channel
// and the required capability declarations.
//
// gRPC is not an MCP-standard transport. There is no protobuf service mapping
// in this module. See ExperimentalGRPC.
package extension
