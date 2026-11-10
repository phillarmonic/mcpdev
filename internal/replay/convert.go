package replay

import (
	"encoding/json"

	"github.com/phillarmonic/mcpdev/internal/model"
)

type listWire struct {
	NextCursor        string         `json:"nextCursor,omitzero"`
	Tools             []toolWire     `json:"tools,omitzero"`
	Prompts           []promptWire   `json:"prompts,omitzero"`
	Resources         []resourceWire `json:"resources,omitzero"`
	Templates         []templateWire `json:"resourceTemplates,omitzero"`
	IncompleteReasons []string       `json:"incomplete_reasons,omitzero"`
	Incomplete        bool           `json:"incomplete,omitzero"`
}

type toolWire struct {
	Name         string          `json:"name"`
	Title        string          `json:"title,omitzero"`
	Description  string          `json:"description,omitzero"`
	InputSchema  json.RawMessage `json:"inputSchema,omitzero"`
	OutputSchema json.RawMessage `json:"outputSchema,omitzero"`
	Meta         json.RawMessage `json:"_meta,omitzero"`
}

type promptWire struct {
	Name        string          `json:"name"`
	Title       string          `json:"title,omitzero"`
	Description string          `json:"description,omitzero"`
	Meta        json.RawMessage `json:"_meta,omitzero"`
}

type resourceWire struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name,omitzero"`
	Title       string          `json:"title,omitzero"`
	Description string          `json:"description,omitzero"`
	MIMEType    string          `json:"mimeType,omitzero"`
	Meta        json.RawMessage `json:"_meta,omitzero"`
}

type templateWire struct {
	URITemplate string          `json:"uriTemplate"`
	Name        string          `json:"name,omitzero"`
	Title       string          `json:"title,omitzero"`
	MIMEType    string          `json:"mimeType,omitzero"`
	Meta        json.RawMessage `json:"_meta,omitzero"`
}

type toolResultWire struct {
	Structured json.RawMessage `json:"structuredContent,omitzero"`
	Content    []contentWire   `json:"content,omitzero"`
	IsError    bool            `json:"isError,omitzero"`
}

type contentWire struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitzero"`
	MIMEType string `json:"mimeType,omitzero"`
	URI      string `json:"uri,omitzero"`
}

type promptResultWire struct {
	Description string              `json:"description,omitzero"`
	Messages    []promptMessageWire `json:"messages,omitzero"`
}

type promptMessageWire struct {
	Role    string      `json:"role"`
	Content contentWire `json:"content"`
}

type resourceBodyWire struct {
	Contents []resourceContentsWire `json:"contents,omitzero"`
}

type resourceContentsWire struct {
	URI      string `json:"uri,omitzero"`
	MIMEType string `json:"mimeType,omitzero"`
	Text     string `json:"text,omitzero"`
}

func catalogFrom(kind model.PrimitiveKind, payload json.RawMessage) (model.CatalogPage, error) {
	var w listWire
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &w); err != nil {
			return model.CatalogPage{}, err
		}
	}
	page := model.CatalogPage{
		Kind:              kind,
		NextCursor:        w.NextCursor,
		Incomplete:        w.Incomplete,
		IncompleteReasons: w.IncompleteReasons,
	}
	for _, t := range w.Tools {
		page.Tools = append(page.Tools, model.Tool{
			Name: t.Name, Title: t.Title, Description: t.Description,
			InputSchema: t.InputSchema, OutputSchema: t.OutputSchema, Meta: t.Meta,
		})
	}
	for _, p := range w.Prompts {
		page.Prompts = append(page.Prompts, model.Prompt{
			Name: p.Name, Title: p.Title, Description: p.Description, Meta: p.Meta,
		})
	}
	for _, r := range w.Resources {
		page.Resources = append(page.Resources, model.Resource{
			URI: r.URI, Name: r.Name, Title: r.Title, Description: r.Description,
			MIMEType: r.MIMEType, Meta: r.Meta,
		})
	}
	for _, tmpl := range w.Templates {
		page.Templates = append(page.Templates, model.ResourceTemplate{
			URITemplate: tmpl.URITemplate, Name: tmpl.Name, Title: tmpl.Title,
			MIMEType: tmpl.MIMEType, Meta: tmpl.Meta,
		})
	}
	return page, nil
}

func catalogTo(page model.CatalogPage) (json.RawMessage, error) {
	w := listWire{
		NextCursor:        page.NextCursor,
		Incomplete:        page.Incomplete,
		IncompleteReasons: page.IncompleteReasons,
	}
	for _, t := range page.Tools {
		w.Tools = append(w.Tools, toolWire{
			Name: t.Name, Title: t.Title, Description: t.Description,
			InputSchema: t.InputSchema, OutputSchema: t.OutputSchema, Meta: t.Meta,
		})
	}
	for _, p := range page.Prompts {
		w.Prompts = append(w.Prompts, promptWire{
			Name: p.Name, Title: p.Title, Description: p.Description, Meta: p.Meta,
		})
	}
	for _, r := range page.Resources {
		w.Resources = append(w.Resources, resourceWire{
			URI: r.URI, Name: r.Name, Title: r.Title, Description: r.Description,
			MIMEType: r.MIMEType, Meta: r.Meta,
		})
	}
	for _, tmpl := range page.Templates {
		w.Templates = append(w.Templates, templateWire{
			URITemplate: tmpl.URITemplate, Name: tmpl.Name, Title: tmpl.Title,
			MIMEType: tmpl.MIMEType, Meta: tmpl.Meta,
		})
	}
	return json.Marshal(w)
}

