package extension

import "github.com/phillarmonic/mcpdev/internal/errclass"

// ExperimentalGRPC is the documented adapter contract for talking to a gRPC
// endpoint that might carry MCP-shaped JSON-RPC. It is not a standard MCP
// transport (neither 2025-11-25 nor 2026-07-28 lists gRPC). There is no
// official protobuf service, method set, or metadata mapping.
//
// Do not ship a concrete codec until a real endpoint supplies:
//   - the protobuf service and RPC names
//   - how initialize vs per-request _meta is encoded
//   - how cancellation and streaming map onto gRPC
//   - how credentials travel (metadata vs TLS vs call credentials)
//
// Until then, a gRPC adapter MUST implement Adapter, declare Semantics, and
// remain opt-in. It MUST NOT appear beside stdio and http in default CLI enums.
type ExperimentalGRPC struct {
	// Service is the protobuf service name. Empty means unspecified.
	Service string
	// Framing is how JSON-RPC bytes sit on the RPC. Empty means unspecified.
	Framing string
}

// Ready reports whether a concrete mapping was supplied. Empty fields mean
// the mapping is still unknown; that is not success.
func (g ExperimentalGRPC) Ready() bool {
	return g.Service != "" && g.Framing != ""
}

// ErrUnspecified is returned when code attempts to use gRPC without a mapping.
var ErrUnspecified = errclass.Usage{Msg: "grpc adapter: service/protobuf mapping is unspecified; gRPC is not a standard MCP transport"}
