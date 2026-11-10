package diff

import (
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

type policyResult struct {
	ignored []model.Change
	failing []model.Change
	kept    model.Diff
}

func applyPolicy(raw model.Diff, ignore []string, failOn model.DiffSeverity) policyResult {
	kept := model.Diff{
		Verdict: raw.Verdict,
		Changed: raw.Changed,
		Changes: []model.Change{},
	}
	out := policyResult{ignored: []model.Change{}, failing: []model.Change{}}
	for _, ch := range raw.Changes {
		if ch.Severity == model.SeverityMetadata && pathIgnored(ch.Path, ignore) {
			out.ignored = append(out.ignored, ch)
			continue
		}
		kept.Changes = append(kept.Changes, ch)
		if severityRank(ch.Severity) >= failRank(failOn) {
			out.failing = append(out.failing, ch)
		}
	}
	switch {
	case raw.Verdict == model.VerdictIndeterminate:
		kept.Changed = false
	case len(kept.Changes) == 0:
		kept.Verdict = model.VerdictEqual
		kept.Changed = false
	default:
		kept.Verdict = model.VerdictChanged
		kept.Changed = true
	}
	out.kept = kept
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

func severityRank(s model.DiffSeverity) int {
	switch s {
	case model.SeverityMetadata:
		return 1
	case model.SeverityAdditive:
		return 2
	case model.SeverityBreaking:
		return 3
	case model.SeverityUnknown:
		return 4
	default:
		return 4
	}
}

func failRank(failOn model.DiffSeverity) int {
	switch failOn {
	case model.SeverityMetadata:
		return 1
	case model.SeverityBreaking:
		return 3
	default:
		return 2
	}
}

// ParseRenames maps kind:from=to identity pairs used by diff and snapshot steps.
func ParseRenames(vals []string) ([]model.Rename, error) {
	out := make([]model.Rename, 0, len(vals))
	for _, raw := range vals {
		kind, rest, ok := strings.Cut(raw, ":")
		if !ok {
			return nil, errclass.Usage{Msg: "--rename must be kind:from=to"}
		}
		from, to, ok := strings.Cut(rest, "=")
		if !ok || from == "" || to == "" {
			return nil, errclass.Usage{Msg: "--rename must be kind:from=to"}
		}
		switch model.PrimitiveKind(kind) {
		case model.KindTools, model.KindPrompts, model.KindResources, model.KindTemplates:
		default:
			return nil, errclass.Usage{Msg: "unknown --rename kind " + kind}
		}
		out = append(out, model.Rename{Kind: model.PrimitiveKind(kind), From: from, To: to})
	}
	return out, nil
}

// ParseFailOn maps --fail-on to a severity threshold.
// Default is additive: metadata-only diffs are below threshold.
func ParseFailOn(s string) (model.DiffSeverity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(model.SeverityAdditive):
		return model.SeverityAdditive, nil
	case string(model.SeverityMetadata):
		return model.SeverityMetadata, nil
	case string(model.SeverityBreaking):
		return model.SeverityBreaking, nil
	default:
		return "", errclass.Usage{Msg: "unknown --fail-on " + s}
	}
}
