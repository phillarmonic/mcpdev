package render

import (
	"fmt"
	"io"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/version"
)

//nolint:govet // JSON key order follows the CLI contract version envelope
type versionEnvelope struct {
	OK      bool         `json:"ok"`
	Command string       `json:"command"`
	Result  version.Info `json:"result"`
}

// VersionJSON writes one compact deterministic version envelope.
func VersionJSON(w io.Writer, info version.Info) error {
	b, err := json.Marshal(versionEnvelope{OK: true, Command: "version", Result: info}, json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// VersionHuman writes identity fields, one per line.
func VersionHuman(w io.Writer, info version.Info) error {
	_, err := fmt.Fprintf(w, "%s\ncommit %s\nbuild_time %s\nbuild_time_policy %s\nsnapshot_schema %d\n", info.Version, info.Commit, info.BuildTime, info.BuildTimePolicy, info.SnapshotSchema)
	return err
}
