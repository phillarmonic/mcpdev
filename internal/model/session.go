package model

// Identity is the mcpdev client name presented during handshake.
type Identity struct {
	Name    string
	Version string
	Title   string
}

// SessionInfo is negotiated session metadata with no SDK identifiers.
type SessionInfo struct {
	Identity           Identity
	ProtocolVersion    string
	Instructions       string
	ServerName         string
	ServerTitle        string
	ServerVersion      string
	Handshake          HandshakeKind
	SessionID          string
	DiscoverState      DiscoverState
	RequestedVersion   string
	FallbackReason     string
	Era                ProtocolEra
	DiscoveredVersions []string
	Capabilities       ServerCapabilities
	Fallback           bool
}
