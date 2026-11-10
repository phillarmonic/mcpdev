package config

import (
	"maps"

	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/redact"
)

// ResolveProfile expands a named profile into a live Target (no flag overlay).
func (s Settings) ResolveProfile(name string, env map[string]string) (port.Target, error) {
	p, ok := s.Profiles[name]
	if !ok {
		return port.Target{}, configErr("", profileField(name, ""), "unknown profile")
	}
	headers := maps.Clone(p.Headers)
	if headers == nil {
		headers = map[string]string{}
	}
	for h, varName := range p.HeaderEnv {
		val, ok := env[varName]
		if !ok || val == "" {
			return port.Target{}, configErr("", profileField(name, "header_env."+h), "missing required environment variable "+varName)
		}
		headers[h] = val
	}
	childEnv := maps.Clone(p.Env)
	if childEnv == nil {
		childEnv = map[string]string{}
	}
	insecure := s.Insecure
	if p.InsecureSet {
		insecure = p.Insecure
	}
	if p.TLS.InsecureSet && p.TLS.Insecure {
		insecure = true
	}
	kind := "stdio"
	switch {
	case p.URL != "":
		kind = "http"
	case p.Connector != "":
		kind = p.Connector
	}
	sensitive := ExtraKeys(s.Sensitive, headers, childEnv)
	return port.Target{
		Headers:        headers,
		Env:            childEnv,
		Kind:           kind,
		Profile:        name,
		URL:            p.URL,
		Connector:      p.Connector,
		Origin:         p.Origin,
		Protocol:       firstNonEmpty(p.Protocol, s.Protocol),
		Dir:            p.Dir,
		Command:        p.Command,
		Timeout:        s.Timeout,
		ConnectTimeout: s.ConnectTimeout,
		CAFile:         p.TLS.CAFile,
		CertFile:       p.TLS.CertFile,
		KeyFile:        p.TLS.KeyFile,
		Sensitive:      sensitive,
		AllowInsecure:  insecure,
	}, nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// DiagnosticTarget is the profile as it may be printed or persisted.
func DiagnosticTarget(t port.Target) port.Target {
	o := redact.Options{ExtraKeys: t.Sensitive}
	out := t
	out.URL = redact.URLString(t.URL, o)
	out.Command = redact.Argv(t.Command, o)
	out.Headers = redact.MapString(t.Headers, o)
	out.Env = redact.MapString(t.Env, o)
	return out
}
