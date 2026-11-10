package testkit

import "time"

// ProtocolEra is a first-class MCP specification date.
// Fixtures never collapse 2025-11-25 and 2026-07-28 into one code path.
type ProtocolEra string

const (
	Era20251125 ProtocolEra = "2025-11-25"
	Era20260728 ProtocolEra = "2026-07-28"
)

// TransportMode is how a fixture speaks JSON-RPC.
type TransportMode string

const (
	TransportStdio TransportMode = "stdio"
	TransportHTTP  TransportMode = "http"
)

// MalformedKind is an intentional protocol defect. Empty means well-formed.
type MalformedKind string

const (
	MalformedNone               MalformedKind = ""
	MalformedJSON               MalformedKind = "invalid-json"
	MalformedEmptyHTTPBody      MalformedKind = "empty-http-body"
	MalformedUnrecognizedError  MalformedKind = "unrecognized-error"
	MalformedUnsupportedVersion MalformedKind = "unsupported-version"
	MalformedCapabilityObject   MalformedKind = "malformed-capabilities"
	MalformedExitOnFirstRPC     MalformedKind = "exit-on-first-rpc"
	MalformedMissingClientCap   MalformedKind = "missing-client-capability"
	MalformedModernUnsupported  MalformedKind = "modern-unsupported-version"
)

// ShutdownMode controls how the fixture ends.
type ShutdownMode string

const (
	ShutdownClean      ShutdownMode = "clean"
	ShutdownWaitCancel ShutdownMode = "wait-cancel"
)

// Capabilities are advertised on initialize (2025-11-25) or discover (2026-07-28).
type Capabilities struct {
	Tools             bool
	Prompts           bool
	Resources         bool
	ListChanged       bool
	ResourceSubscribe bool
}

// SessionPolicy is Streamable HTTP session behavior for the stateful era.
type SessionPolicy struct {
	ID     string
	MintID bool
}

// ShutdownPolicy is how pipes, HTTP, and child processes stop.
type ShutdownPolicy struct {
	Mode ShutdownMode
}

// Descriptor declares one fixture. Later tasks look up named descriptors
// instead of assembling servers inline.
type Descriptor struct {
	Shutdown          ShutdownPolicy
	Era               ProtocolEra
	Transport         TransportMode
	Name              string
	Malformed         MalformedKind
	AuthToken         string
	Instructions      string
	Session           SessionPolicy
	ExtraTools        int
	Delay             time.Duration
	PageSize          int
	Capabilities      Capabilities
	AuthRequired      bool
	RequireHandshake  bool
	SSEOnCall         bool
	NotifyListChanged bool
	DropGETOnce       bool
}

func (d Descriptor) pageSize() int {
	if d.PageSize > 0 {
		return d.PageSize
	}
	return 50
}

func (d Descriptor) sessionID() string {
	if d.Session.ID != "" {
		return d.Session.ID
	}
	return "fixture-session"
}

func (d Descriptor) shutdownMode() ShutdownMode {
	if d.Shutdown.Mode == "" {
		return ShutdownClean
	}
	return d.Shutdown.Mode
}
