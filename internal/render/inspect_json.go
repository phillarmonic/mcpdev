package render

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"

	"github.com/phillarmonic/mcpdev/internal/app/inspect"
	"github.com/phillarmonic/mcpdev/internal/model"
)

//nolint:govet // JSON key order follows the CLI contract envelope
type inspectEnvelope struct {
	OK                bool       `json:"ok"`
	Command           string     `json:"command"`
	Target            targetJSON `json:"target"`
	Protocol          string     `json:"protocol"`
	Discover          string     `json:"discover"`
	Incomplete        bool       `json:"incomplete"`
	IncompleteReasons []string   `json:"incomplete_reasons"`
	Result            resultJSON `json:"result"`
	Error             string     `json:"error,omitzero"`
}

//nolint:govet // JSON key order is part of inspect snapshot input
type targetJSON struct {
	Kind    string            `json:"kind"`
	Profile string            `json:"profile,omitzero"`
	URL     string            `json:"url,omitzero"`
	Command []string          `json:"command,omitzero"`
	Headers map[string]string `json:"headers,omitzero"`
}

//nolint:govet // JSON key order is part of inspect snapshot input
type resultJSON struct {
	Server       serverJSON `json:"server"`
	Capabilities capJSON    `json:"capabilities"`
	Tools        *toolSec   `json:"tools,omitzero"`
	Prompts      *promptSec `json:"prompts,omitzero"`
	Resources    *resSec    `json:"resources,omitzero"`
	Templates    *tmplSec   `json:"templates,omitzero"`
}

type serverJSON struct {
	Name         string `json:"name,omitzero"`
	Title        string `json:"title,omitzero"`
	Version      string `json:"version,omitzero"`
	Instructions string `json:"instructions,omitzero"`
}

type capJSON struct {
	Tools     bool `json:"tools"`
	Prompts   bool `json:"prompts"`
	Resources bool `json:"resources"`
}

type sectionHead struct {
	Status        string   `json:"status"`
	StoppedCursor string   `json:"stopped_cursor,omitzero"`
	Reasons       []string `json:"reasons"`
	Count         int      `json:"count"`
}

type toolSec struct {
	Items []toolJSON `json:"items"`
	sectionHead
}

type promptSec struct {
	Items []promptJSON `json:"items"`
	sectionHead
}

type resSec struct {
	Items []resJSON `json:"items"`
	sectionHead
}

type tmplSec struct {
	Items []tmplJSON `json:"items"`
	sectionHead
}

type toolJSON struct {
	Name         string         `json:"name"`
	Title        string         `json:"title,omitzero"`
	Description  string         `json:"description,omitzero"`
	InputSchema  jsontext.Value `json:"input_schema,omitzero"`
	OutputSchema jsontext.Value `json:"output_schema,omitzero"`
}

type promptJSON struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitzero"`
	Description string `json:"description,omitzero"`
}

type resJSON struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitzero"`
	Title       string `json:"title,omitzero"`
	Description string `json:"description,omitzero"`
	MIMEType    string `json:"mime_type,omitzero"`
}

type tmplJSON struct {
	URITemplate string `json:"uri_template"`
	Name        string `json:"name,omitzero"`
	Title       string `json:"title,omitzero"`
	Description string `json:"description,omitzero"`
	MIMEType    string `json:"mime_type,omitzero"`
}

// InspectJSON writes one compact deterministic envelope to stdout.
func InspectJSON(w io.Writer, res inspect.Result, err error) error {
	env := inspectEnvelope{
		OK:                err == nil,
		Command:           "inspect",
		Target:            encodeTarget(res.Target),
		Protocol:          res.Inventory.ProtocolVersion,
		Discover:          string(res.Inventory.DiscoverState),
		Incomplete:        res.Inventory.Incomplete,
		IncompleteReasons: nonNil(res.Inventory.Reasons),
		Result:            encodeResult(res),
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

func encodeResult(res inspect.Result) resultJSON {
	inv := res.Inventory
	out := resultJSON{
		Server: serverJSON{
			Name:         inv.ServerName,
			Title:        inv.ServerTitle,
			Version:      inv.ServerVersion,
			Instructions: inv.Instructions,
		},
		Capabilities: capJSON{
			Tools:     inv.Capabilities.Tools,
			Prompts:   inv.Capabilities.Prompts,
			Resources: inv.Capabilities.Resources,
		},
	}
	if inv.Tools.Kind != "" {
		items := make([]toolJSON, 0, len(inv.Tools.Tools))
		for _, t := range inv.Tools.Tools {
			items = append(items, toolJSON{
				Name:         t.Name,
				Title:        t.Title,
				Description:  t.Description,
				InputSchema:  t.InputSchema,
				OutputSchema: t.OutputSchema,
			})
		}
		out.Tools = &toolSec{sectionHead: head(inv.Tools), Items: items}
	}
	if inv.Prompts.Kind != "" {
		items := make([]promptJSON, 0, len(inv.Prompts.Prompts))
		for _, p := range inv.Prompts.Prompts {
			items = append(items, promptJSON{Name: p.Name, Title: p.Title, Description: p.Description})
		}
		out.Prompts = &promptSec{sectionHead: head(inv.Prompts), Items: items}
	}
	if inv.Resources.Kind != "" {
		items := make([]resJSON, 0, len(inv.Resources.Resources))
		for _, r := range inv.Resources.Resources {
			items = append(items, resJSON{URI: r.URI, Name: r.Name, Title: r.Title, Description: r.Description, MIMEType: r.MIMEType})
		}
		out.Resources = &resSec{sectionHead: head(inv.Resources), Items: items}
	}
	if inv.Templates.Kind != "" {
		items := make([]tmplJSON, 0, len(inv.Templates.Templates))
		for _, r := range inv.Templates.Templates {
			items = append(items, tmplJSON{URITemplate: r.URITemplate, Name: r.Name, Title: r.Title, Description: r.Description, MIMEType: r.MIMEType})
		}
		out.Templates = &tmplSec{sectionHead: head(inv.Templates), Items: items}
	}
	return out
}

func head(sec model.CatalogSection) sectionHead {
	return sectionHead{
		Status:        string(sec.Status),
		StoppedCursor: sec.StoppedCursor,
		Reasons:       nonNil(sec.Reasons),
		Count:         sec.ItemCount,
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
