package version

import (
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestCurrentReportsSnapshotSchema(t *testing.T) {
	info := Current()
	if info.SnapshotSchema != model.SnapshotSchemaVersion {
		t.Fatalf("schema %d", info.SnapshotSchema)
	}
	if info.Version == "" || info.Commit == "" || info.BuildTime == "" || info.BuildTimePolicy == "" {
		t.Fatalf("empty fields %+v", info)
	}
	if info.GOOS == "" || info.GOARCH == "" {
		t.Fatalf("platform %+v", info)
	}
}

func TestFillFromBuildInfoModuleVersion(t *testing.T) {
	info := Info{Version: defaultVersion, Commit: unknown, BuildTime: unknown, BuildTimePolicy: PolicyUnknown}
	fillFromBuildInfo(&info)
	if info.SnapshotSchema != 0 {
		t.Fatalf("fill must not set schema")
	}
}
