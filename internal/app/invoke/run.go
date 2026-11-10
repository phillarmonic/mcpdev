package invoke

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Kind is a primitive operation.
type Kind string

const (
	KindCall   Kind = "call"
	KindPrompt Kind = "prompt"
	KindRead   Kind = "read"
)

// Request is one call, prompt, or read after flag resolution.
type Request struct {
	Stdin           io.Reader
	ArgPairs        []string
	Kind            Kind
	Name            string
	URI             string
	Template        string
	ArgsJSON        string
	ArgsFile        string
	SavePath        string
	Target          port.Target
	MaxBytes        int
	FailOnToolError bool
}

// Result is a primitive RPC payload plus connection metadata for renderers.
type Result struct {
	Diagnostics map[string]string
	Tool        *model.ToolResult
	Prompt      *model.PromptResult
	Resource    *model.ResourceBody
	Connector   string
	Constructed string
	Command     string
	Name        string
	URI         string
	SavePath    string
	Session     model.SessionInfo
	Target      port.Target
	SavedBytes  int
	MaxBytes    int
	Truncated   bool
}

// Run opens a session, invokes one primitive, optionally saves the body, and bounds stdout payload size.
func Run(ctx context.Context, conn port.Connector, req Request) (Result, error) {
	res := Result{
		Command:  string(req.Kind),
		Name:     req.Name,
		URI:      req.URI,
		SavePath: req.SavePath,
		Target:   req.Target,
		MaxBytes: req.MaxBytes,
	}
	if res.MaxBytes <= 0 {
		res.MaxBytes = DefaultMaxBytes
	}
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
	return OnSession(ctx, sess, req)
}

// OnSession runs call, prompt, or read on an already-open session.
func OnSession(ctx context.Context, sess port.Session, req Request) (Result, error) {
	res := Result{
		Command:  string(req.Kind),
		Name:     req.Name,
		URI:      req.URI,
		SavePath: req.SavePath,
		Target:   req.Target,
		MaxBytes: req.MaxBytes,
	}
	if res.MaxBytes <= 0 {
		res.MaxBytes = DefaultMaxBytes
	}
	res.Session = sess.Info()
	if meta, ok := sess.(sessionMeta); ok {
		res.Connector = meta.ConnectorName()
		res.Constructed = meta.Constructed()
		res.Diagnostics = meta.Diagnostics()
	}
	rpcCtx := ctx
	if req.Target.Timeout > 0 {
		var cancel context.CancelFunc
		rpcCtx, cancel = context.WithTimeout(ctx, req.Target.Timeout)
		defer cancel()
	}
	var err error
	switch req.Kind {
	case KindCall:
		err = runCall(rpcCtx, sess, req, &res)
	case KindPrompt:
		err = runPrompt(rpcCtx, sess, req, &res)
	case KindRead:
		err = runRead(rpcCtx, sess, req, &res)
	default:
		return res, errclass.Usage{Msg: "unknown invoke kind"}
	}
	var assert errclass.Assertion
	if err != nil && !errors.As(err, &assert) {
		return res, err
	}
	if saveErr := persistAndBound(&res); saveErr != nil {
		return res, saveErr
	}
	return res, err
}

func runCall(ctx context.Context, sess port.Session, req Request, res *Result) error {
	if req.Name == "" {
		return errclass.Usage{Msg: "call requires a tool name"}
	}
	if !sess.Info().Capabilities.Tools {
		return errclass.Protocol{Err: errors.New("call: tools capability is not advertised")}
	}
	args, err := loadToolArgs(req)
	if err != nil {
		return err
	}
	out, err := sess.CallTool(ctx, req.Name, args)
	if err != nil {
		return err
	}
	res.Tool = &out
	if out.IsError && req.FailOnToolError {
		return errclass.Assertion{Err: fmt.Errorf("tool %s returned isError", req.Name)}
	}
	return nil
}

func runPrompt(ctx context.Context, sess port.Session, req Request, res *Result) error {
	if req.Name == "" {
		return errclass.Usage{Msg: "prompt requires a prompt name"}
	}
	if !sess.Info().Capabilities.Prompts {
		return errclass.Protocol{Err: errors.New("prompt: prompts capability is not advertised")}
	}
	strArgs, err := loadStringArgs(req)
	if err != nil {
		return err
	}
	raw, err := jsonRawFromStrings(strArgs)
	if err != nil {
		return err
	}
	out, err := sess.GetPrompt(ctx, req.Name, raw)
	if err != nil {
		return err
	}
	res.Prompt = &out
	return nil
}

func runRead(ctx context.Context, sess port.Session, req Request, res *Result) error {
	if !sess.Info().Capabilities.Resources {
		return errclass.Protocol{Err: errors.New("read: resources capability is not advertised")}
	}
	uri := req.URI
	var err error
	if req.Template != "" {
		if uri != "" {
			return errclass.Usage{Msg: "read URI cannot be combined with --template"}
		}
		vars, loadErr := loadStringArgs(req)
		if loadErr != nil {
			return loadErr
		}
		uri, err = ExpandURITemplate(req.Template, vars)
		if err != nil {
			return err
		}
	}
	if uri == "" {
		return errclass.Usage{Msg: "read requires a URI or --template"}
	}
	res.URI = uri
	out, err := sess.ReadResource(ctx, uri)
	if err != nil {
		return err
	}
	res.Resource = &out
	return nil
}

type sessionMeta interface {
	ConnectorName() string
	Constructed() string
	Diagnostics() map[string]string
}