func toolResultFrom(payload json.RawMessage) (model.ToolResult, error) {
	var w toolResultWire
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &w); err != nil {
			return model.ToolResult{}, err
		}
	}
	res := model.ToolResult{Structured: w.Structured, IsError: w.IsError}
	for _, c := range w.Content {
		res.Content = append(res.Content, contentFrom(c))
	}
	return res, nil
}

func toolResultTo(res model.ToolResult) (json.RawMessage, error) {
	w := toolResultWire{Structured: res.Structured, IsError: res.IsError}
	for _, c := range res.Content {
		w.Content = append(w.Content, contentTo(c))
	}
	return json.Marshal(w)
}

func promptResultFrom(payload json.RawMessage) (model.PromptResult, error) {
	var w promptResultWire
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &w); err != nil {
			return model.PromptResult{}, err
		}
	}
	res := model.PromptResult{Description: w.Description}
	for _, m := range w.Messages {
		res.Messages = append(res.Messages, model.PromptMessage{Role: m.Role, Content: contentFrom(m.Content)})
	}
	return res, nil
}

func promptResultTo(res model.PromptResult) (json.RawMessage, error) {
	w := promptResultWire{Description: res.Description}
	for _, m := range res.Messages {
		w.Messages = append(w.Messages, promptMessageWire{Role: m.Role, Content: contentTo(m.Content)})
	}
	return json.Marshal(w)
}

func resourceBodyFrom(payload json.RawMessage) (model.ResourceBody, error) {
	var w resourceBodyWire
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &w); err != nil {
			return model.ResourceBody{}, err
		}
	}
	var body model.ResourceBody
	for _, c := range w.Contents {
		body.Contents = append(body.Contents, model.ResourceContents{
			URI: c.URI, MIMEType: c.MIMEType, Text: c.Text,
		})
	}
	return body, nil
}

func resourceBodyTo(body model.ResourceBody) (json.RawMessage, error) {
	w := resourceBodyWire{}
	for _, c := range body.Contents {
		w.Contents = append(w.Contents, resourceContentsWire{
			URI: c.URI, MIMEType: c.MIMEType, Text: c.Text,
		})
	}
	return json.Marshal(w)
}

func contentFrom(c contentWire) model.Content {
	kind := model.ContentKind(c.Type)
	switch c.Type {
	case "text":
		kind = model.ContentText
	case "image":
		kind = model.ContentImage
	case "audio":
		kind = model.ContentAudio
	case "resource":
		kind = model.ContentResource
	case "resource_link":
		kind = model.ContentResourceLink
	default:
		if c.Type != "" {
			kind = model.ContentUnrecognized
		}
	}
	return model.Content{Kind: kind, Text: c.Text, MIMEType: c.MIMEType, URI: c.URI}
}

func contentTo(c model.Content) contentWire {
	typ := string(c.Kind)
	switch c.Kind {
	case model.ContentText:
		typ = "text"
	case model.ContentImage:
		typ = "image"
	case model.ContentAudio:
		typ = "audio"
	case model.ContentResource:
		typ = "resource"
	case model.ContentResourceLink:
		typ = "resource_link"
	}
	return contentWire{Type: typ, Text: c.Text, MIMEType: c.MIMEType, URI: c.URI}
}

func listMethod(kind model.PrimitiveKind) string {
	switch kind {
	case model.KindTools:
		return "tools/list"
	case model.KindPrompts:
		return "prompts/list"
	case model.KindResources:
		return "resources/list"
	case model.KindTemplates:
		return "resources/templates/list"
	default:
		return string(kind)
	}
}

func sessionInfo(spec SessionSpec) model.SessionInfo {
	era := model.ProtocolEra(spec.Era)
	ds := model.DiscoverState(spec.DiscoverState)
	if ds == "" {
		if era == model.Era20251125 {
			ds = model.DiscoverNotApplicable
		} else {
			ds = model.DiscoverOK
		}
	}
	hs := model.HandshakeKind(spec.Handshake)
	if hs == "" {
		if era == model.Era20260728 {
			hs = model.HandshakeDiscover
		} else {
			hs = model.HandshakeInitialize
		}
	}
	ver := spec.ProtocolVersion
	if ver == "" {
		ver = spec.Era
	}
	caps := spec.Capabilities
	info := model.SessionInfo{
		Era:             era,
		Handshake:       hs,
		DiscoverState:   ds,
		ProtocolVersion: ver,
		ServerName:      spec.ServerName,
		ServerTitle:     spec.ServerTitle,
		ServerVersion:   spec.ServerVersion,
		SessionID:       spec.SessionID,
		Instructions:    spec.Instructions,
		FallbackReason:  spec.FallbackReason,
		Fallback:        spec.Fallback,
		Capabilities: model.ServerCapabilities{
			Tools:     caps["tools"],
			Prompts:   caps["prompts"],
			Resources: caps["resources"],
		},
	}
	if len(caps) == 0 {
		info.Capabilities.Tools = true
		info.Capabilities.Prompts = true
		info.Capabilities.Resources = true
	}
	return info
}
