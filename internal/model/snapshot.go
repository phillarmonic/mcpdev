package model

import "time"

// SnapshotSchemaVersion is the on-disk snapshot schema this mcpdev writes.
const SnapshotSchemaVersion = 1

// SnapshotGeneratorName is the generator.name value written by this tool.
const SnapshotGeneratorName = "mcpdev"

// Snapshot is a portable public-interface contract plus capture provenance.
// Semantic comparison must ignore Provenance, generator version, and CapturedAt.
type Snapshot struct {
	CapturedAt        time.Time
	Provenance        SnapshotProvenance
	Protocol          SnapshotProtocol
	Server            SnapshotServer
	GeneratorVersion  string
	IncompleteReasons []string
	Tools             CatalogSection
	Prompts           CatalogSection
	Resources         CatalogSection
	Templates         CatalogSection
	Capabilities      ServerCapabilities
	SchemaVersion     int
	Incomplete        bool
}

// SnapshotProvenance is diagnostic capture context. It is stored and is not
// part of semantic comparison (connection details are expected to change).
type SnapshotProvenance struct {
	Headers   map[string]string
	Kind      string
	Profile   string
	URL       string
	SessionID string
	Command   []string
}

// SnapshotProtocol is the negotiated MCP era recorded on the snapshot.
type SnapshotProtocol struct {
	Era       ProtocolEra
	Version   string
	Handshake HandshakeKind
	Discover  DiscoverState
}

// SnapshotServer is serverInfo copied from the inventory.
type SnapshotServer struct {
	Name         string
	Title        string
	Version      string
	Instructions string
}
