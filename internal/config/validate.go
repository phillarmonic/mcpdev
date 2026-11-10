package config

import (
	"net"
	"net/url"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/redact"
)

func validateFile(path string, f File) error {
	if f.VersionSet && f.Version != CurrentVersion {
		return configErr(path, "version", "unsupported config version")
	}
	for name, p := range f.Profiles {
		if f.Insecure {
			p.Insecure = true
		}
		if err := validateProfile(path, name, p, f.Sensitive, false); err != nil {
			return err
		}
	}
	return nil
}

func validateResolved(path string, s Settings) error {
	for name, p := range s.Profiles {
		if s.Insecure {
			p.Insecure = true
		}
		if err := validateProfile(path, name, p, s.Sensitive, true); err != nil {
			return err
		}
	}
	return nil
}

func validateProfile(path, name string, p Profile, extra []string, resolved bool) error {
	sources := 0
	if len(p.Command) > 0 {
		sources++
	}
	if p.URL != "" {
		sources++
	}
	if p.Connector != "" {
		sources++
	}
	if sources == 0 {
		return configErr(path, profileField(name, ""), "exactly one of command, url, or connector is required")
	}
	if sources > 1 {
		return configErr(path, profileField(name, ""), "command, url, and connector are mutually exclusive")
	}
	if p.Connector == "grpc" {
		return configErr(path, profileField(name, "connector"), "grpc is not a standard MCP transport")
	}
	for k, v := range p.Headers {
		field := profileField(name, "headers."+k)
		if _, ok := p.HeaderEnv[k]; ok {
			return configErr(path, field, "conflicts with header_env; use one reference")
		}
		if !resolved && redact.Sensitive(k, extra) && !looksLikeInterpolation(v) {
			return configErr(path, field, "inline secret refused; use header_env or ${NAME:?}")
		}
	}
	for k, v := range p.HeaderEnv {
		if v == "" {
			return configErr(path, profileField(name, "header_env."+k), "environment variable name is required")
		}
		if looksLikeInterpolation(v) {
			return configErr(path, profileField(name, "header_env."+k), "must be an environment variable name, not an interpolated value")
		}
	}
	if err := validateTLS(path, name, p.TLS); err != nil {
		return err
	}
	if p.URL != "" {
		skipURL := !resolved && looksLikeInterpolation(p.URL)
		if !skipURL {
			if err := validateURL(path, name, p.URL, p.Insecure || p.TLS.Insecure); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTLS(path, name string, t TLS) error {
	if (t.CertFile == "") != (t.KeyFile == "") {
		return configErr(path, profileField(name, "tls"), "cert_file and key_file must be set together")
	}
	for field, v := range map[string]string{
		"tls.ca_file":   t.CAFile,
		"tls.cert_file": t.CertFile,
		"tls.key_file":  t.KeyFile,
	} {
		if looksLikePEM(v) {
			return configErr(path, profileField(name, field), "inline PEM refused; use a file path")
		}
	}
	return nil
}

func validateURL(path, name, raw string, insecure bool) error {
	if looksLikeInterpolation(raw) {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return configErr(path, profileField(name, "url"), "invalid URL")
	}
	switch u.Scheme {
	case "https", "http":
	default:
		return configErr(path, profileField(name, "url"), "unsupported URL scheme "+u.Scheme)
	}
	if u.User != nil {
		return configErr(path, profileField(name, "url"), "userinfo is refused; use header_env for credentials")
	}
	if u.Scheme == "http" && !isLoopback(u.Hostname()) && !insecure {
		return configErr(path, profileField(name, "url"), "non-loopback http requires insecure = true")
	}
	return nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
