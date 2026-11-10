package protocol

import (
	"context"
	"errors"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/protocol/legacy"
	"github.com/phillarmonic/mcpdev/internal/trace"
)

// Options configure a client core session.
type Options struct {
	Tracer       *trace.Recorder
	Identity     model.Identity
	PreferEra    model.ProtocolEra
	Transport    string
	SkipDiscover bool
}

func (o Options) transportName() string {
	if o.Transport != "" {
		return o.Transport
	}
	return "stdio"
}

// Connect starts an MCP session over stream using the official SDK internally.
func Connect(ctx context.Context, stream Stream, opts Options) (port.Session, error) {
	if stream.Reader == nil || stream.Writer == nil {
		return nil, errclass.Usage{Msg: "protocol: stream reader and writer are required"}
	}
	ident := opts.Identity
	if ident.Name == "" {
		ident.Name = "mcpdev"
	}
	if ident.Version == "" {
		ident.Version = "0.1.1"
	}
	requested := requestedVersion(opts.PreferEra)
	var decision Decision
	switch {
	case opts.SkipDiscover:
		decision = Decision{Requested: requested, Fallback: true, Reason: "discover_skipped"}
	case opts.PreferEra != model.Era20251125:
		probed, probeErr := probeDiscover(ctx, stream, ident, requested, opts.Tracer, opts.transportName())
		decision = probed
		if probeErr != nil {
			return nil, probeErr
		}
	default:
		decision = Decision{Requested: requested}
	}
	if decision.Fallback {
		opts.Tracer.Fallback(opts.transportName(), decision.Reason)
	}
	client := mcp.NewClient(
		&mcp.Implementation{Name: ident.Name, Version: ident.Version, Title: ident.Title},
		&mcp.ClientOptions{
			// Non-nil empty capabilities avoid the SDK default that advertises deprecated roots.
			Capabilities: &mcp.ClientCapabilities{},
			Logger:       slog.New(slog.DiscardHandler),
		},
	)
	sessOpts := &mcp.ClientSessionOptions{ProtocolVersion: sdkProtocolVersion(opts.PreferEra, decision)}
	cs, err := client.Connect(ctx, &mcp.IOTransport{Reader: stream.Reader, Writer: stream.Writer}, sessOpts)
	if err != nil {
		if ctx.Err() != nil {
			opts.Tracer.Cancel(opts.transportName(), ctx.Err().Error())
		}
		return nil, mapError(err)
	}
	if ctx.Err() != nil {
		_ = cs.Close()
		opts.Tracer.Cancel(opts.transportName(), ctx.Err().Error())
		return nil, mapError(ctx.Err())
	}
	info, convErr := sessionInfo(ident, opts.PreferEra, decision, cs.ID(), cs.InitializeResult())
	if convErr != nil {
		_ = cs.Close()
		return nil, convErr
	}
	return newSession(cs, info), nil
}

func sessionInfo(ident model.Identity, preferred model.ProtocolEra, decision Decision, sessionID string, init *mcp.InitializeResult) (model.SessionInfo, error) {
	if init == nil {
		return model.SessionInfo{}, errclass.Protocol{Err: errors.New("protocol: missing initialize result")}
	}
	era, handshake, err := negotiatedEra(init.ProtocolVersion)
	if err != nil {
		return model.SessionInfo{}, err
	}
	info := model.SessionInfo{
		Identity:        ident,
		Era:             era,
		Handshake:       handshake,
		ProtocolVersion: init.ProtocolVersion,
		SessionID:       sessionID,
		Instructions:    init.Instructions,
		Capabilities:    convertCaps(init.Capabilities),
		DiscoverState:   discoverState(preferred, era, decision),
	}
	applyDecision(&info, decision)
	if init.ServerInfo != nil {
		info.ServerName = init.ServerInfo.Name
		info.ServerTitle = init.ServerInfo.Title
		info.ServerVersion = init.ServerInfo.Version
	}
	return info, nil
}

func negotiatedEra(version string) (model.ProtocolEra, model.HandshakeKind, error) {
	if model.ProtocolEra(version) == model.Era20260728 {
		return model.Era20260728, model.HandshakeDiscover, nil
	}
	if err := legacy.Accept(version); err != nil {
		return "", "", err
	}
	return model.Era20251125, model.HandshakeInitialize, nil
}

func discoverState(preferred, negotiated model.ProtocolEra, decision Decision) model.DiscoverState {
	if decision.Reason == "discover_skipped" {
		return model.DiscoverSkipped
	}
	if preferred == model.Era20251125 {
		return model.DiscoverNotApplicable
	}
	if decision.Fallback {
		return model.DiscoverSkipped
	}
	switch negotiated {
	case model.Era20260728:
		return model.DiscoverOK
	case model.Era20251125:
		return model.DiscoverSkipped
	default:
		return model.DiscoverFailed
	}
}
