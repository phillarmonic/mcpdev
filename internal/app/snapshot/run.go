package snapshot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Request is one snapshot command after flag and profile resolution.
type Request struct {
	Kinds           []model.PrimitiveKind
	Out             string
	Target          port.Target
	AllowIncomplete bool
}

// Result is the captured document plus where it was written.
type Result struct {
	Out      string
	Snapshot model.Snapshot
}

// Run connects, captures, and optionally writes an atomic snapshot file.
// An existing --out path is never replaced when collection is incomplete.
func Run(ctx context.Context, conn port.Connector, store port.SnapshotStore, req Request) (Result, error) {
	res := Result{}
	connectCtx := ctx
	if req.Target.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, req.Target.ConnectTimeout)
		defer cancel()
	}
	sess, err := conn.Connect(connectCtx, req.Target)
	if err != nil {
		return res, err
	}
	defer func() { _ = sess.Close(ctx) }()
	rpcCtx := ctx
	if req.Target.Timeout > 0 {
		var cancel context.CancelFunc
		rpcCtx, cancel = context.WithTimeout(ctx, req.Target.Timeout)
		defer cancel()
	}
	snap, err := store.Capture(rpcCtx, sess, req.Kinds)
	if err != nil {
		return res, err
	}
	res.Snapshot = snap
	incomplete := snap.Incomplete
	if incomplete && req.Out != "" && req.Out != "-" && fileExists(req.Out) {
		return res, incompleteErr(snap, "refusing to replace existing snapshot after incomplete collection")
	}
	if req.Out != "" && req.Out != "-" {
		if err := store.Write(rpcCtx, req.Out, snap); err != nil {
			return res, err
		}
		res.Out = req.Out
	}
	if incomplete && !req.AllowIncomplete {
		return res, incompleteErr(snap, "")
	}
	return res, nil
}

func incompleteErr(snap model.Snapshot, extra string) error {
	reason := strings.Join(snap.IncompleteReasons, ", ")
	if reason == "" {
		reason = "incomplete collection"
	}
	msg := "snapshot: " + reason
	if extra != "" {
		msg = fmt.Sprintf("snapshot: %s (%s)", extra, reason)
	}
	return errclass.Incomplete{Err: errors.New(msg)}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
