package registry

// Connector names that appear in configuration. Default CLI enums are stdio and http only.
const (
	NameStdio     = "stdio"
	NameHTTP      = "http"
	NameHTTP2025  = "http-2025-11-25"
	NameHTTP2026  = "http-2026-07-28"
	NameForbidden = "grpc"
)

const (
	ProtocolAuto = "auto"
)
