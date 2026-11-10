package render

import (
	"io"
	"strings"
	"unicode/utf8"

	"github.com/phillarmonic/mcpdev/internal/app/invoke"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// InvokeHuman writes a labeled primitive result. Binary bodies are metadata only.
func InvokeHuman(w io.Writer, res invoke.Result) error {
	var b strings.Builder
	if err := writef(&b, "target: %s\n", humanTarget(res.Target)); err != nil {
		return err
	}
	if res.Session.ServerName != "" || res.Session.ProtocolVersion != "" {
		if err := writef(&b, "server: %s  protocol=%s  discover=%s\n", display(res.Session.ServerTitle, res.Session.ServerName), res.Session.ProtocolVersion, res.Session.DiscoverState); err != nil {
			return err
		}
	}
	if err := writef(&b, "command: %s\n", res.Command); err != nil {
		return err
	}
	switch res.Command {
	case "call":
		if err := writef(&b, "tool: %s\n", res.Name); err != nil {
			return err
		}
		if res.Tool != nil {
			if err := writef(&b, "isError: %t\n", res.Tool.IsError); err != nil {
				return err
			}
			if len(res.Tool.Structured) > 0 {
				if err := writef(&b, "structured: %s\n", string(res.Tool.Structured)); err != nil {
					return err
				}
			}
			for _, c := range res.Tool.Content {
				if err := writeContent(&b, c); err != nil {
					return err
				}
			}
		}
	case "prompt":
		if err := writef(&b, "prompt: %s\n", res.Name); err != nil {
			return err
		}
		if res.Prompt != nil {
			if res.Prompt.Description != "" {
				if err := writef(&b, "description: %s\n", res.Prompt.Description); err != nil {
					return err
				}
			}
			for _, m := range res.Prompt.Messages {
				if err := writef(&b, "message role=%s\n", m.Role); err != nil {
					return err
				}
				if err := writeContent(&b, m.Content); err != nil {
					return err
				}
			}
		}
	case "read":
		if err := writef(&b, "uri: %s\n", res.URI); err != nil {
			return err
		}
		if res.Resource != nil {
			for _, c := range res.Resource.Contents {
				if err := writeBody(&b, c); err != nil {
					return err
				}
			}
		}
	}
	if res.SavePath != "" {
		if err := writef(&b, "saved: %s  bytes=%d\n", res.SavePath, res.SavedBytes); err != nil {
			return err
		}
	}
	if res.Truncated {
		if err := writef(&b, "truncated: true  max_bytes=%d\n", res.MaxBytes); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeContent(b *strings.Builder, c model.Content) error {
	switch c.Kind {
	case model.ContentText:
		return writef(b, "text: %s\n", c.Text)
	case model.ContentImage, model.ContentAudio:
		return writef(b, "%s: mime=%s  bytes=%d  %s\n", c.Kind, c.MIMEType, len(c.Data), c.Description)
	case model.ContentResource, model.ContentResourceLink:
		return writef(b, "%s: uri=%s  mime=%s  %s\n", c.Kind, c.URI, c.MIMEType, c.Description)
	default:
		return writef(b, "%s: %s\n", c.Kind, c.Description)
	}
}

func writeBody(b *strings.Builder, c model.ResourceContents) error {
	if len(c.Blob) > 0 {
		return writef(b, "blob: uri=%s  mime=%s  bytes=%d\n", c.URI, c.MIMEType, len(c.Blob))
	}
	if !utf8.ValidString(c.Text) {
		return writef(b, "blob: uri=%s  mime=%s  bytes=%d (non-utf8 text)\n", c.URI, c.MIMEType, len(c.Text))
	}
	return writef(b, "text: %s\n", c.Text)
}
