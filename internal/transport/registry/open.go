package registry

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/transport/http/legacy"
	"github.com/phillarmonic/mcpdev/internal/transport/http/modern"
	"github.com/phillarmonic/mcpdev/internal/transport/stdio"
)

// Open selects a connector and constructs it. The registry does not store the Handle.
func (r *Registry) Open(ctx context.Context, req Request) (Handle, error) {
	sel, err := r.Select(req)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	f := r.factories[sel.Descriptor.Name]
	r.mu.RUnlock()
	if f == nil {
		return nil, errclass.Internal{Err: errMissingFactory(sel.Descriptor.Name)}
	}
	return f(ctx, req, sel)
}

func openStdio(ctx context.Context, req Request, sel Selection) (Handle, error) {
	spec := port.ProcSpec{
		Clock: req.Clock,
		Env:   req.Env,
		Dir:   req.Dir,
		Argv:  req.Command,
	}
	var (
		conn *stdio.Conn
		err  error
	)
	if req.Supervisor != nil {
		conn, err = stdio.OpenWith(ctx, req.Supervisor, spec)
	} else {
		conn, err = stdio.Open(ctx, spec)
	}
	if err != nil {
		return nil, err
	}
	return &stdioHandle{conn: conn, sel: sel, tracer: req.Tracer}, nil
}

func openHTTP(_ context.Context, req Request, sel Selection) (Handle, error) {
	switch sel.Constructed {
	case NameHTTP2025:
		conn, err := legacy.Dial(legacy.Config{
			Headers:       req.Headers,
			Clock:         req.Clock,
			HTTPClient:    req.HTTPClient,
			Endpoint:      req.URL,
			Origin:        req.Origin,
			CAFile:        req.CAFile,
			CertFile:      req.CertFile,
			KeyFile:       req.KeyFile,
			AllowInsecure: req.AllowInsecure,
			Tracer:        req.Tracer,
		})
		if err != nil {
			return nil, err
		}
		return &legacyHandle{conn: conn, sel: sel, tracer: req.Tracer}, nil
	case NameHTTP2026:
		conn, err := modern.Dial(modern.Config{
			Headers:       req.Headers,
			HTTPClient:    req.HTTPClient,
			Endpoint:      req.URL,
			Origin:        req.Origin,
			CAFile:        req.CAFile,
			CertFile:      req.CertFile,
			KeyFile:       req.KeyFile,
			AllowInsecure: req.AllowInsecure,
			Tracer:        req.Tracer,
		})
		if err != nil {
			return nil, err
		}
		return &modernHandle{conn: conn, sel: sel, tracer: req.Tracer}, nil
	default:
		return nil, errclass.Internal{Err: errUnconstructedHTTP(sel.Constructed)}
	}
}
