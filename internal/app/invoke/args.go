package invoke

import (
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	stdjson "encoding/json"

	json "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func loadJSONObject(req Request) (map[string]any, error) {
	raw, err := readArgsSource(req)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(bytesTrim(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, errclass.Usage{Err: err, Msg: "--args"}
		}
	}
	for _, pair := range req.ArgPairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || k == "" {
			return nil, errclass.Usage{Msg: "--arg must be KEY=JSON"}
		}
		var val any
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			return nil, errclass.Usage{Err: err, Msg: "--arg " + k}
		}
		out[k] = val
	}
	return out, nil
}

func loadToolArgs(req Request) (stdjson.RawMessage, error) {
	obj, err := loadJSONObject(req)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(obj, json.Deterministic(true))
	if err != nil {
		return nil, errclass.Internal{Err: err}
	}
	return stdjson.RawMessage(b), nil
}

func loadStringArgs(req Request) (map[string]string, error) {
	out := map[string]string{}
	if req.ArgsJSON != "" || req.ArgsFile != "" {
		obj, err := loadJSONObject(Request{
			Stdin:    req.Stdin,
			ArgsJSON: req.ArgsJSON,
			ArgsFile: req.ArgsFile,
		})
		if err != nil {
			return nil, err
		}
		for k, v := range obj {
			s, ok := v.(string)
			if !ok {
				return nil, errclass.Usage{Msg: fmt.Sprintf("argument %q must be a string", k)}
			}
			out[k] = s
		}
	}
	pairs, err := loadStringPairsOrError(req.ArgPairs)
	if err != nil {
		return nil, err
	}
	maps.Copy(out, pairs)
	return out, nil
}

func readArgsSource(req Request) ([]byte, error) {
	src := strings.TrimSpace(req.ArgsJSON)
	file := strings.TrimSpace(req.ArgsFile)
	if src != "" && file != "" {
		return nil, errclass.Usage{Msg: "--args cannot be combined with --args-file"}
	}
	switch {
	case file != "":
		return readFileOrStdin(file, req.Stdin)
	case src == "", src == "{}":
		if src == "{}" {
			return []byte("{}"), nil
		}
		return []byte("{}"), nil
	case src == "-":
		return readFileOrStdin("-", req.Stdin)
	case strings.HasPrefix(src, "@"):
		return readFileOrStdin(strings.TrimPrefix(src, "@"), req.Stdin)
	default:
		return []byte(src), nil
	}
}

func readFileOrStdin(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		b, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
		if err != nil {
			return nil, errclass.Usage{Err: err, Msg: "--args stdin"}
		}
		return b, nil
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: --args-file is an operator-supplied local path
	if err != nil {
		return nil, errclass.Usage{Err: err, Msg: "--args file"}
	}
	return b, nil
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func loadStringPairsOrError(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || k == "" {
			return nil, errclass.Usage{Msg: "--arg must be KEY=VALUE"}
		}
		out[k] = v
	}
	return out, nil
}
