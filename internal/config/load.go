package config

import (
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// Load walks project and user files then applies env. Flags are applied by the CLI.
func Load(cwd string, env map[string]string) (Settings, error) {
	out := Defaults()
	user, err := userConfigPath(env)
	if err != nil {
		return Settings{}, err
	}
	if user != "" {
		var exists bool
		var merged Settings
		merged, exists, err = mergePath(out, user, explicitUser(env))
		if err != nil {
			return Settings{}, err
		}
		out = merged
		if exists {
			out.UserPath = user
		}
	}
	project := findProject(cwd)
	if project != "" {
		var merged Settings
		merged, _, err = mergePath(out, project, true)
		if err != nil {
			return Settings{}, err
		}
		out = merged
		out.ProjectPath = project
	}
	out, err = applyEnv(out, env)
	if err != nil {
		return Settings{}, err
	}
	out, err = interpolateSettings(out, env)
	if err != nil {
		return Settings{}, err
	}
	label := out.ProjectPath
	if label == "" {
		label = out.UserPath
	}
	err = validateResolved(label, out)
	if err != nil {
		return Settings{}, err
	}
	return out, nil
}

func explicitUser(env map[string]string) bool {
	return env["MCPDEV_CONFIG"] != ""
}

func mergePath(base Settings, path string, required bool) (Settings, bool, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: config path is walked from cwd, XDG, APPDATA, or MCPDEV_CONFIG
	if err != nil {
		if os.IsNotExist(err) && !required {
			return base, false, nil
		}
		if os.IsNotExist(err) {
			return Settings{}, false, configErr(path, "", "file not found")
		}
		return Settings{}, false, errclass.Usage{Err: err, Msg: "config: read " + path}
	}
	file, err := parseTOML(string(data))
	if err != nil {
		return Settings{}, false, configErr(path, "", err.Error())
	}
	err = validateFile(path, file)
	if err != nil {
		return Settings{}, false, err
	}
	var merged Settings
	merged, err = mergeFile(base, path, file)
	if err != nil {
		return Settings{}, false, err
	}
	return merged, true, nil
}

func mergeFile(base Settings, path string, file File) (Settings, error) {
	if file.Protocol != "" {
		base.Protocol = file.Protocol
	}
	if file.Format != "" {
		f, err := parseFormat(file.Format)
		if err != nil {
			return Settings{}, configErr(path, "format", err.Error())
		}
		base.Format = f
	}
	if file.Timeout != "" {
		d, err := time.ParseDuration(file.Timeout)
		if err != nil {
			return Settings{}, configErr(path, "timeout", "invalid duration")
		}
		base.Timeout = d
	}
	if file.ConnectTimeout != "" {
		d, err := time.ParseDuration(file.ConnectTimeout)
		if err != nil {
			return Settings{}, configErr(path, "connect_timeout", "invalid duration")
		}
		base.ConnectTimeout = d
	}
	if file.InsecureSet {
		base.Insecure, base.InsecureSet = file.Insecure, true
		if file.Insecure {
			base.Warnings = appendWarning(base.Warnings, "config: "+path+": insecure = true enables insecure TLS or skipped Origin checks")
		}
	}
	if file.AllowHTTPSSESet {
		base.AllowHTTPSSE, base.AllowHTTPSSESet = file.AllowHTTPSSE, true
	}
	if len(file.Sensitive) > 0 {
		base.Sensitive = mergeUnique(base.Sensitive, file.Sensitive)
	}
	if base.Profiles == nil {
		base.Profiles = map[string]Profile{}
	}
	for name, p := range file.Profiles {
		base.Profiles[name] = cloneProfile(p)
	}
	return base, nil
}

func mergeUnique(a, b []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range append(slices.Clone(a), b...) {
		if _, ok := seen[s]; ok || s == "" {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func cloneProfile(p Profile) Profile {
	p.Headers = maps.Clone(p.Headers)
	p.HeaderEnv = maps.Clone(p.HeaderEnv)
	p.Env = maps.Clone(p.Env)
	p.Command = slices.Clone(p.Command)
	return p
}

func interpolateSettings(s Settings, env map[string]string) (Settings, error) {
	out := s
	out.Profiles = make(map[string]Profile, len(s.Profiles))
	for name, p := range s.Profiles {
		np, err := interpolateProfile(name, cloneProfile(p), env)
		if err != nil {
			return Settings{}, err
		}
		out.Profiles[name] = np
	}
	return out, nil
}

func interpolateProfile(name string, p Profile, env map[string]string) (Profile, error) {
	var err error
	field := func(f string) string { return profileField(name, f) }
	if p.URL, err = interpolate(p.URL, field("url"), env); err != nil {
		return Profile{}, err
	}
	if p.Connector, err = interpolate(p.Connector, field("connector"), env); err != nil {
		return Profile{}, err
	}
	if p.Origin, err = interpolate(p.Origin, field("origin"), env); err != nil {
		return Profile{}, err
	}
	if p.Dir, err = interpolate(p.Dir, field("cwd"), env); err != nil {
		return Profile{}, err
	}
	if p.Protocol, err = interpolate(p.Protocol, field("protocol"), env); err != nil {
		return Profile{}, err
	}
	if p.TLS.CAFile, err = interpolate(p.TLS.CAFile, field("tls.ca_file"), env); err != nil {
		return Profile{}, err
	}
	if p.TLS.CertFile, err = interpolate(p.TLS.CertFile, field("tls.cert_file"), env); err != nil {
		return Profile{}, err
	}
	if p.TLS.KeyFile, err = interpolate(p.TLS.KeyFile, field("tls.key_file"), env); err != nil {
		return Profile{}, err
	}
	for i, slot := range p.Command {
		p.Command[i], err = interpolate(slot, field("command"), env)
		if err != nil {
			return Profile{}, err
		}
	}
	if p.Headers, err = interpolateMap(p.Headers, field("headers"), env); err != nil {
		return Profile{}, err
	}
	if p.Env, err = interpolateMap(p.Env, field("env"), env); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func parseFormat(s string) (model.Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "human":
		return model.FormatHuman, nil
	case "json":
		return model.FormatJSON, nil
	default:
		return "", errclass.Usage{Msg: "config: unknown format " + s}
	}
}
