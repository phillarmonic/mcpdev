package invoke

import (
	"errors"
	"fmt"
	"os"
	"unicode/utf8"

	stdjson "encoding/json"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// DefaultMaxBytes caps human and JSON stdout bodies unless --max-bytes is set.
const DefaultMaxBytes = 64 * 1024

func jsonRawFromStrings(m map[string]string) (stdjson.RawMessage, error) {
	if m == nil {
		m = map[string]string{}
	}
	b, err := json.Marshal(m, json.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: err}
	}
	return stdjson.RawMessage(b), nil
}

func persistAndBound(res *Result) error {
	if res.SavePath != "" {
		body, err := saveBytes(*res)
		if err != nil {
			return err
		}
		if err := os.WriteFile(res.SavePath, body, 0o600); err != nil {
			return errclass.Internal{Err: err}
		}
		res.SavedBytes = len(body)
	}
	limit := res.MaxBytes
	if limit <= 0 {
		limit = DefaultMaxBytes
	}
	if res.Tool != nil {
		truncated := false
		for i := range res.Tool.Content {
			if clipContent(&res.Tool.Content[i], limit) {
				truncated = true
			}
		}
		res.Truncated = truncated
	}
	if res.Prompt != nil {
		truncated := false
		for i := range res.Prompt.Messages {
			if clipContent(&res.Prompt.Messages[i].Content, limit) {
				truncated = true
			}
		}
		res.Truncated = truncated
	}
	if res.Resource != nil {
		truncated := false
		for i := range res.Resource.Contents {
			if clipResource(&res.Resource.Contents[i], limit) {
				truncated = true
			}
		}
		res.Truncated = truncated
	}
	return nil
}

func saveBytes(res Result) ([]byte, error) {
	switch {
	case res.Tool != nil:
		return marshalOrText(res.Tool.Content, res.Tool.Structured)
	case res.Prompt != nil:
		parts := make([]model.Content, 0, len(res.Prompt.Messages))
		for _, m := range res.Prompt.Messages {
			parts = append(parts, m.Content)
		}
		return marshalOrText(parts, nil)
	case res.Resource != nil:
		if len(res.Resource.Contents) == 1 {
			c := res.Resource.Contents[0]
			if len(c.Blob) > 0 {
				return c.Blob, nil
			}
			return []byte(c.Text), nil
		}
		b, err := json.Marshal(res.Resource.Contents, json.Deterministic(true))
		if err != nil {
			return nil, errclass.Internal{Err: err}
		}
		return b, nil
	default:
		return nil, errclass.Internal{Err: errors.New("invoke: nothing to save")}
	}
}

func marshalOrText(parts []model.Content, structured stdjson.RawMessage) ([]byte, error) {
	if len(structured) == 0 && len(parts) == 1 && parts[0].Kind == model.ContentText {
		return []byte(parts[0].Text), nil
	}
	if len(structured) == 0 && len(parts) == 1 && len(parts[0].Data) > 0 {
		return parts[0].Data, nil
	}
	b, err := json.Marshal(map[string]any{"structured": structured, "content": parts}, json.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: err}
	}
	return b, nil
}

func clipContent(c *model.Content, limit int) bool {
	if c.Text != "" && len(c.Text) > limit {
		c.Text = clipString(c.Text, limit)
		return true
	}
	if n := len(c.Data); n > limit {
		c.Description = fmt.Sprintf("omitted %d bytes", n)
		c.Data = nil
		return true
	}
	return false
}

func clipResource(c *model.ResourceContents, limit int) bool {
	if c.Text != "" && len(c.Text) > limit {
		c.Text = clipString(c.Text, limit)
		return true
	}
	if n := len(c.Blob); n > limit {
		c.Blob = nil
		c.Text = fmt.Sprintf("omitted %d bytes", n)
		return true
	}
	return false
}

func clipString(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	n := limit
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
