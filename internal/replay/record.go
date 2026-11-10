package replay

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Recording wraps a live connector and writes a sanitized fixture on Close.
// Path is required; recording is never implied.
type Recording struct {
	Inner     port.Connector
	Path      string
	ExtraKeys []string
}

var _ port.Connector = Recording{}

func (r Recording) Connect(ctx context.Context, t port.Target) (port.Session, error) {
	if r.Path == "" || r.Path == "-" {
		return nil, errclass.Usage{Msg: "replay: --record requires an output path"}
	}
	if r.Inner == nil {
		return nil, errclass.Internal{Err: errString("replay: nil inner connector")}
	}
	sess, err := r.Inner.Connect(ctx, t)
	if err != nil {
		return nil, err
	}
	info := sess.Info()
	return &recorder{
		Session: sess,
		path:    r.Path,
		extra:   append([]string(nil), r.ExtraKeys...),
		doc: Document{
			SchemaVersion: SchemaVersion,
			Kind:          Kind,
			Meta: Meta{
				Transport: t.Kind,
				Era:       string(info.Era),
				Protocol:  info.ProtocolVersion,
			},
			Session: SessionSpec{
				Era:             string(info.Era),
				Handshake:       string(info.Handshake),
				DiscoverState:   string(info.DiscoverState),
				ProtocolVersion: info.ProtocolVersion,
				ServerName:      info.ServerName,
				ServerTitle:     info.ServerTitle,
				ServerVersion:   info.ServerVersion,
				SessionID:       info.SessionID,
				Instructions:    info.Instructions,
				FallbackReason:  info.FallbackReason,
				Fallback:        info.Fallback,
				Capabilities: map[string]bool{
					"tools":     info.Capabilities.Tools,
					"prompts":   info.Capabilities.Prompts,
					"resources": info.Capabilities.Resources,
				},
			},
		},
	}, nil
}

type recorder struct {
	port.Session
	path  string
	extra []string
	doc   Document
	mu    sync.Mutex
	wrote bool
}

func (r *recorder) Inventory() port.InventoryReader { return r }

func (r *recorder) List(ctx context.Context, kind model.PrimitiveKind, page model.Page) (model.CatalogPage, error) {
	params := map[string]any{}
	if page.Cursor != "" {
		params["cursor"] = page.Cursor
	}
	pageOut, err := r.Session.Inventory().List(ctx, kind, page)
	r.append(listMethod(kind), params, catalogPayload(pageOut), err)
	return pageOut, err
}

func (r *recorder) CallTool(ctx context.Context, name string, args json.RawMessage) (model.ToolResult, error) {
	params := callParams("name", name, "arguments", args)
	res, err := r.Session.CallTool(ctx, name, args)
	payload, _ := toolResultTo(res)
	r.append("tools/call", params, payload, err)
	return res, err
}

func (r *recorder) GetPrompt(ctx context.Context, name string, args json.RawMessage) (model.PromptResult, error) {
	params := callParams("name", name, "arguments", args)
	res, err := r.Session.GetPrompt(ctx, name, args)
	payload, _ := promptResultTo(res)
	r.append("prompts/get", params, payload, err)
	return res, err
}

func (r *recorder) ReadResource(ctx context.Context, uri string) (model.ResourceBody, error) {
	res, err := r.Session.ReadResource(ctx, uri)
	payload, _ := resourceBodyTo(res)
	r.append("resources/read", map[string]any{"uri": uri}, payload, err)
	return res, err
}

func (r *recorder) Close(ctx context.Context) error {
	err := r.Session.Close(ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wrote {
		return err
	}
	r.wrote = true
	werr := Save(r.path, r.doc, r.extra)
	if err != nil {
		return err
	}
	return werr
}

func (r *recorder) append(method string, params any, payload json.RawMessage, callErr error) {
	raw, _ := json.Marshal(params)
	d := Delivery{Kind: KindResult, Payload: payload}
	if callErr != nil {
		d = Delivery{
			Kind:     KindError,
			Message:  callErr.Error(),
			Category: errclass.Category(callErr),
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.doc.Steps = append(r.doc.Steps, Step{
		Expect:  &Expect{Method: method, Params: raw},
		Deliver: []Delivery{d},
	})
}

func catalogPayload(page model.CatalogPage) json.RawMessage {
	b, err := catalogTo(page)
	if err != nil {
		return nil
	}
	return b
}

func callParams(k1, v1, k2 string, args json.RawMessage) map[string]any {
	m := map[string]any{k1: v1}
	if len(args) > 0 {
		var v any
		if json.Unmarshal(args, &v) == nil {
			m[k2] = v
		}
	}
	return m
}

type errString string

func (e errString) Error() string { return string(e) }
