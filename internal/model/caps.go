package model

// HandshakeKind records which MCP lifecycle established the session.
type HandshakeKind string

const (
	HandshakeInitialize HandshakeKind = "initialize"
	HandshakeDiscover   HandshakeKind = "discover"
)

// ServerCapabilities is the declared server feature set after negotiation.
// Presence flags are true when the matching capability object was present.
type ServerCapabilities struct {
	Experimental         []string
	Extensions           []string
	Tools                bool
	Prompts              bool
	Resources            bool
	Completions          bool
	Logging              bool
	ToolsListChanged     bool
	PromptsListChanged   bool
	ResourcesListChanged bool
	ResourceSubscribe    bool
}

// Advertises reports whether initialize/discover included the catalog family.
// Resource templates are gated by the resources capability.
func (c ServerCapabilities) Advertises(kind PrimitiveKind) bool {
	switch kind {
	case KindTools:
		return c.Tools
	case KindPrompts:
		return c.Prompts
	case KindResources, KindTemplates:
		return c.Resources
	default:
		return false
	}
}
