package modern

import "github.com/phillarmonic/mcpdev/internal/model"

// Protocol version and params._meta keys from MCP 2026-07-28 (SEP-2575).
const (
	Era = string(model.Era20260728)

	MetaProtocolVersion    = "io.modelcontextprotocol/protocolVersion"
	MetaClientInfo         = "io.modelcontextprotocol/clientInfo"
	MetaClientCapabilities = "io.modelcontextprotocol/clientCapabilities"
	MetaServerInfo         = "io.modelcontextprotocol/serverInfo"
)

// PreferredVersions is newest-first. Select walks this list against the server's supportedVersions.
var PreferredVersions = []string{Era}

// RequestMeta builds the three required per-request _meta fields for a modern RPC.
func RequestMeta(ident model.Identity, version string) map[string]any {
	if version == "" {
		version = Era
	}
	name := ident.Name
	if name == "" {
		name = "mcpdev"
	}
	ver := ident.Version
	if ver == "" {
		ver = "0.1.0"
	}
	info := map[string]any{"name": name, "version": ver}
	if ident.Title != "" {
		info["title"] = ident.Title
	}
	return map[string]any{
		MetaProtocolVersion:    version,
		MetaClientInfo:         info,
		MetaClientCapabilities: map[string]any{},
	}
}

// DiscoverParams is the JSON-RPC params object for server/discover.
func DiscoverParams(ident model.Identity, version string) map[string]any {
	return map[string]any{"_meta": RequestMeta(ident, version)}
}
