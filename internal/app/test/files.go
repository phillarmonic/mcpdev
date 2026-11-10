package test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func collectFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, errclass.Usage{Msg: "test requires a scenario file or directory"}
	}
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, errclass.Usage{Msg: "test path", Err: err}
		}
		if info.IsDir() {
			err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				switch strings.ToLower(filepath.Ext(path)) {
				case ".yaml", ".yml", ".json":
					out = append(out, path)
				}
				return nil
			})
			if err != nil {
				return nil, errclass.Usage{Msg: "test directory", Err: err}
			}
			continue
		}
		out = append(out, p)
	}
	slices.Sort(out)
	out = slices.Compact(out)
	if len(out) == 0 {
		return nil, errclass.Usage{Msg: "no scenario YAML or JSON files found"}
	}
	return out, nil
}
