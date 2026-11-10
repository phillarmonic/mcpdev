package registry_test

import (
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/extension"
	"github.com/phillarmonic/mcpdev/internal/extension/fake"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/transport/registry"
)

func TestSelectDeterministic(t *testing.T) {
	r := registry.Builtins()
	cases := []struct {
		name       string
		wantName   string
		wantBuilt  string
		wantPolicy string
		req        registry.Request
	}{
		{
			name:       "command infers stdio",
			req:        registry.Request{Command: []string{"./server"}},
			wantName:   registry.NameStdio,
			wantBuilt:  registry.NameStdio,
			wantPolicy: registry.ProtocolAuto,
		},
		{
			name:       "url plus auto prefers modern construction",
			req:        registry.Request{URL: "http://127.0.0.1:8080/mcp"},
			wantName:   registry.NameHTTP,
			wantBuilt:  registry.NameHTTP2026,
			wantPolicy: registry.ProtocolAuto,
		},
		{
			name:       "url plus 2025 pins legacy",
			req:        registry.Request{URL: "http://127.0.0.1:8080/mcp", Protocol: string(model.Era20251125)},
			wantName:   registry.NameHTTP,
			wantBuilt:  registry.NameHTTP2025,
			wantPolicy: string(model.Era20251125),
		},
		{
			name:       "explicit modern name",
			req:        registry.Request{URL: "http://127.0.0.1:8080/mcp", Connector: registry.NameHTTP2026},
			wantName:   registry.NameHTTP2026,
			wantBuilt:  registry.NameHTTP2026,
			wantPolicy: registry.ProtocolAuto,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sel, err := r.Select(tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if sel.Descriptor.Name != tc.wantName || sel.Constructed != tc.wantBuilt || sel.EraPolicy != tc.wantPolicy {
				t.Fatalf("got name=%s constructed=%s policy=%s", sel.Descriptor.Name, sel.Constructed, sel.EraPolicy)
			}
		})
	}
}

func TestSelectRejectsConflicts(t *testing.T) {
	r := registry.Builtins()
	cases := []registry.Request{
		{Command: []string{"./s"}, URL: "http://127.0.0.1/mcp"},
		{Connector: registry.NameForbidden, URL: "http://127.0.0.1/mcp"},
		{URL: "http://127.0.0.1/mcp", Protocol: "not-a-date"},
		{Connector: registry.NameHTTP2026, URL: "http://127.0.0.1/mcp", Protocol: string(model.Era20251125)},
		{Connector: "no-such"},
	}
	for _, req := range cases {
		_, err := r.Select(req)
		if errclass.ExitCode(err) != errclass.ExitUsage {
			t.Fatalf("req %+v: exit %d err %v", req, errclass.ExitCode(err), err)
		}
	}
}

func TestDefaultEnumsOmitGRPC(t *testing.T) {
	r := registry.Builtins()
	got := registry.DefaultEnums(r)
	if !slices.Equal(got, []string{registry.NameStdio, registry.NameHTTP}) {
		t.Fatalf("enums %v", got)
	}
	if slices.Contains(r.Names(), registry.NameForbidden) {
		t.Fatal("grpc must not be a built-in")
	}
}

func TestOpenHTTPAndStdio(t *testing.T) {
	r := registry.Builtins()
	httpH, err := r.Open(t.Context(), registry.Request{URL: "http://127.0.0.1:9/mcp"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = httpH.Close(context.Background()) })
	d := httpH.Diagnostics()
	if d["constructed"] != registry.NameHTTP2026 || d["standard"] != "true" {
		t.Fatalf("http diag %+v", d)
	}

	legacyH, err := r.Open(t.Context(), registry.Request{
		URL:      "http://127.0.0.1:9/mcp",
		Protocol: string(model.Era20251125),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = legacyH.Close(context.Background()) })
	if legacyH.Diagnostics()["constructed"] != registry.NameHTTP2025 {
		t.Fatalf("legacy diag %+v", legacyH.Diagnostics())
	}

	stdioH, err := r.Open(t.Context(), registry.Request{
		Command:    []string{"./server"},
		Supervisor: stubSupervisor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stdioH.Close(context.Background()) })
	if stdioH.Name() != registry.NameStdio {
		t.Fatalf("stdio name %s", stdioH.Name())
	}
}

func TestFakeExtension(t *testing.T) {
	r := registry.Builtins()
	ad := fake.Connector{
		Label: "loopback-pipe",
		Sem: extension.Semantics{
			Cancellation: extension.CancelNone,
			Streaming:    extension.StreamNewlineJSON,
			Metadata:     extension.MetaCustom,
			Credentials:  extension.CredNone,
		},
	}
	if err := r.RegisterAdapter(ad); err != nil {
		t.Fatal(err)
	}
	h, err := r.Open(t.Context(), registry.Request{Connector: "loopback-pipe", URL: "pipe://local"})
	if err != nil {
		t.Fatal(err)
	}
	d := h.Diagnostics()
	if d["standard"] != "false" || d["endpoint"] != "pipe://local" || d["closed"] != "false" {
		t.Fatalf("diag %+v", d)
	}
	if err := h.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if h.Diagnostics()["closed"] != "true" {
		t.Fatalf("after close %+v", h.Diagnostics())
	}
}

func TestRegisterRejectsIncompleteExtension(t *testing.T) {
	r := registry.New()
	err := r.RegisterAdapter(fake.Connector{Label: "bad"})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("exit %d err %v", errclass.ExitCode(err), err)
	}
}

func TestRegisterRejectsStandardGRPC(t *testing.T) {
	r := registry.New()
	err := r.Register(registry.Descriptor{
		Name:     registry.NameForbidden,
		Kind:     registry.KindBuiltin,
		Standard: true,
		Semantics: extension.Semantics{
			Cancellation: extension.CancelNone,
			Streaming:    extension.StreamCustom,
			Metadata:     extension.MetaCustom,
			Credentials:  extension.CredNone,
		},
	}, func(context.Context, registry.Request, registry.Selection) (registry.Handle, error) {
		return nil, errors.New("unused factory")
	})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("exit %d err %v", errclass.ExitCode(err), err)
	}
}

type stubSupervisor struct{}

func (stubSupervisor) Start(context.Context, port.ProcSpec) (port.Proc, error) {
	rOut, wOut := io.Pipe()
	rIn, wIn := io.Pipe()
	return &stubProc{stdin: wIn, stdout: rOut, extra: []io.Closer{wOut, rIn}}, nil
}

type stubProc struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	extra  []io.Closer
}

func (p *stubProc) Stdin() io.WriteCloser  { return p.stdin }
func (p *stubProc) Stdout() io.ReadCloser  { return p.stdout }
func (p *stubProc) Wait() error            { return nil }
func (p *stubProc) Signal(os.Signal) error { return nil }
func (p *stubProc) Close(context.Context) error {
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	for _, c := range p.extra {
		_ = c.Close()
	}
	return nil
}
