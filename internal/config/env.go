package config

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func applyEnv(base Settings, env map[string]string) (Settings, error) {
	if v := env["MCPDEV_PROTOCOL"]; v != "" {
		base.Protocol = v
	}
	if v := env["MCPDEV_FORMAT"]; v != "" {
		f, err := parseFormat(v)
		if err != nil {
			return Settings{}, err
		}
		base.Format = f
	}
	if v := env["MCPDEV_TIMEOUT"]; v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Settings{}, errclass.Usage{Err: err, Msg: "MCPDEV_TIMEOUT"}
		}
		base.Timeout = d
	}
	if v := env["MCPDEV_CONNECT_TIMEOUT"]; v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Settings{}, errclass.Usage{Err: err, Msg: "MCPDEV_CONNECT_TIMEOUT"}
		}
		base.ConnectTimeout = d
	}
	if v, ok := env["MCPDEV_INSECURE"]; ok && v != "" {
		b, err := ParseBoolEnv(v)
		if err != nil {
			return Settings{}, err
		}
		base.Insecure, base.InsecureSet = b, true
		if b {
			base.Warnings = appendWarning(base.Warnings, "MCPDEV_INSECURE enables insecure TLS or skipped Origin checks")
		}
	}
	if v, ok := env["MCPDEV_ALLOW_HTTP_SSE"]; ok && v != "" {
		b, err := ParseBoolEnv(v)
		if err != nil {
			return Settings{}, err
		}
		base.AllowHTTPSSE, base.AllowHTTPSSESet = b, true
	}
	return base, nil
}

func appendWarning(dst []string, msg string) []string {
	if slices.Contains(dst, msg) {
		return dst
	}
	return append(dst, msg)
}

// ParseCommandEnv decodes MCPDEV_COMMAND as a JSON argv array or a single slot.
func ParseCommandEnv(v string) ([]string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if strings.HasPrefix(v, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(v), &arr); err != nil {
			return nil, errclass.Usage{Err: err, Msg: "MCPDEV_COMMAND"}
		}
		return arr, nil
	}
	return []string{v}, nil
}

// ParseBoolEnv accepts 1/true/TRUE/yes and 0/false/FALSE/no/empty.
func ParseBoolEnv(v string) (bool, error) {
	switch v {
	case "", "0", "false", "FALSE", "no":
		return false, nil
	case "1", "true", "TRUE", "yes":
		return true, nil
	default:
		return false, errclass.Usage{Msg: "invalid boolean env value " + v}
	}
}
