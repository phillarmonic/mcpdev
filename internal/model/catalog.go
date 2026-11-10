package model

import "encoding/json"

// PrimitiveKind selects a catalog family.
type PrimitiveKind string

const (
	KindTools     PrimitiveKind = "tools"
	KindPrompts   PrimitiveKind = "prompts"
	KindResources PrimitiveKind = "resources"
	KindTemplates PrimitiveKind = "templates"
)

// Page is a cursor into a paginated catalog.
type Page struct {
	Cursor string
}

// CatalogPage is one page of one primitive kind.
// Incomplete is true when the adapter could not represent every item.
type CatalogPage struct {
	Kind              PrimitiveKind
	NextCursor        string
	Tools             []Tool
	Prompts           []Prompt
	Resources         []Resource
	Templates         []ResourceTemplate
	IncompleteReasons []string
	Incomplete        bool
}

// Tool is an mcpdev tool descriptor.
type Tool struct {
	Annotations  *ToolAnnotations
	Name         string
	Title        string
	Description  string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	Meta         json.RawMessage
	Icons        []Icon
}

// ToolAnnotations are untrusted hints about tool behavior.
type ToolAnnotations struct {
	DestructiveHint *bool
	OpenWorldHint   *bool
	Title           string
	IdempotentHint  bool
	ReadOnlyHint    bool
}

// Prompt is an mcpdev prompt descriptor.
type Prompt struct {
	Meta        json.RawMessage
	Name        string
	Title       string
	Description string
	Arguments   []PromptArgument
	Icons       []Icon
}

// PromptArgument is a named prompt input.
type PromptArgument struct {
	Name        string
	Title       string
	Description string
	Required    bool
}

// Resource is an mcpdev resource descriptor.
type Resource struct {
	Annotations *Annotations
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
	Meta        json.RawMessage
	Icons       []Icon
	Size        int64
}

// ResourceTemplate is an mcpdev resource template descriptor.
type ResourceTemplate struct {
	Annotations *Annotations
	URITemplate string
	Name        string
	Title       string
	Description string
	MIMEType    string
	Meta        json.RawMessage
	Icons       []Icon
}

// Annotations are display and audience hints on resources and templates.
type Annotations struct {
	LastModified string
	Audience     []string
	Priority     float64
}

// Icon is a visual identifier on a catalog item.
type Icon struct {
	Source   string
	MIMEType string
	Theme    string
	Sizes    []string
}
