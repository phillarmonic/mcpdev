package model

// SectionStatus is how one catalog family was collected.
// unsupported, failed, and complete-empty must stay distinct.
type SectionStatus string

const (
	SectionComplete    SectionStatus = "complete"
	SectionUnsupported SectionStatus = "unsupported"
	SectionFailed      SectionStatus = "failed"
	SectionIncomplete  SectionStatus = "incomplete"
)

// Stable incomplete-reason tokens. Callers must not invent silent empty catalogs.
const (
	ReasonCapabilityAbsent   = "capability_absent"
	ReasonListFailed         = "list_failed"
	ReasonPaginationStopped  = "pagination_stopped"
	ReasonCursorLoop         = "cursor_loop"
	ReasonItemLimit          = "item_limit"
	ReasonPageIncomplete     = "page_incomplete"
	ReasonDuplicateItem      = "duplicate_item"
	ReasonCollectionCanceled = "collection_canceled"
)

// CatalogSection is one primitive family after cursor walking.
type CatalogSection struct {
	Kind          PrimitiveKind
	StoppedCursor string
	Status        SectionStatus
	Reasons       []string
	Tools         []Tool
	Prompts       []Prompt
	Resources     []Resource
	Templates     []ResourceTemplate
	Pages         int
	ItemCount     int
}

// Inventory is the deterministic public surface of one session.
// Incomplete is true when any advertised family failed or stopped; unsupported families do not set it.
type Inventory struct {
	Instructions    string
	ServerName      string
	ServerTitle     string
	ServerVersion   string
	ProtocolVersion string
	Handshake       HandshakeKind
	DiscoverState   DiscoverState
	Era             ProtocolEra
	Reasons         []string
	Tools           CatalogSection
	Prompts         CatalogSection
	Resources       CatalogSection
	Templates       CatalogSection
	Capabilities    ServerCapabilities
	Incomplete      bool
}
