package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/phillarmonic/mcpdev/internal/model"
)

type scriptReader struct {
	pages  map[model.PrimitiveKind][]model.CatalogPage
	errs   map[model.PrimitiveKind]error
	failOn map[model.PrimitiveKind]int
	calls  map[model.PrimitiveKind]int
}

func (r *scriptReader) List(_ context.Context, kind model.PrimitiveKind, page model.Page) (model.CatalogPage, error) {
	if r.calls == nil {
		r.calls = map[model.PrimitiveKind]int{}
	}
	n := r.calls[kind]
	r.calls[kind] = n + 1
	if err := r.errs[kind]; err != nil {
		failOn := r.failOn[kind]
		if failOn == 0 || failOn == n+1 {
			return model.CatalogPage{}, err
		}
	}
	seq := r.pages[kind]
	if n >= len(seq) {
		return model.CatalogPage{}, fmt.Errorf("unexpected extra list for %s cursor %q", kind, page.Cursor)
	}
	got := seq[n]
	if page.Cursor != expectedCursor(seq, n) {
		return model.CatalogPage{}, fmt.Errorf("%s: cursor %q want %q", kind, page.Cursor, expectedCursor(seq, n))
	}
	return got, nil
}

func expectedCursor(pages []model.CatalogPage, n int) string {
	if n == 0 {
		return ""
	}
	return pages[n-1].NextCursor
}

func toolPages(names ...[]string) []model.CatalogPage {
	out := make([]model.CatalogPage, 0, len(names))
	for i, page := range names {
		p := model.CatalogPage{Kind: model.KindTools}
		for _, name := range page {
			p.Tools = append(p.Tools, model.Tool{Name: name, Description: "d-" + name})
		}
		if i < len(names)-1 {
			p.NextCursor = fmt.Sprintf("c%d", i+1)
		}
		out = append(out, p)
	}
	return out
}

func fullInfo() model.SessionInfo {
	return model.SessionInfo{
		ServerName:      "lab",
		ServerTitle:     "Lab",
		ServerVersion:   "1",
		ProtocolVersion: string(model.Era20251125),
		Era:             model.Era20251125,
		Handshake:       model.HandshakeInitialize,
		DiscoverState:   model.DiscoverSkipped,
		Capabilities: model.ServerCapabilities{
			Tools:     true,
			Prompts:   true,
			Resources: true,
		},
	}
}

func emptyPages(kind model.PrimitiveKind) []model.CatalogPage {
	return []model.CatalogPage{{Kind: kind}}
}

var errList = errors.New("list exploded")
