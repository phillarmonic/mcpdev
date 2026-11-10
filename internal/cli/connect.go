package cli

import (
	"context"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/protocol"
	"github.com/phillarmonic/mcpdev/internal/trace"
	"github.com/phillarmonic/mcpdev/internal/transport/registry"
	"github.com/phillarmonic/mcpdev/internal/version"
)

type boundSession struct {
	port.Session
	handle registry.Handle
}

func (s *boundSession) Close(ctx context.Context) error {
	err1 := s.Session.Close(ctx)
	err2 := s.handle.Close(ctx)
	if err1 != nil {
		return err1
	}
	return err2
}

func (s *boundSession) ConnectorName() string { return s.handle.Name() }

func (s *boundSession) Constructed() string { return s.handle.Selection().Constructed }

func (s *boundSession) Diagnostics() map[string]string { return s.handle.Diagnostics() }

type registryConnector struct {
	reg    *registry.Registry
	tracer *trace.Recorder
}

func (c registryConnector) Connect(ctx context.Context, t port.Target) (port.Session, error) {
	connector := t.Connector
	if connector == "" && t.Kind != "" && t.Kind != "stdio" && t.Kind != "http" {
		connector = t.Kind
	}
	req := registry.Request{
		Headers:       t.Headers,
		Env:           t.Env,
		Clock:         clock.Wall{},
		Connector:     connector,
		Protocol:      t.Protocol,
		URL:           t.URL,
		Origin:        t.Origin,
		Dir:           t.Dir,
		CAFile:        t.CAFile,
		CertFile:      t.CertFile,
		KeyFile:       t.KeyFile,
		Command:       t.Command,
		AllowInsecure: t.AllowInsecure,
		Tracer:        c.tracer,
	}
	h, err := c.reg.Open(ctx, req)
	if err != nil {
		return nil, err
	}
	r, w := h.ByteStream(ctx)
	sess, err := protocol.Connect(ctx, protocol.Stream{Reader: r, Writer: w}, protocol.Options{
		Identity:     model.Identity{Name: "mcpdev", Version: version.Version},
		PreferEra:    preferEra(t.Protocol),
		SkipDiscover: t.SkipDiscover,
		Transport:    t.Kind,
		Tracer:       c.tracer,
	})
	if err != nil {
		_ = h.Close(ctx)
		return nil, err
	}
	return &boundSession{Session: sess, handle: h}, nil
}

func preferEra(p string) model.ProtocolEra {
	switch p {
	case string(model.Era20251125):
		return model.Era20251125
	case string(model.Era20260728):
		return model.Era20260728
	default:
		return ""
	}
}

var _ port.Connector = registryConnector{}
