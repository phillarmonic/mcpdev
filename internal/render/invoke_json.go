package render

import (
	"encoding/base64"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/app/invoke"
	"github.com/phillarmonic/mcpdev/internal/model"
)

//nolint:govet // JSON key order follows the CLI contract envelope
type invokeEnvelope struct {
	OK                bool             `json:"ok"`
	Command           string           `json:"command"`
	Target            targetJSON       `json:"target"`
	Protocol          string           `json:"protocol"`
	Discover          string           `json:"discover"`
	Incomplete        bool             `json:"incomplete"`
	IncompleteReasons []string         `json:"incomplete_reasons"`
	Result            invokeResultJSON `json:"result"`
	Error             string           `json:"error,omitzero"`
}

type invokeResultJSON struct {
	Structured  jsontext.Value `json:"structured,omitzero"`
	Name        string         `json:"name,omitzero"`
	URI         string         `json:"uri,omitzero"`
	Description string         `json:"description,omitzero"`
	Saved       string         `json:"saved,omitzero"`
	Content     []contentJSON  `json:"content,omitzero"`
	Messages    []messageJSON  `json:"messages,omitzero"`
	Contents    []bodyJSON     `json:"contents,omitzero"`
	SavedBytes  int            `json:"saved_bytes,omitzero"`
	IsError     bool           `json:"isError,omitzero"`
	Truncated   bool           `json:"truncated,omitzero"`
}

type contentJSON struct {
	Kind        string `json:"kind"`
	Text        string `json:"text,omitzero"`
	MIMEType    string `json:"mime_type,omitzero"`
	URI         string `json:"uri,omitzero"`
	Name        string `json:"name,omitzero"`
	Title       string `json:"title,omitzero"`
	Description string `json:"description,omitzero"`
	DataB64     string `json:"data_base64,omitzero"`
	Bytes       int    `json:"bytes,omitzero"`
	Omitted     bool   `json:"omitted,omitzero"`
}

type messageJSON struct {
	Role    string      `json:"role"`
	Content contentJSON `json:"content"`
}

type bodyJSON struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mime_type,omitzero"`
	Text     string `json:"text,omitzero"`
	DataB64  string `json:"blob_base64,omitzero"`
	Bytes    int    `json:"bytes,omitzero"`
	Omitted  bool   `json:"omitted,omitzero"`
}

// InvokeJSON writes one compact deterministic envelope for call, prompt, or read.
func InvokeJSON(w io.Writer, res invoke.Result, err error) error {
	env := invokeEnvelope{
		OK:                err == nil,
		Command:           res.Command,
		Target:            encodeTarget(res.Target),
		Protocol:          res.Session.ProtocolVersion,
		Discover:          string(res.Session.DiscoverState),
		Incomplete:        false,
		IncompleteReasons: []string{},
		Result:            encodeInvokeResult(res),
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

func encodeInvokeResult(res invoke.Result) invokeResultJSON {
	out := invokeResultJSON{
		Name:       res.Name,
		URI:        res.URI,
		Saved:      res.SavePath,
		SavedBytes: res.SavedBytes,
		Truncated:  res.Truncated,
	}
	if res.Tool != nil {
		out.IsError = res.Tool.IsError
		out.Structured = res.Tool.Structured
		out.Content = encodeContents(res.Tool.Content)
	}
	if res.Prompt != nil {
		out.Description = res.Prompt.Description
		for _, m := range res.Prompt.Messages {
			out.Messages = append(out.Messages, messageJSON{Role: m.Role, Content: encodeContent(m.Content)})
		}
	}
	if res.Resource != nil {
		for _, c := range res.Resource.Contents {
			out.Contents = append(out.Contents, encodeBody(c))
		}
	}
	return out
}

func encodeContents(items []model.Content) []contentJSON {
	out := make([]contentJSON, 0, len(items))
	for _, c := range items {
		out = append(out, encodeContent(c))
	}
	return out
}

func encodeContent(c model.Content) contentJSON {
	out := contentJSON{
		Kind:        string(c.Kind),
		Text:        c.Text,
		MIMEType:    c.MIMEType,
		URI:         c.URI,
		Name:        c.Name,
		Title:       c.Title,
		Description: c.Description,
	}
	if len(c.Data) > 0 {
		out.Bytes = len(c.Data)
		out.DataB64 = b64(c.Data)
	} else if c.Description != "" && c.Kind != model.ContentText && c.Kind != model.ContentUnrecognized {
		out.Omitted = true
	}
	return out
}

func encodeBody(c model.ResourceContents) bodyJSON {
	out := bodyJSON{URI: c.URI, MIMEType: c.MIMEType, Text: c.Text}
	if len(c.Blob) > 0 {
		out.Bytes = len(c.Blob)
		out.DataB64 = b64(c.Blob)
	} else if c.Text != "" && len(c.Blob) == 0 && startsOmitted(c.Text) {
		out.Omitted = true
	}
	return out
}

func startsOmitted(s string) bool {
	return strings.HasPrefix(s, "omitted")
}

func b64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
