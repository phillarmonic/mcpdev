package render

import (
	"fmt"
	"io"
	"strings"

	json "encoding/json/v2"

	diffapp "github.com/phillarmonic/mcpdev/internal/app/diff"
	"github.com/phillarmonic/mcpdev/internal/model"
)

//nolint:govet // JSON key order follows the CLI contract envelope
type diffEnvelope struct {
	OK         bool              `json:"ok"`
	Command    string            `json:"command"`
	From       sourceJSON        `json:"from"`
	To         sourceJSON        `json:"to"`
	Outcome    string            `json:"outcome"`
	Verdict    model.DiffVerdict `json:"verdict"`
	Changed    bool              `json:"changed"`
	Incomplete bool              `json:"incomplete"`
	Result     model.Diff        `json:"result"`
	Ignored    []model.Change    `json:"ignored"`
	Error      string            `json:"error,omitzero"`
}

type sourceJSON struct {
	Kind    string   `json:"kind"`
	Path    string   `json:"path,omitzero"`
	Label   string   `json:"label,omitzero"`
	Profile string   `json:"profile,omitzero"`
	URL     string   `json:"url,omitzero"`
	Command []string `json:"command,omitzero"`
}

// CommandErrorJSON writes a one-object failure document for commands without a richer envelope.
func CommandErrorJSON(w io.Writer, command string, err error) error {
	env := map[string]any{
		"ok":      false,
		"command": command,
		"error":   "",
	}
	if err != nil {
		env["error"] = err.Error()
	}
	b, marshalErr := json.Marshal(env, json.Deterministic(true))
	if marshalErr != nil {
		return marshalErr
	}
	_, writeErr := w.Write(append(b, '\n'))
	return writeErr
}

// DiffJSON writes one compact deterministic diff envelope to stdout.
func DiffJSON(w io.Writer, res diffapp.Result, err error) error {
	env := diffEnvelope{
		OK:         err == nil,
		Command:    "diff",
		From:       encodeSource(res.From),
		To:         encodeSource(res.To),
		Outcome:    res.Outcome,
		Verdict:    res.Diff.Verdict,
		Changed:    res.Diff.Changed,
		Incomplete: res.Outcome == diffapp.OutcomeIndeterminate,
		Result:     res.Diff,
		Ignored:    nonNilChanges(res.Ignored),
	}
	if env.Result.Changes == nil {
		env.Result.Changes = []model.Change{}
	}
	if err != nil {
		env.Error = err.Error()
		env.OK = false
	}
	b, marshalErr := json.Marshal(env, json.Deterministic(true))
	if marshalErr != nil {
		return marshalErr
	}
	_, writeErr := w.Write(append(b, '\n'))
	return writeErr
}

// DiffHuman writes a labeled unified-style summary of classified changes.
func DiffHuman(w io.Writer, res diffapp.Result) error {
	var b strings.Builder
	fmt.Fprintf(&b, "--- from: %s\n", sourceLabel(res.From))
	fmt.Fprintf(&b, "+++ to: %s\n", sourceLabel(res.To))
	fmt.Fprintf(&b, "verdict: %s\n", res.Diff.Verdict)
	fmt.Fprintf(&b, "outcome: %s\n", res.Outcome)
	if len(res.Ignored) > 0 {
		fmt.Fprintf(&b, "ignored: %d metadata path(s)\n", len(res.Ignored))
	}
	for _, ch := range res.Diff.Changes {
		mark := "~"
		switch ch.Category {
		case model.CategoryAddition:
			mark = "+"
		case model.CategoryRemoval:
			mark = "-"
		}
		fmt.Fprintf(&b, "%s %s (%s) %s\n", mark, ch.Path, ch.Severity, ch.Rationale)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func encodeSource(src diffapp.Source) sourceJSON {
	out := sourceJSON{Label: src.Label, Path: src.Path, Profile: src.Target.Profile, Command: src.Target.Command}
	switch {
	case src.Path != "":
		out.Kind = "file"
	case src.Live:
		out.Kind = src.Target.Kind
		if out.Kind == "" {
			if len(src.Target.Command) > 0 {
				out.Kind = "stdio"
			} else if src.Target.URL != "" {
				out.Kind = "http"
			}
		}
		out.URL = src.Target.URL
	default:
		out.Kind = "unknown"
	}
	return out
}

func sourceLabel(src diffapp.Source) string {
	if src.Label != "" {
		return src.Label
	}
	if src.Path != "" {
		return src.Path
	}
	return src.Target.Kind
}

func nonNilChanges(in []model.Change) []model.Change {
	if in == nil {
		return []model.Change{}
	}
	return in
}
