package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/app/inspect"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// InspectHuman writes a labeled catalog. Incomplete, unsupported, failed, and empty stay distinct.
func InspectHuman(w io.Writer, res inspect.Result) error {
	var b strings.Builder
	inv := res.Inventory
	if err := writef(&b, "target: %s\n", humanTarget(res.Target)); err != nil {
		return err
	}
	if err := writef(&b, "server: %s  name=%s  version=%s\n", display(inv.ServerTitle, inv.ServerName), inv.ServerName, inv.ServerVersion); err != nil {
		return err
	}
	if err := writef(&b, "protocol: %s  handshake=%s  discover=%s\n", inv.ProtocolVersion, inv.Handshake, inv.DiscoverState); err != nil {
		return err
	}
	if res.Connector != "" {
		if err := writef(&b, "connector: %s\n", res.Connector); err != nil {
			return err
		}
	}
	if inv.Incomplete {
		if err := writef(&b, "incomplete: %s\n", strings.Join(inv.Reasons, ", ")); err != nil {
			return err
		}
	}
	for _, sec := range []model.CatalogSection{inv.Tools, inv.Prompts, inv.Resources, inv.Templates} {
		if err := writeSection(&b, sec, res.Expand); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeSection(b *strings.Builder, sec model.CatalogSection, expand bool) error {
	if sec.Kind == "" {
		return nil
	}
	if err := writef(b, "%s: %d  status=%s", sec.Kind, sec.ItemCount, sec.Status); err != nil {
		return err
	}
	if len(sec.Reasons) > 0 {
		if err := writef(b, "  reasons=%s", strings.Join(sec.Reasons, ",")); err != nil {
			return err
		}
	}
	if sec.StoppedCursor != "" {
		if err := writef(b, "  cursor=%s", sec.StoppedCursor); err != nil {
			return err
		}
	}
	if err := b.WriteByte('\n'); err != nil {
		return err
	}
	switch sec.Kind {
	case model.KindTools:
		for _, t := range sec.Tools {
			if err := writef(b, "  %s  %s\n", t.Name, t.Description); err != nil {
				return err
			}
			if expand && len(t.InputSchema) > 0 {
				if err := writef(b, "    input_schema: %s\n", string(t.InputSchema)); err != nil {
					return err
				}
			}
		}
	case model.KindPrompts:
		for _, p := range sec.Prompts {
			if err := writef(b, "  %s  %s\n", p.Name, p.Description); err != nil {
				return err
			}
		}
	case model.KindResources:
		for _, r := range sec.Resources {
			if err := writef(b, "  %s  %s\n", r.URI, r.Description); err != nil {
				return err
			}
		}
	case model.KindTemplates:
		for _, r := range sec.Templates {
			if err := writef(b, "  %s  %s\n", r.URITemplate, r.Description); err != nil {
				return err
			}
		}
	}
	return nil
}

func writef(b *strings.Builder, format string, args ...any) error {
	_, err := fmt.Fprintf(b, format, args...)
	return err
}

func display(title, name string) string {
	if title != "" {
		return title
	}
	return name
}
