package watchrender

import (
	"io"
	"os"
	"strings"
)

// Mode selects color and in-place status for human watch output.
type Mode struct {
	Color   bool
	InPlace bool
}

// Detect enables color and in-place updates only for a TTY that honors NO_COLOR.
func Detect(w io.Writer, env []string) Mode {
	vars := envMap(env)
	if vars["NO_COLOR"] != "" {
		return Mode{}
	}
	if vars["TERM"] == "dumb" {
		return Mode{}
	}
	f, ok := w.(*os.File)
	if !ok {
		return Mode{}
	}
	st, err := f.Stat()
	if err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return Mode{}
	}
	return Mode{Color: true, InPlace: true}
}

func envMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, pair := range env {
		k, v, ok := strings.Cut(pair, "=")
		if ok {
			out[k] = v
		}
	}
	return out
}
