package render

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	json "encoding/json/v2"

	apptest "github.com/phillarmonic/mcpdev/internal/app/test"
	"github.com/phillarmonic/mcpdev/internal/scenario/runner"
)

//nolint:govet // JSON key order follows the CLI contract envelope
type testEnvelope struct {
	OK                bool           `json:"ok"`
	Command           string         `json:"command"`
	Incomplete        bool           `json:"incomplete"`
	IncompleteReasons []string       `json:"incomplete_reasons"`
	Result            apptest.Result `json:"result"`
	Error             string         `json:"error,omitzero"`
}

// TestJSON writes one compact deterministic test envelope.
func TestJSON(w io.Writer, res apptest.Result, err error) error {
	env := testEnvelope{
		OK:                err == nil,
		Command:           "test",
		Incomplete:        res.Incomplete,
		IncompleteReasons: incompleteReasons(res),
		Result:            res,
	}
	if env.Result.Suites == nil {
		env.Result.Suites = []runner.Report{}
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

// TestHuman writes per-suite and per-case lines, then the failed or skipped
// assertions for any non-passing case.
func TestHuman(w io.Writer, res apptest.Result) error {
	for _, s := range res.Suites {
		if _, err := fmt.Fprintf(w, "%s %s (%s)\n", strings.ToUpper(s.Status), s.Name, s.File); err != nil {
			return err
		}
		for _, c := range s.Cases {
			line := fmt.Sprintf("  %s %s %s", strings.ToUpper(c.Status), c.Name, time.Duration(c.DurationMS)*time.Millisecond)
			if c.Message != "" {
				line += " " + c.Message
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
			if c.Status == "pass" {
				continue
			}
			if err := writeAssertionDetails(w, c.Steps); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeAssertionDetails prints one indented line per failing or skipped
// assertion so the default human output names the path and the mismatch.
func writeAssertionDetails(w io.Writer, steps []runner.StepReport) error {
	for _, st := range steps {
		for _, a := range st.Assertions {
			if a.Status == "pass" {
				continue
			}
			head := strings.TrimSpace(a.Kind + " " + a.Path)
			if head == "" {
				head = "assertion"
			}
			detail := a.Diff
			if detail == "" {
				detail = a.Message
			}
			if detail != "" {
				head += ": " + truncateRunes(detail, 300)
			}
			if _, err := fmt.Fprintf(w, "    %s\n", head); err != nil {
				return err
			}
		}
	}
	return nil
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

type junitSuites struct {
	XMLName xml.Name     `xml:"testsuites"`
	Suites  []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name     string      `xml:"name,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []junitCase `xml:"testcase"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Errors   int         `xml:"errors,attr"`
	Skipped  int         `xml:"skipped,attr"`
}

type junitCase struct {
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitFailure `xml:"error,omitempty"`
	Skipped   *junitFailure `xml:"skipped,omitempty"`
	Classname string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
}

type junitFailure struct {
	Type    string `xml:"type,attr,omitempty"`
	Message string `xml:"message,attr,omitempty"`
	Body    string `xml:",chardata"`
}

// TestJUnit writes a JUnit XML document distinguishing assertion, connection, protocol, and cleanup.
func TestJUnit(w io.Writer, res apptest.Result) error {
	out := junitSuites{}
	for _, s := range res.Suites {
		js := junitSuite{Name: s.Name, Tests: len(s.Cases), Time: fmt.Sprintf("%.3f", float64(s.DurationMS)/1000)}
		for _, c := range s.Cases {
			jc := junitCase{Classname: s.Name, Name: c.Name, Time: fmt.Sprintf("%.3f", float64(c.DurationMS)/1000)}
			switch {
			case c.Status == "pass":
			case c.Status == "skip":
				js.Skipped++
				jc.Skipped = &junitFailure{Message: c.Message}
			case c.Category == "assertion" || c.Category == "incomplete":
				js.Failures++
				jc.Failure = &junitFailure{Type: c.Category, Message: c.Message, Body: c.Message}
			default:
				js.Errors++
				jc.Error = &junitFailure{Type: c.Category, Message: c.Message, Body: c.Message}
			}
			js.Cases = append(js.Cases, jc)
		}
		out.Suites = append(out.Suites, js)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func incompleteReasons(res apptest.Result) []string {
	var out []string
	for _, s := range res.Suites {
		if s.Category == "incomplete" {
			out = append(out, "assertion_unavailable")
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}
