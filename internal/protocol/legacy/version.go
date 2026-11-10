package legacy

import (
	"fmt"
	"slices"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// HandshakeVersions are dated initialize-era protocol strings the SDK still speaks.
// 2026-07-28 is not a handshake version.
var HandshakeVersions = []string{
	"2025-11-25",
	"2025-06-18",
	"2025-03-26",
}

// Accept reports a protocol error when version is not a supported initialize-era date.
func Accept(version string) error {
	if slices.Contains(HandshakeVersions, version) {
		return nil
	}
	return errclass.Protocol{Err: fmt.Errorf("protocol: unsupported MCP version %q", version)}
}
