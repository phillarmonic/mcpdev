package process

import (
	"errors"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/port"
)

const defaultStderrLimit = 64 * 1024

func mergeEnv(spec port.ProcSpec) []string {
	values := make(map[string]string)
	if !spec.ClearEnv {
		for _, pair := range os.Environ() {
			key, val, ok := strings.Cut(pair, "=")
			if !ok {
				continue
			}
			values[key] = val
		}
	}
	maps.Copy(values, spec.Env)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func resolveArgv(spec port.ProcSpec, lookPath func(string) (string, error)) ([]string, error) {
	if len(spec.Argv) == 0 || strings.TrimSpace(spec.Argv[0]) == "" {
		return nil, errors.New("process: argv is required")
	}
	argv := slices.Clone(spec.Argv)
	path := argv[0]
	if lookPath != nil && !strings.ContainsRune(path, os.PathSeparator) {
		resolved, err := lookPath(path)
		if err != nil {
			return nil, err
		}
		argv[0] = resolved
	}
	return argv, nil
}
