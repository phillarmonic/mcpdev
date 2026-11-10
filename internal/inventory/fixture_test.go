package inventory_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/inventory"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/protocol"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestCollectLegacyPaginateFixture(t *testing.T) {
	lab := startNamed(t, "legacy-paginate")
	sess, err := protocol.Connect(context.Background(), protocol.Stream{
		Reader: lab.ClientOut,
		Writer: lab.ClientIn,
	}, protocol.Options{Identity: model.Identity{Name: "mcpdev-test", Version: "dev"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sess.Close(context.Background()); closeErr != nil {
			t.Errorf("close session: %v", closeErr)
		}
	})
	inv := inventory.Collect(context.Background(), sess.Info(), sess.Inventory(), inventory.Limits{})
	if inv.Incomplete {
		t.Fatalf("paginate fixture must finish: %+v", inv)
	}
	if inv.Tools.ItemCount != 6 || inv.Tools.Pages != 3 {
		t.Fatalf("tools: %+v", inv.Tools)
	}
	if inv.Tools.Tools[0].Name != "echo" || inv.Tools.Tools[0].Description == "" {
		t.Fatalf("preserved description: %+v", inv.Tools.Tools[0])
	}
	if len(inv.Tools.Tools[0].InputSchema) == 0 {
		t.Fatal("input schema missing")
	}
	if inv.Prompts.ItemCount != 1 || inv.Resources.ItemCount != 2 || inv.Templates.ItemCount != 1 {
		t.Fatalf("other families: prompts=%+v res=%+v tmpl=%+v", inv.Prompts, inv.Resources, inv.Templates)
	}
}

func startNamed(t *testing.T, name string) *testkit.StdioLab {
	t.Helper()
	d, err := testkit.Named(name)
	if err != nil {
		t.Fatal(err)
	}
	lab, err := testkit.StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := lab.Close(); closeErr != nil && !errors.Is(closeErr, io.ErrClosedPipe) && !errors.Is(closeErr, io.EOF) {
			t.Errorf("close fixture: %v", closeErr)
		}
	})
	return lab
}
