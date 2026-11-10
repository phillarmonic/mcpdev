package cycle

import (
	"strings"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func filterIgnored(raw model.Diff, ignore []string) model.Diff {
	out := model.Diff{Verdict: raw.Verdict, Changes: []model.Change{}}
	for _, ch := range raw.Changes {
		if ch.Severity == model.SeverityMetadata && pathIgnored(ch.Path, ignore) {
			continue
		}
		out.Changes = append(out.Changes, ch)
	}
	switch {
	case raw.Verdict == model.VerdictIndeterminate:
		out.Changed = false
	case len(out.Changes) == 0:
		out.Verdict = model.VerdictEqual
		out.Changed = false
	default:
		out.Verdict = model.VerdictChanged
		out.Changed = true
	}
	return out
}

func pathIgnored(path string, rules []string) bool {
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if path == rule || strings.HasPrefix(path, rule+"/") {
			return true
		}
	}
	return false
}
