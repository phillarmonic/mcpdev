package version

import (
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/model"
)

const (
	defaultVersion = "0.1.1"
	unknown        = "unknown"

	// PolicyUnknown means this binary was not stamped (typical `go run` / local build).
	PolicyUnknown = "unknown"
	// PolicySourceDateEpoch means pack set build time from SOURCE_DATE_EPOCH.
	PolicySourceDateEpoch = "source_date_epoch"
	// PolicyGitCommit means pack set build time from the git committer timestamp.
	PolicyGitCommit = "git_commit"
	// PolicyVCSTime means Go embedded vcs.time (for example `go install` of a module version).
	PolicyVCSTime = "vcs_time"
)

// Stamped by -ldflags at pack time. Local builds keep these defaults and may
// fill commit and module version from runtime/debug.ReadBuildInfo.
var (
	Version         = defaultVersion
	Commit          = unknown
	BuildTime       = unknown
	BuildTimePolicy = PolicyUnknown
)

// Info is the machine-readable identity `mcpdev version` prints.
type Info struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	BuildTime       string `json:"build_time"`
	BuildTimePolicy string `json:"build_time_policy"`
	GOOS            string `json:"goos"`
	GOARCH          string `json:"goarch"`
	SnapshotSchema  int    `json:"snapshot_schema"`
}

// Current returns stamped values plus buildinfo fallbacks.
func Current() Info {
	info := Info{
		Version:         Version,
		Commit:          Commit,
		BuildTime:       BuildTime,
		BuildTimePolicy: BuildTimePolicy,
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		SnapshotSchema:  model.SnapshotSchemaVersion,
	}
	fillFromBuildInfo(&info)
	return info
}

func fillFromBuildInfo(info *Info) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Version == defaultVersion {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			info.Version = strings.TrimPrefix(v, "v")
		}
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if info.Commit == unknown && s.Value != "" {
				info.Commit = s.Value
			}
		case "vcs.time":
			if info.BuildTime == unknown && s.Value != "" {
				info.BuildTime = s.Value
				if info.BuildTimePolicy == PolicyUnknown {
					info.BuildTimePolicy = PolicyVCSTime
				}
			}
		}
	}
}
