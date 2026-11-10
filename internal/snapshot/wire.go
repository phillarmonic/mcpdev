package snapshot

import (
	"encoding/json/jsontext"

	"github.com/phillarmonic/mcpdev/internal/model"
)

//nolint:govet // JSON key order is part of the snapshot schema
type wireDoc struct {
	SchemaVersion     int            `json:"schema_version"`
	Generator         wireGenerator  `json:"generator,omitzero"`
	CapturedAt        string         `json:"captured_at,omitzero"`
	Provenance        wireProvenance `json:"provenance,omitzero"`
	Protocol          wireProtocol   `json:"protocol"`
	Server            wireServer     `json:"server"`
	Capabilities      wireCaps       `json:"capabilities"`
	Incomplete        bool           `json:"incomplete"`
	IncompleteReasons []string       `json:"incomplete_reasons"`
	Tools             *wireSection   `json:"tools,omitzero"`
	Prompts           *wireSection   `json:"prompts,omitzero"`
	Resources         *wireSection   `json:"resources,omitzero"`
	Templates         *wireSection   `json:"templates,omitzero"`
}

type wireGenerator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type wireProvenance struct {
	Headers   map[string]string `json:"headers,omitzero"`
	Kind      string            `json:"kind,omitzero"`
	Profile   string            `json:"profile,omitzero"`
	URL       string            `json:"url,omitzero"`
	SessionID string            `json:"session_id,omitzero"`
	Command   []string          `json:"command,omitzero"`
}

type wireProtocol struct {
	Era       string `json:"era"`
	Version   string `json:"version"`
	Handshake string `json:"handshake,omitzero"`
	Discover  string `json:"discover"`
}

type wireServer struct {
	Name         string `json:"name,omitzero"`
	Title        string `json:"title,omitzero"`
	Version      string `json:"version,omitzero"`
	Instructions string `json:"instructions,omitzero"`
}

type wireCaps struct {
	Experimental         []string `json:"experimental"`
	Extensions           []string `json:"extensions"`
	Tools                bool     `json:"tools"`
	Prompts              bool     `json:"prompts"`
	Resources            bool     `json:"resources"`
	Completions          bool     `json:"completions"`
	Logging              bool     `json:"logging"`
	ToolsListChanged     bool     `json:"tools_list_changed"`
	PromptsListChanged   bool     `json:"prompts_list_changed"`
	ResourcesListChanged bool     `json:"resources_list_changed"`
	ResourceSubscribe    bool     `json:"resource_subscribe"`
}

type wireSection struct {
	Status        string     `json:"status"`
	StoppedCursor string     `json:"stopped_cursor,omitzero"`
	Reasons       []string   `json:"reasons"`
	Items         []wireItem `json:"items"`
	Pages         int        `json:"pages,omitzero"`
	Count         int        `json:"count"`
}

type wireItem struct {
	Annotations  *wireToolAnn   `json:"annotations,omitzero"`
	Hints        *wireResAnn    `json:"resource_annotations,omitzero"`
	InputSchema  jsontext.Value `json:"input_schema,omitzero"`
	OutputSchema jsontext.Value `json:"output_schema,omitzero"`
	Meta         jsontext.Value `json:"_meta,omitzero"`
	Name         string         `json:"name,omitzero"`
	Title        string         `json:"title,omitzero"`
	Description  string         `json:"description,omitzero"`
	URI          string         `json:"uri,omitzero"`
	URITemplate  string         `json:"uri_template,omitzero"`
	MIMEType     string         `json:"mime_type,omitzero"`
	Icons        []wireIcon     `json:"icons,omitzero"`
	Arguments    []wireArg      `json:"arguments,omitzero"`
	Size         int64          `json:"size,omitzero"`
}

type wireToolAnn struct {
	DestructiveHint *bool  `json:"destructive_hint,omitzero"`
	OpenWorldHint   *bool  `json:"open_world_hint,omitzero"`
	Title           string `json:"title,omitzero"`
	ReadOnlyHint    bool   `json:"read_only_hint,omitzero"`
	IdempotentHint  bool   `json:"idempotent_hint,omitzero"`
}

type wireResAnn struct {
	LastModified string   `json:"last_modified,omitzero"`
	Audience     []string `json:"audience,omitzero"`
	Priority     float64  `json:"priority,omitzero"`
}

type wireIcon struct {
	Source   string   `json:"src"`
	MIMEType string   `json:"mime_type,omitzero"`
	Theme    string   `json:"theme,omitzero"`
	Sizes    []string `json:"sizes,omitzero"`
}

type wireArg struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitzero"`
	Description string `json:"description,omitzero"`
	Required    bool   `json:"required"`
}

func emptyStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func wireCapsFrom(c model.ServerCapabilities) wireCaps {
	return wireCaps{
		Experimental:         emptyStrings(c.Experimental),
		Extensions:           emptyStrings(c.Extensions),
		Tools:                c.Tools,
		Prompts:              c.Prompts,
		Resources:            c.Resources,
		Completions:          c.Completions,
		Logging:              c.Logging,
		ToolsListChanged:     c.ToolsListChanged,
		PromptsListChanged:   c.PromptsListChanged,
		ResourcesListChanged: c.ResourcesListChanged,
		ResourceSubscribe:    c.ResourceSubscribe,
	}
}
