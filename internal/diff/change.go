package diff

import (
	jsonv2 "encoding/json/v2"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/model"
)

type report struct {
	changes []model.Change
}

func (r *report) add(path string, cat model.DiffCategory, sev model.DiffSeverity, before, after any, rationale string) {
	r.changes = append(r.changes, model.Change{
		Path:      path,
		Category:  cat,
		Severity:  sev,
		Before:    marshalValue(before),
		After:     marshalValue(after),
		Rationale: rationale,
	})
}

func (r *report) sort() {
	slices.SortStableFunc(r.changes, cmpChange)
}

func cmpChange(a, b model.Change) int {
	if n := strings.Compare(a.Path, b.Path); n != 0 {
		return n
	}
	if n := strings.Compare(string(a.Category), string(b.Category)); n != 0 {
		return n
	}
	if n := strings.Compare(string(a.Severity), string(b.Severity)); n != 0 {
		return n
	}
	return strings.Compare(a.Rationale, b.Rationale)
}

func marshalValue(v any) []byte {
	if v == nil {
		return nil
	}
	body, err := jsonv2.Marshal(v, jsonv2.Deterministic(true))
	if err != nil {
		return nil
	}
	return body
}
