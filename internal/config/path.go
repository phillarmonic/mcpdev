package config

import (
	"os"
	"path/filepath"
	"runtime"
)

func findProject(cwd string) string {
	dir := cwd
	for {
		for _, name := range []string{"mcpdev.toml", ".mcpdev.toml"} {
			p := filepath.Join(dir, name)
			st, err := os.Stat(p)
			if err == nil && !st.IsDir() {
				return p
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func userConfigPath(env map[string]string) (string, error) {
	return userConfigPathGOOS(env, runtime.GOOS)
}

func userConfigPathGOOS(env map[string]string, goosName string) (string, error) {
	if p := env["MCPDEV_CONFIG"]; p != "" {
		return p, nil
	}
	xdg := env["XDG_CONFIG_HOME"]
	if xdg != "" {
		return filepath.Join(xdg, "mcpdev", "config.toml"), nil
	}
	if goosName == "windows" {
		if app := env["APPDATA"]; app != "" {
			return filepath.Join(app, "mcpdev", "config.toml"), nil
		}
	}
	home := env["HOME"]
	if home == "" {
		return "", nil
	}
	return filepath.Join(home, ".config", "mcpdev", "config.toml"), nil
}
