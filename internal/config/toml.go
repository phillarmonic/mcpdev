package config

import (
	"errors"
	"fmt"
	"strings"
)

func parseTOML(data string) (File, error) {
	out := File{Profiles: map[string]Profile{}}
	var profile, section string
	for i, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			name, sec, err := parseTable(line)
			if err != nil {
				return File{}, fmt.Errorf("line %d: %w", i+1, err)
			}
			profile, section = name, sec
			continue
		}
		key, val, err := parseAssign(line)
		if err != nil {
			return File{}, fmt.Errorf("line %d: %w", i+1, err)
		}
		if err := applyKey(&out, profile, section, key, val); err != nil {
			return File{}, fmt.Errorf("line %d: %w", i+1, err)
		}
	}
	return out, nil
}

func parseTable(line string) (string, string, error) {
	if !strings.HasSuffix(line, "]") {
		return "", "", errors.New("invalid table")
	}
	body := strings.TrimSpace(line[1 : len(line)-1])
	const prefix = "profiles."
	if !strings.HasPrefix(body, prefix) {
		return "", "", errors.New("only [profiles.name] tables are supported")
	}
	rest := strings.TrimSpace(body[len(prefix):])
	if rest == "" {
		return "", "", errors.New("empty profile name")
	}
	sec := ""
	name := rest
	for _, s := range []string{"headers", "header_env", "env", "tls"} {
		suffix := "." + s
		if strings.HasSuffix(rest, suffix) {
			name = strings.TrimSpace(rest[:len(rest)-len(suffix)])
			sec = s
			break
		}
	}
	if name == "" || strings.Contains(name, ".") {
		return "", "", errors.New("invalid profile name")
	}
	return name, sec, nil
}

func applyKey(out *File, profile, section, key string, val value) error {
	if profile == "" {
		return applyTop(out, key, val)
	}
	p := out.Profiles[profile]
	err := applyProfile(&p, section, key, val)
	out.Profiles[profile] = p
	return err
}

func applyTop(out *File, key string, val value) error {
	switch key {
	case "version":
		n, err := requireInt(val, key)
		if err != nil {
			return err
		}
		out.Version, out.VersionSet = n, true
	case "protocol":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		out.Protocol = s
	case "timeout":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		out.Timeout = s
	case "connect_timeout":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		out.ConnectTimeout = s
	case "format":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		out.Format = s
	case "insecure":
		b, err := requireBool(val, key)
		if err != nil {
			return err
		}
		out.Insecure, out.InsecureSet = b, true
	case "allow_http_sse":
		b, err := requireBool(val, key)
		if err != nil {
			return err
		}
		out.AllowHTTPSSE, out.AllowHTTPSSESet = b, true
	case "sensitive":
		arr, err := requireArray(val, key)
		if err != nil {
			return err
		}
		out.Sensitive = arr
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}

func applyProfile(p *Profile, section, key string, val value) error {
	switch section {
	case "":
		return applyProfileRoot(p, key, val)
	case "headers":
		return putMap(&p.Headers, key, val, "headers")
	case "header_env":
		return putMap(&p.HeaderEnv, key, val, "header_env")
	case "env":
		return putMap(&p.Env, key, val, "env")
	case "tls":
		return applyTLS(&p.TLS, key, val)
	default:
		return fmt.Errorf("unknown profile section %q", section)
	}
}

func applyProfileRoot(p *Profile, key string, val value) error {
	switch key {
	case "url":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		p.URL = s
	case "command":
		if val.kind == kindString {
			return errors.New("command must be an argv array")
		}
		arr, err := requireArray(val, key)
		if err != nil {
			return err
		}
		p.Command = arr
	case "connector":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		p.Connector = s
	case "cwd":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		p.Dir = s
	case "origin":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		p.Origin = s
	case "protocol":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		p.Protocol = s
	case "insecure":
		b, err := requireBool(val, key)
		if err != nil {
			return err
		}
		p.Insecure, p.InsecureSet = b, true
	default:
		return fmt.Errorf("unknown profile key %q", key)
	}
	return nil
}

func applyTLS(t *TLS, key string, val value) error {
	switch key {
	case "ca_file":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		t.CAFile = s
	case "cert_file":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		t.CertFile = s
	case "key_file":
		s, err := requireString(val, key)
		if err != nil {
			return err
		}
		t.KeyFile = s
	case "insecure":
		b, err := requireBool(val, key)
		if err != nil {
			return err
		}
		t.Insecure, t.InsecureSet = b, true
	default:
		return fmt.Errorf("unknown tls key %q", key)
	}
	return nil
}

func putMap(dst *map[string]string, key string, val value, section string) error {
	s, err := requireString(val, key)
	if err != nil {
		return err
	}
	if *dst == nil {
		*dst = map[string]string{}
	}
	(*dst)[key] = s
	_ = section
	return nil
}
