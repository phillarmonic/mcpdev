package watchrender

import (
	"io"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/watch"
)

// JSONL writes one deterministic event object followed by a newline.
func JSONL(w io.Writer, n watch.Note) error {
	b, err := json.Marshal(n, json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}
