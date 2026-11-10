package inventory

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/model"
)

func TestCollectUnsupportedIsNotEmptyComplete(t *testing.T) {
	info := fullInfo()
	info.Capabilities.Prompts = false
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools:     emptyPages(model.KindTools),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), info, reader, Limits{})
	if inv.Prompts.Status != model.SectionUnsupported || inv.Prompts.Reasons[0] != model.ReasonCapabilityAbsent {
		t.Fatalf("prompts: %+v", inv.Prompts)
	}
	if inv.Tools.Status != model.SectionComplete || inv.Tools.ItemCount != 0 {
		t.Fatalf("empty tools must be complete: %+v", inv.Tools)
	}
	if reader.calls[model.KindPrompts] != 0 {
		t.Fatal("must not list an unadvertised family")
	}
	if inv.Incomplete {
		t.Fatalf("unsupported is finished work: %+v", inv)
	}
}

func TestCollectFailedSectionKeepsOtherFamilies(t *testing.T) {
	reader := &scriptReader{
		pages: map[model.PrimitiveKind][]model.CatalogPage{
			model.KindPrompts:   emptyPages(model.KindPrompts),
			model.KindResources: emptyPages(model.KindResources),
			model.KindTemplates: emptyPages(model.KindTemplates),
		},
		errs: map[model.PrimitiveKind]error{model.KindTools: errList},
	}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{})
	if inv.Tools.Status != model.SectionFailed || inv.Tools.Reasons[0] != model.ReasonListFailed {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if inv.Prompts.Status != model.SectionComplete {
		t.Fatalf("prompts: %+v", inv.Prompts)
	}
	if !inv.Incomplete || !contains(inv.Reasons, model.ReasonListFailed) {
		t.Fatalf("inventory: %+v", inv)
	}
}

func TestCollectDuplicateKeepsFirstAndRecordsReason(t *testing.T) {
	reader := &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
		model.KindTools: {
			{Kind: model.KindTools, Tools: []model.Tool{{Name: "echo", Description: "first"}}, NextCursor: "n"},
			{Kind: model.KindTools, Tools: []model.Tool{{Name: "echo", Description: "second"}, {Name: "fail"}}},
		},
		model.KindPrompts:   emptyPages(model.KindPrompts),
		model.KindResources: emptyPages(model.KindResources),
		model.KindTemplates: emptyPages(model.KindTemplates),
	}}
	inv := Collect(context.Background(), fullInfo(), reader, Limits{})
	if inv.Tools.ItemCount != 2 || inv.Tools.Tools[0].Description != "first" {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if !contains(inv.Tools.Reasons, model.ReasonDuplicateItem) {
		t.Fatalf("duplicate reason: %+v", inv.Tools.Reasons)
	}
	if inv.Tools.Status != model.SectionComplete || inv.Incomplete {
		t.Fatalf("duplicates are not dropped pages: %+v", inv)
	}
}

func TestCollectIsDeterministic(t *testing.T) {
	reader := func() *scriptReader {
		return &scriptReader{pages: map[model.PrimitiveKind][]model.CatalogPage{
			model.KindTools:     toolPages([]string{"zeta", "alpha"}),
			model.KindPrompts:   {{Kind: model.KindPrompts, Prompts: []model.Prompt{{Name: "greet"}}}},
			model.KindResources: {{Kind: model.KindResources, Resources: []model.Resource{{URI: "b://x", Name: "b"}, {URI: "a://x", Name: "a"}}}},
			model.KindTemplates: {{Kind: model.KindTemplates, Templates: []model.ResourceTemplate{{URITemplate: "t://{id}", Name: "t"}}}},
		}}
	}
	first := Collect(context.Background(), fullInfo(), reader(), Limits{})
	second := Collect(context.Background(), fullInfo(), reader(), Limits{})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not equal\n%+v\n%+v", first, second)
	}
	if first.Tools.Tools[0].Name != "alpha" || first.Resources.Resources[0].URI != "a://x" {
		t.Fatalf("order: %+v %+v", first.Tools.Tools, first.Resources.Resources)
	}
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}
