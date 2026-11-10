package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/protocol/modern"
	"github.com/phillarmonic/mcpdev/internal/trace"
)

const discoverMethod = "server/discover"

func probeDiscover(ctx context.Context, stream Stream, ident model.Identity, requested string, rec *trace.Recorder, transport string) (Decision, error) {
	version := requested
	if version == "" {
		version = modern.Era
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if err := ctx.Err(); err != nil {
			rec.Cancel(transport, err.Error())
			return Decision{Requested: requested, Reason: "timeout"}, mapError(err)
		}
		res, rpcErr, err := callDiscover(ctx, stream, ident, version)
		if err != nil {
			return Decision{Requested: requested, Fallback: true, Reason: "discover_failed"}, err
		}
		if rpcErr != nil {
			got := modern.ClassifyRPC(rpcErr.Code, rpcErr.Data)
			if got.Kind == modern.KindUnsupportedVersion {
				next := modern.Select(modern.PreferredVersions, got.Supported)
				if next != "" {
					rec.Retry(transport, string(got.Kind), attempt)
					version = next
					continue
				}
				return Decision{
					Requested:  requested,
					Discovered: got.Supported,
					Reason:     string(got.Kind),
				}, errclass.Protocol{Err: fmt.Errorf("protocol: no mutually supported modern version (requested %s, supported %v)", requested, got.Supported)}
			}
			if got.StayModern() {
				return Decision{
					Requested:  requested,
					Discovered: got.Supported,
					Reason:     string(got.Kind),
				}, errclass.Protocol{Err: fmt.Errorf("protocol: modern %s (%d)", got.Kind, got.Code)}
			}
			return Decision{Requested: requested, Fallback: true, Reason: modern.ReasonFallback(got)}, nil
		}
		negotiated := modern.Select(modern.PreferredVersions, res.SupportedVersions)
		if negotiated == "" {
			return Decision{
				Requested:  requested,
				Discovered: res.SupportedVersions,
				Fallback:   true,
				Reason:     "no_modern_overlap",
			}, nil
		}
		return Decision{
			Requested:  requested,
			Discovered: res.SupportedVersions,
			Negotiated: negotiated,
		}, nil
	}
	return Decision{
		Requested: requested,
		Reason:    string(modern.KindUnsupportedVersion),
	}, errclass.Protocol{Err: errors.New("protocol: server/discover version retry exhausted")}
}

type discoverWireResult struct {
	SupportedVersions []string `json:"supportedVersions"`
}

type wireRPCError struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func callDiscover(ctx context.Context, stream Stream, ident model.Identity, version string) (discoverWireResult, *wireRPCError, error) {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  discoverMethod,
		"params":  modern.DiscoverParams(ident, version),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return discoverWireResult{}, nil, errclass.Protocol{Err: err}
	}
	wrote := make(chan error, 1)
	go func() {
		_, err := stream.Writer.Write(append(body, '\n'))
		wrote <- err
	}()
	select {
	case <-ctx.Done():
		return discoverWireResult{}, nil, mapError(ctx.Err())
	case err := <-wrote:
		if err != nil {
			return discoverWireResult{}, nil, mapError(err)
		}
	}
	type outcome struct {
		err  error
		line []byte
	}
	read := make(chan outcome, 1)
	go func() {
		line, err := readJSONLine(stream.Reader)
		read <- outcome{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return discoverWireResult{}, nil, mapError(ctx.Err())
	case got := <-read:
		if got.err != nil {
			return discoverWireResult{}, nil, mapError(got.err)
		}
		var resp struct {
			Result *discoverWireResult `json:"result"`
			Error  *wireRPCError       `json:"error"`
		}
		if err := json.Unmarshal(got.line, &resp); err != nil {
			return discoverWireResult{}, nil, errclass.Protocol{Err: err}
		}
		if resp.Error != nil {
			return discoverWireResult{}, resp.Error, nil
		}
		if resp.Result == nil {
			return discoverWireResult{}, nil, errclass.Protocol{Err: errors.New("protocol: empty server/discover result")}
		}
		return *resp.Result, nil, nil
	}
}

func readJSONLine(r io.Reader) ([]byte, error) {
	var buf []byte
	one := make([]byte, 1)
	for {
		n, err := r.Read(one)
		if n > 0 {
			buf = append(buf, one[0])
			if one[0] == '\n' {
				return buf, nil
			}
		}
		if err != nil {
			if len(buf) > 0 && err == io.EOF {
				return buf, nil
			}
			return buf, err
		}
	}
}

func sdkProtocolVersion(prefer model.ProtocolEra, d Decision) string {
	if prefer == model.Era20251125 {
		return string(model.Era20251125)
	}
	if d.Fallback {
		return string(model.Era20251125)
	}
	if d.Negotiated != "" {
		return d.Negotiated
	}
	if prefer != "" {
		return string(prefer)
	}
	return modern.Era
}

func requestedVersion(prefer model.ProtocolEra) string {
	if prefer == model.Era20251125 {
		return string(model.Era20251125)
	}
	return modern.Era
}
