package legacy

import (
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func TestAcceptKnownHandshakeVersions(t *testing.T) {
	for _, version := range HandshakeVersions {
		if err := Accept(version); err != nil {
			t.Fatalf("%s: %v", version, err)
		}
	}
}

func TestAcceptRejectsModernAndUnknown(t *testing.T) {
	for _, version := range []string{"", "2026-07-28", "1999-01-01", "2024-11-05"} {
		err := Accept(version)
		if errclass.ExitCode(err) != errclass.ExitProtocol {
			t.Fatalf("%q: class %d err=%v", version, errclass.ExitCode(err), err)
		}
	}
}
