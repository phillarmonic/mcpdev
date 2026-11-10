package inventory

import (
	"context"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestCollectFollowsOpaqueCursors(t *testing.T) {
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools:     toolPages([]string{"a", "b"}, []string{"c"}, []string{"d"}),
		model.KindPrompts:   emptyPages(model.KindPrompts),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{})
	if inv.Tools.Pages != 3 || inv.Tools.ItemCount != 4 {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if inv.Incomplete || inv.Tools.StoppedCursor != "" {
		t.Fatalf("finished walk: %+v", inv.Tools)
	}
}

func TestCollectRepeatedCursorStops(t *testing.T) {
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {
			{Kind: model.KindTools, Tools: []model.Tool{{Name: "a"}}, NextCursor: "loop"},
			{Kind: model.KindTools, Tools: []model.Tool{{Name: "b"}}, NextCursor: "loop"},
		},
		model.KindPrompts:   emptyPages(model.KindPrompts),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{})
	if inv.Tools.Status != model.SectionIncomplete || !contains(inv.Tools.Reasons, model.ReasonCursorLoop) {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if inv.Tools.ItemCount != 2 {
		t.Fatalf("kept pages before loop: %+v", inv.Tools)
	}
	if !inv.Incomplete {
		t.Fatal("loop must mark the inventory incomplete")
	}
}

func TestCollectPageLimitRecordsStoppedCursor(t *testing.T) {
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools:     toolPages([]string{"a"}, []string{"b"}, []string{"c"}),
		model.KindPrompts:   emptyPages(model.KindPrompts),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{MaxPages: 1, MaxItems: 100})
	if inv.Tools.Status != model.SectionIncomplete || inv.Tools.Pages != 1 {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if inv.Tools.StoppedCursor != "c1" || !contains(inv.Reasons, model.ReasonPaginationStopped) {
		t.Fatalf("stopped: %+v", inv.Tools)
	}
}

func TestCollectItemLimitDoesNotDropAccounting(t *testing.T) {
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools:     toolPages([]string{"a", "b", "c"}),
		model.KindPrompts:   emptyPages(model.KindPrompts),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{MaxItems: 2})
	if inv.Tools.ItemCount != 2 || inv.Tools.Tools[1].Name != "b" {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if !contains(inv.Tools.Reasons, model.ReasonItemLimit) {
		t.Fatalf("reasons: %v", inv.Tools.Reasons)
	}
}

func TestCollectPageIncompleteIsVisible(t *testing.T) {
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {{
			Kind: model.KindTools, Tools: []model.Tool{{Name: "a"}}, Incomplete: true,
		}},
		model.KindPrompts:   emptyPages(model.KindPrompts),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{})
	if inv.Tools.Status != model.SectionIncomplete || !contains(inv.Reasons, model.ReasonPageIncomplete) {
		t.Fatalf("tools: %+v", inv.Tools)
	}
}

func TestCollectPartialFailureKeepsPriorItems(t *testing.T) {
	reader := &scriptReader{
		pages: map[model.PrimitiveKind][]model.CatalogPage{
			model.KindTools:     toolPages([]string{"a"}, []string{"b"}),
			model.KindPrompts:   emptyPages(model.KindPrompts),
			model.KindResources: emptyPages(model.KindResources),
			model.KindTemplates: emptyPages(model.KindTemplates),
		},
		errs:   map[model.PrimitiveKind]error{model.KindTools: errList},
		failOn: map[model.PrimitiveKind]int{model.KindTools: 2},
	}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{})
	if inv.Tools.Status != model.SectionFailed || inv.Tools.ItemCount != 1 {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if inv.Tools.Tools[0].Name != "a" || !contains(inv.Reasons, model.ReasonListFailed) {
		t.Fatalf("partial: %+v", inv)
	}
}

func TestCollectCancelStopsRemainingPages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: toolPages([]string{"a"}),
	}}
	inv := Collect(ctx, fullInfo(), reader, Limits{})
	if inv.Tools.Status != model.SectionIncomplete || !contains(inv.Tools.Reasons, model.ReasonCollectionCanceled) {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if reader.calls[model.KindTools] != 0 {
		t.Fatal("canceled context must not start listing")
	}
}
