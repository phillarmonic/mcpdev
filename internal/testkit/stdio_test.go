package testkit

import (
	"testing"
	"testing/synctest"
)

func TestNamedCatalogCoversBothEras(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("catalog is empty")
	}
	var saw2025, saw2026 bool
	for _, name := range names {
		d, err := Named(name)
		if err != nil {
			t.Fatal(err)
		}
		if d.Name != name {
			t.Fatalf("name: got %q want %q", d.Name, name)
		}
		switch d.Era {
		case Era20251125:
			saw2025 = true
		case Era20260728:
			saw2026 = true
		default:
			t.Fatalf("%s: unknown era %q", name, d.Era)
		}
		if d.Transport != TransportStdio && d.Transport != TransportHTTP {
			t.Fatalf("%s: unknown transport %q", name, d.Transport)
		}
	}
	if !saw2025 || !saw2026 {
		t.Fatalf("catalog must include both eras: 2025=%v 2026=%v", saw2025, saw2026)
	}
	if _, err := Named("does-not-exist"); err == nil {
		t.Fatal("unknown fixture must be an error, not a default")
	}
}

func TestLegacyStdioPrimitives(t *testing.T) {
	d, err := Named("legacy-stdio")
	if err != nil {
		t.Fatal(err)
	}
	lab, err := StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	closeLab(t, lab)
	if hsErr := handshakeStdio(lab); hsErr != nil {
		t.Fatal(hsErr)
	}
	assertList(t, lab, "tools/list", "tools")
	assertList(t, lab, "prompts/list", "prompts")
	assertList(t, lab, "resources/list", "resources")
	assertList(t, lab, "resources/templates/list", "resourceTemplates")

	echo, err := CallStdio(lab, Request{
		JSONRPC: jsonRPCVersion, ID: 10, Method: "tools/call",
		Params: mustMarshal(map[string]any{"name": "echo", "arguments": map[string]any{"text": "hi"}}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if echo.Error != nil {
		t.Fatal(echo.Error)
	}
	prompt, err := CallStdio(lab, Request{
		JSONRPC: jsonRPCVersion, ID: 11, Method: "prompts/get",
		Params: mustMarshal(map[string]any{"name": "greet", "arguments": map[string]any{"name": "lab"}}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Error != nil {
		t.Fatal(prompt.Error)
	}
	res, err := CallStdio(lab, Request{
		JSONRPC: jsonRPCVersion, ID: 12, Method: "resources/read",
		Params: mustMarshal(map[string]any{"uri": "fixture://stable/note"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Error != nil {
		t.Fatal(res.Error)
	}
}

func TestLegacyNotifyEmitsListChanged(t *testing.T) {
	d, err := Named("legacy-notify")
	if err != nil {
		t.Fatal(err)
	}
	lab, err := StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	closeLab(t, lab)
	if hsErr := handshakeStdio(lab); hsErr != nil {
		t.Fatal(hsErr)
	}
}

func TestLegacyPaginationDoesNotDropRemainder(t *testing.T) {
	d, err := Named("legacy-paginate")
	if err != nil {
		t.Fatal(err)
	}
	lab, err := StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	closeLab(t, lab)
	if hsErr := handshakeStdio(lab); hsErr != nil {
		t.Fatal(hsErr)
	}
	first, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "tools/list", Params: mustMarshal(map[string]any{})})
	if err != nil {
		t.Fatal(err)
	}
	result, _ := first.Result.(map[string]any)
	next, _ := result["nextCursor"].(string)
	if next == "" {
		t.Fatal("expected nextCursor on first page; incomplete collection must stay visible")
	}
	second, err := CallStdio(lab, Request{
		JSONRPC: jsonRPCVersion, ID: 3, Method: "tools/list",
		Params: mustMarshal(map[string]any{"cursor": next}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Error != nil {
		t.Fatal(second.Error)
	}
}

func TestModernStdioDiscoverAndListings(t *testing.T) {
	d, err := Named("modern-stdio")
	if err != nil {
		t.Fatal(err)
	}
	lab, err := StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	closeLab(t, lab)
	disc, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "server/discover", Params: modernParams(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if disc.Error != nil {
		t.Fatal(disc.Error)
	}
	initResp, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "initialize", Params: initParams()})
	if err != nil {
		t.Fatal(err)
	}
	if initResp.Error == nil || initResp.Error.Code != codeMethodNotFound {
		t.Fatalf("modern fixture must not accept initialize: %+v", initResp)
	}
	list, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 3, Method: "tools/list", Params: modernParams(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if list.Error != nil {
		t.Fatal(list.Error)
	}
}

func TestLegacyStdioRejectsWorkBeforeInitialized(t *testing.T) {
	d, err := Named("legacy-stdio")
	if err != nil {
		t.Fatal(err)
	}
	lab, err := StartStdio(d)
	if err != nil {
		t.Fatal(err)
	}
	closeLab(t, lab)
	list, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 2, Method: "tools/list", Params: mustMarshal(map[string]any{})})
	if err != nil {
		t.Fatal(err)
	}
	if list.Error == nil || list.Error.Code != codeInvalidRequest {
		t.Fatalf("expected not-initialized: %+v", list)
	}
}

func TestStdioCloseIsDeterministic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, err := Named("legacy-stdio")
		if err != nil {
			t.Fatal(err)
		}
		lab, err := StartStdio(d)
		if err != nil {
			t.Fatal(err)
		}
		if err := lab.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDelayHonorsCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, err := Named("delay-stdio")
		if err != nil {
			t.Fatal(err)
		}
		lab, err := StartStdio(d)
		if err != nil {
			t.Fatal(err)
		}
		closeLab(t, lab)
		type outcome struct {
			err  error
			resp Response
		}
		done := make(chan outcome, 1)
		go func() {
			resp, callErr := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: 1, Method: "initialize", Params: initParams()})
			done <- outcome{resp: resp, err: callErr}
		}()
		synctest.Wait()
		lab.Cancel()
		got := <-done
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.resp.Error == nil {
			t.Fatal("expected canceled RPC error after Cancel")
		}
	})
}

func assertList(t *testing.T, lab *StdioLab, method, key string) {
	t.Helper()
	resp, err := CallStdio(lab, Request{JSONRPC: jsonRPCVersion, ID: method, Method: method, Params: mustMarshal(map[string]any{})})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("%s: result %T", method, resp.Result)
	}
	if _, ok := m[key]; !ok {
		t.Fatalf("%s: missing %s in %v", method, key, m)
	}
}
