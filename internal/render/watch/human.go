package watchrender

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/watch"
)

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	clearLine  = "\r\x1b[2K"
)

// Writer prints concise human status. Non-TTY output is append-only.
type Writer struct {
	Out     io.Writer
	Mode    Mode
	pending bool
}

// Write renders one Note. Running builds may occupy a TTY line until the next event.
func (w *Writer) Write(n watch.Note) error {
	line := formatHuman(n)
	if w.Mode.Color {
		line = colorize(n, line)
	}
	inPlace := w.Mode.InPlace && n.Kind == watch.KindBuild && n.State == "running"
	if w.pending && (!inPlace || !w.Mode.InPlace) {
		if _, err := io.WriteString(w.Out, "\n"); err != nil {
			return err
		}
		w.pending = false
	}
	if inPlace {
		_, err := io.WriteString(w.Out, clearLine+line)
		w.pending = true
		return err
	}
	_, err := fmt.Fprintln(w.Out, line)
	return err
}

// Flush ends an in-place status line so shutdown is a complete record.
func (w *Writer) Flush() error {
	if !w.pending {
		return nil
	}
	w.pending = false
	_, err := io.WriteString(w.Out, "\n")
	return err
}

func formatHuman(n watch.Note) string {
	ts := n.At.UTC().Format(time.RFC3339)
	var b strings.Builder
	b.WriteString(ts)
	b.WriteByte(' ')
	b.WriteString(n.ID)
	b.WriteByte(' ')
	b.WriteString(string(n.Kind))
	b.WriteByte(' ')
	b.WriteString(n.State)
	if n.Generation != 0 {
		_, _ = fmt.Fprintf(&b, " gen=%d", n.Generation)
	}
	if n.Class != "" {
		b.WriteString(" class=")
		b.WriteString(n.Class)
	}
	if n.Outcome != "" {
		b.WriteString(" outcome=")
		b.WriteString(n.Outcome)
	}
	if n.Path != "" {
		b.WriteByte(' ')
		b.WriteString(n.Path)
	}
	if n.Message != "" {
		b.WriteByte(' ')
		b.WriteString(n.Message)
	}
	if n.Incomplete {
		b.WriteString(" incomplete")
	}
	return b.String()
}

func colorize(n watch.Note, line string) string {
	code := ansiCyan
	switch {
	case n.Incomplete || n.State == "failed":
		code = ansiRed
	case n.State == "ok" || n.State == "ready" || n.State == "stopped":
		code = ansiGreen
	case n.Kind == watch.KindBuild || n.State == "running" || n.State == "starting":
		code = ansiYellow
	}
	return code + line + ansiReset
}
