// Package modern implements 2026-07-28 discovery helpers without the official SDK.
//
// Version selection, per-request _meta, and JSON-RPC error classification live here so
// protocol/negotiation can decide fallback without importing github.com/modelcontextprotocol/go-sdk.
package modern
