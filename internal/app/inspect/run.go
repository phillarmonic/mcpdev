package inspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/inventory"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Request is one inspect invocation after flag and profile resolution.
type Request struct {
	Names           []string
	Kinds           []model.PrimitiveKind
	Target          port.Target
	Expand          bool
	AllowIncomplete bool
}

// Result is inventory plus connection metadata for renderers.
type Result struct {
	Diagnostics  map[string]string
	Connector    string
	Constructed  string
	Inventory    model.Inventory
	Target       port.Target
	Expand       bool
	CollectedAll bool
}

// Run opens a session, collects the requested families, and maps incomplete work to class 60.
func Run(ctx context.Context, conn port.Connector, req Request) (Result, error) {
	connectCtx := ctx
	if req.Target.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, req.Target.ConnectTimeout)
		defer cancel()
	}
	sess, err := conn.Connect(connectCtx, req.Target)
	if err != nil {
		return Result{Target: req.Target, Expand: req.Expand}, err
	}
	defer func() { _ = sess.Close(ctx) }()
	return OnSession(ctx, sess, req)
}

// OnSession collects inventory on an already-open session (scenario cases share one connection).
func OnSession(ctx context.Context, sess port.Session, req Request) (Result, error) {
	rpcCtx := ctx
	if req.Target.Timeout > 0 {
		var cancel context.CancelFunc
		rpcCtx, cancel = context.WithTimeout(ctx, req.Target.Timeout)
		defer cancel()
	}
	inv := inventory.Collect(rpcCtx, sess.Info(), sess.Inventory(), inventory.Limits{}, req.Kinds...)
	inv = filterNames(inv, req.Names)
	res := Result{
		Inventory:    inv,
		Target:       req.Target,
		Connector:    req.Target.Kind,
		Expand:       req.Expand,
		CollectedAll: len(req.Kinds) == 0,
	}
	if meta, ok := sess.(sessionMeta); ok {
		res.Connector = meta.ConnectorName()
		res.Constructed = meta.Constructed()
		res.Diagnostics = meta.Diagnostics()
	}
	if inv.Incomplete && !req.AllowIncomplete {
		reason := strings.Join(inv.Reasons, ", ")
		if reason == "" {
			reason = "incomplete collection"
		}
		return res, errclass.Incomplete{Err: fmt.Errorf("inspect: %s", reason)}
	}
	return res, nil
}

type sessionMeta interface {
	ConnectorName() string
	Constructed() string
	Diagnostics() map[string]string
}
