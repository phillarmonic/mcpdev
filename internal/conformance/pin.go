package conformance

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Pin is the checked-in npm package version and per-revision fixtures.
type Pin struct {
	Package   string     `json:"package"`
	Version   string     `json:"version"`
	Owner     string     `json:"owner"`
	Revisions []Revision `json:"revisions"`
}

// Revision maps one official requirement set to one HTTP fixture and baseline.
type Revision struct {
	Requirements string `json:"requirements"`
	Fixture      string `json:"fixture"`
	Baseline     string `json:"baseline"`
	Rationale    string `json:"rationale"`
}

// LoadPin reads testdata/conformance/pin.json from the module root.
func LoadPin(root string) (Pin, error) {
	path := filepath.Join(root, "testdata", "conformance", "pin.json")
	raw, err := os.ReadFile(path) //nolint:gosec // G304: pin.json is joined from the module root
	if err != nil {
		return Pin{}, err
	}
	var pin Pin
	if err := json.Unmarshal(raw, &pin); err != nil {
		return Pin{}, fmt.Errorf("conformance pin: %w", err)
	}
	if pin.Package == "" || pin.Version == "" {
		return Pin{}, errors.New("conformance pin: package and version are required")
	}
	if len(pin.Revisions) == 0 {
		return Pin{}, errors.New("conformance pin: at least one revision is required")
	}
	seenReq := make(map[string]struct{}, len(pin.Revisions))
	seenBase := make(map[string]struct{}, len(pin.Revisions))
	for i, rev := range pin.Revisions {
		if rev.Requirements == "" || rev.Fixture == "" || rev.Baseline == "" {
			return Pin{}, fmt.Errorf("conformance pin: revision %d needs requirements, fixture, and baseline", i)
		}
		if _, ok := seenReq[rev.Requirements]; ok {
			return Pin{}, fmt.Errorf("conformance pin: duplicate requirements %q", rev.Requirements)
		}
		seenReq[rev.Requirements] = struct{}{}
		if _, ok := seenBase[rev.Baseline]; ok {
			return Pin{}, fmt.Errorf("conformance pin: baseline %q is shared across revisions", rev.Baseline)
		}
		seenBase[rev.Baseline] = struct{}{}
	}
	return pin, nil
}

// RevisionByRequirements finds a pin entry. Unknown revisions are an error.
func (p Pin) RevisionByRequirements(name string) (Revision, error) {
	for _, rev := range p.Revisions {
		if rev.Requirements == name {
			return rev, nil
		}
	}
	return Revision{}, fmt.Errorf("conformance pin: unknown requirements %q", name)
}

// NPXPackage is the exact npm package@version to exec.
func (p Pin) NPXPackage() string {
	return p.Package + "@" + p.Version
}

// ServerArgs is the official runner argv after `npx --yes <package@version>`.
func ServerArgs(url, requirements, baseline, outDir string) []string {
	return []string{
		"server",
		"--url", url,
		"--requirements", requirements,
		"--expected-failures", baseline,
		"--output-dir", outDir,
	}
}
