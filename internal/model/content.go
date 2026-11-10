package model

import "encoding/json"

// ContentKind classifies a content block. Unrecognized kinds are kept explicit.
type ContentKind string

const (
	ContentText         ContentKind = "text"
	ContentImage        ContentKind = "image"
	ContentAudio        ContentKind = "audio"
	ContentResource     ContentKind = "resource"
	ContentResourceLink ContentKind = "resource_link"
	ContentUnrecognized ContentKind = "unrecognized"
)

// Content is an mcpdev content block.
type Content struct {
	Kind        ContentKind
	Text        string
	MIMEType    string
	URI         string
	Name        string
	Title       string
	Description string
	Data        []byte
}

// ToolResult is a tools/call payload. IsError is a successful RPC with a tool-level error.
type ToolResult struct {
	Structured json.RawMessage
	Content    []Content
	IsError    bool
}

// PromptResult is a prompts/get payload.
type PromptResult struct {
	Description string
	Messages    []PromptMessage
}

// PromptMessage is one prompts/get message.
type PromptMessage struct {
	Role    string
	Content Content
}

// ResourceBody is a resources/read payload.
type ResourceBody struct {
	Contents []ResourceContents
}

// ResourceContents is one resource body part.
type ResourceContents struct {
	URI      string
	MIMEType string
	Text     string
	Blob     []byte
}
