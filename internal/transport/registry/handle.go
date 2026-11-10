package registry

import (
	"context"
	"io"
	"maps"

	"github.com/phillarmonic/mcpdev/internal/trace"
	"github.com/phillarmonic/mcpdev/internal/transport/http/legacy"
	"github.com/phillarmonic/mcpdev/internal/transport/http/modern"
	"github.com/phillarmonic/mcpdev/internal/transport/stdio"
)

type stdioHandle struct {
	conn   *stdio.Conn
	tracer *trace.Recorder
	sel    Selection
}

func (h *stdioHandle) Name() string         { return h.sel.Descriptor.Name }
func (h *stdioHandle) Selection() Selection { return h.sel }
func (h *stdioHandle) ByteStream(context.Context) (io.ReadCloser, io.WriteCloser) {
	s := h.conn.Stream()
	return trace.WrapReader(s.Reader, h.tracer, "stdio"), trace.WrapWriter(s.Writer, h.tracer, "stdio")
}

func (h *stdioHandle) Close(ctx context.Context) error {
	h.tracer.Stderr("stdio", h.conn.StderrText())
	return h.conn.Close(ctx)
}

func (h *stdioHandle) Diagnostics() map[string]string {
	return map[string]string{
		"connector":   h.sel.Descriptor.Name,
		"constructed": h.sel.Constructed,
		"standard":    "true",
		"stderr":      nonempty(h.conn.StderrText()),
	}
}

type legacyHandle struct {
	conn   *legacy.Conn
	tracer *trace.Recorder
	sel    Selection
}

func (h *legacyHandle) Name() string         { return h.sel.Descriptor.Name }
func (h *legacyHandle) Selection() Selection { return h.sel }
func (h *legacyHandle) ByteStream(ctx context.Context) (io.ReadCloser, io.WriteCloser) {
	return h.conn.ByteStream(ctx)
}

func (h *legacyHandle) Close(ctx context.Context) error {
	return h.conn.Close(ctx)
}

func (h *legacyHandle) Diagnostics() map[string]string {
	return httpDiag(h.sel, "2025-11-25")
}

type modernHandle struct {
	conn   *modern.Conn
	tracer *trace.Recorder
	sel    Selection
}

func (h *modernHandle) Name() string         { return h.sel.Descriptor.Name }
func (h *modernHandle) Selection() Selection { return h.sel }
func (h *modernHandle) ByteStream(ctx context.Context) (io.ReadCloser, io.WriteCloser) {
	return h.conn.ByteStream(ctx)
}

func (h *modernHandle) Close(ctx context.Context) error {
	return h.conn.Close(ctx)
}

func (h *modernHandle) Diagnostics() map[string]string {
	return httpDiag(h.sel, "2026-07-28")
}

func httpDiag(sel Selection, era string) map[string]string {
	return map[string]string{
		"connector":   sel.Descriptor.Name,
		"constructed": sel.Constructed,
		"era_policy":  sel.EraPolicy,
		"era":         era,
		"standard":    "true",
	}
}

func nonempty(s string) string {
	if s == "" {
		return "false"
	}
	return "true"
}

func cloneDiag(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	return maps.Clone(src)
}
