package protocol

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func convertTool(t *mcp.Tool) model.Tool {
	return model.Tool{
		Name:         t.Name,
		Title:        t.Title,
		Description:  t.Description,
		InputSchema:  marshalSchema(t.InputSchema),
		OutputSchema: marshalSchema(t.OutputSchema),
		Meta:         marshalSchema(map[string]any(t.Meta)),
		Icons:        convertIcons(t.Icons),
		Annotations:  convertToolAnnotations(t.Annotations),
	}
}

func convertPrompt(p *mcp.Prompt) model.Prompt {
	out := model.Prompt{
		Name:        p.Name,
		Title:       p.Title,
		Description: p.Description,
		Meta:        marshalSchema(map[string]any(p.Meta)),
		Icons:       convertIcons(p.Icons),
	}
	for _, arg := range p.Arguments {
		if arg == nil {
			continue
		}
		out.Arguments = append(out.Arguments, model.PromptArgument{
			Name:        arg.Name,
			Title:       arg.Title,
			Description: arg.Description,
			Required:    arg.Required,
		})
	}
	return out
}

func convertResource(r *mcp.Resource) model.Resource {
	return model.Resource{
		URI:         r.URI,
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		MIMEType:    r.MIMEType,
		Size:        r.Size,
		Meta:        marshalSchema(map[string]any(r.Meta)),
		Icons:       convertIcons(r.Icons),
		Annotations: convertAnnotations(r.Annotations),
	}
}

func convertTemplate(t *mcp.ResourceTemplate) model.ResourceTemplate {
	return model.ResourceTemplate{
		URITemplate: t.URITemplate,
		Name:        t.Name,
		Title:       t.Title,
		Description: t.Description,
		MIMEType:    t.MIMEType,
		Meta:        marshalSchema(map[string]any(t.Meta)),
		Icons:       convertIcons(t.Icons),
		Annotations: convertAnnotations(t.Annotations),
	}
}

func convertIcons(icons []mcp.Icon) []model.Icon {
	if len(icons) == 0 {
		return nil
	}
	out := make([]model.Icon, 0, len(icons))
	for _, icon := range icons {
		out = append(out, model.Icon{
			Source:   icon.Source,
			MIMEType: icon.MIMEType,
			Sizes:    append([]string(nil), icon.Sizes...),
			Theme:    string(icon.Theme),
		})
	}
	return out
}

func convertToolAnnotations(a *mcp.ToolAnnotations) *model.ToolAnnotations {
	if a == nil {
		return nil
	}
	return &model.ToolAnnotations{
		Title:           a.Title,
		DestructiveHint: cloneBool(a.DestructiveHint),
		OpenWorldHint:   cloneBool(a.OpenWorldHint),
		IdempotentHint:  a.IdempotentHint,
		ReadOnlyHint:    a.ReadOnlyHint,
	}
}

func convertAnnotations(a *mcp.Annotations) *model.Annotations {
	if a == nil {
		return nil
	}
	out := &model.Annotations{LastModified: a.LastModified, Priority: a.Priority}
	for _, role := range a.Audience {
		out.Audience = append(out.Audience, string(role))
	}
	return out
}

func cloneBool(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
