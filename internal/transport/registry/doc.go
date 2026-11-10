// Package registry selects and constructs transport connectors from configuration.
//
// Built-in names are stdio and Streamable HTTP (with automatic or pinned era).
// Custom adapters register through internal/extension. gRPC is not a built-in
// and is not a standard MCP transport.
package registry
