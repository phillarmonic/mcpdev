package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	jsonv2 "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Load reads and validates a replay document.
func Load(path string) (Document, error) {
	clean, err := validatedPath(path, false)
	if err != nil {
		return Document{}, err
	}
	body, err := os.ReadFile(clean) //nolint:gosec // G304: operator-supplied replay path after validation
	if err != nil {
		return Document{}, errclass.Usage{Err: err, Msg: "replay: read " + clean}
	}
	return Decode(body)
}

// Decode validates a replay document from bytes.
func Decode(body []byte) (Document, error) {
	var doc Document
	if err := jsonv2.Unmarshal(body, &doc); err != nil {
		return Document{}, errclass.Usage{Err: err, Msg: "replay: invalid JSON"}
	}
	if err := validate(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

// Save writes a sanitized document atomically with mode 0600.
func Save(path string, doc Document, extra []string) error {
	clean, err := validatedPath(path, true)
	if err != nil {
		return err
	}
	doc = Sanitize(doc, extra)
	if valErr := validate(doc); valErr != nil {
		return valErr
	}
	body, encErr := jsonv2.Marshal(doc, jsonv2.Deterministic(true))
	if encErr != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: encode: %w", encErr)}
	}
	body = append(body, '\n')
	dir := filepath.Dir(clean)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, ".mcpdev-replay-*.tmp")
	if err != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: temp %s: %w", dir, err)}
	}
	tmpName := tmp.Name()
	replaced := false
	defer func() {
		_ = tmp.Close()
		if !replaced {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: chmod: %w", err)}
	}
	if _, err := tmp.Write(body); err != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: write: %w", err)}
	}
	if err := tmp.Sync(); err != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: sync: %w", err)}
	}
	if err := tmp.Close(); err != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: close: %w", err)}
	}
	if err := os.Rename(tmpName, clean); err != nil {
		return errclass.Internal{Err: fmt.Errorf("replay: replace %s: %w", clean, err)}
	}
	replaced = true
	return nil
}

func validate(doc Document) error {
	if doc.Kind != Kind {
		return errclass.Usage{Msg: "replay: kind must be " + Kind}
	}
	if doc.SchemaVersion != SchemaVersion {
		return errclass.Usage{Msg: fmt.Sprintf("replay: unsupported schema_version %d", doc.SchemaVersion)}
	}
	if doc.Session.Era == "" && len(doc.Steps) == 0 {
		return errclass.Usage{Msg: "replay: session.era or steps are required"}
	}
	for i, st := range doc.Steps {
		if st.Expect != nil && st.Expect.Method == "" {
			return errclass.Usage{Msg: fmt.Sprintf("replay: step %d expect.method is required", i+1)}
		}
		if len(st.Deliver) == 0 {
			return errclass.Usage{Msg: fmt.Sprintf("replay: step %d needs at least one deliver", i+1)}
		}
		for j, d := range st.Deliver {
			switch d.Kind {
			case KindResult, KindError, KindNotification, KindSSE, KindMalformed, KindProcessExit:
			default:
				return errclass.Usage{Msg: "replay: step " + strconv.Itoa(i+1) + " deliver " + strconv.Itoa(j+1) + ": unknown kind " + d.Kind}
			}
			if d.DelayMS < 0 {
				return errclass.Usage{Msg: fmt.Sprintf("replay: step %d deliver %d: delay_ms must be >= 0", i+1, j+1)}
			}
		}
	}
	return nil
}

func validatedPath(path string, forWrite bool) (string, error) {
	if path == "" || path == "-" {
		return "", errclass.Usage{Msg: "replay: output path is required"}
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err == nil && info.IsDir() {
		return "", errclass.Usage{Msg: "replay: path is a directory"}
	}
	if !forWrite && err != nil {
		return "", errclass.Usage{Err: err, Msg: "replay: read " + clean}
	}
	return clean, nil
}
