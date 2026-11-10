package registry

import (
	"context"
	"io"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/extension"
)

// RegisterAdapter adds an experimental custom connector. The adapter must not
// import internal/protocol; the registry only stores the Adapter interface.
func (r *Registry) RegisterAdapter(a extension.Adapter) error {
	if a == nil {
		return errclass.Usage{Msg: "connector: adapter is required"}
	}
	d := Descriptor{
		Name:      a.Name(),
		Kind:      KindExtension,
		Standard:  false,
		Semantics: a.Semantics(),
	}
	return r.Register(d, adapterFactory(a))
}

func adapterFactory(a extension.Adapter) Factory {
	return func(ctx context.Context, req Request, sel Selection) (Handle, error) {
		conn, err := a.Open(ctx, extension.Config{
			Headers:       req.Headers,
			Endpoint:      req.URL,
			AllowInsecure: req.AllowInsecure,
		})
		if err != nil {
			return nil, err
		}
		return &extHandle{conn: conn, sel: sel}, nil
	}
}

type extHandle struct {
	conn extension.Conn
	sel  Selection
}

func (h *extHandle) Name() string         { return h.sel.Descriptor.Name }
func (h *extHandle) Selection() Selection { return h.sel }
func (h *extHandle) ByteStream(context.Context) (io.ReadCloser, io.WriteCloser) {
	return h.conn.Reader(), h.conn.Writer()
}

func (h *extHandle) Diagnostics() map[string]string {
	out := cloneDiag(h.conn.Diagnostics())
	out["connector"] = h.sel.Descriptor.Name
	out["constructed"] = h.sel.Constructed
	out["standard"] = "false"
	return out
}

func (h *extHandle) Close(ctx context.Context) error {
	return h.conn.Close(ctx)
}
