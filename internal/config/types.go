package config

import (
	"time"

	"github.com/phillarmonic/mcpdev/internal/model"
)

// CurrentVersion is the only supported mcpdev.toml version.
const CurrentVersion = 1

// Settings is merged configuration before flag overlay.
type Settings struct {
	Profiles        map[string]Profile
	UserPath        string
	Protocol        string
	Format          model.Format
	ProjectPath     string
	Warnings        []string
	Sensitive       []string
	Timeout         time.Duration
	ConnectTimeout  time.Duration
	Insecure        bool
	InsecureSet     bool
	AllowHTTPSSE    bool
	AllowHTTPSSESet bool
}

// File is one decoded config document.
type File struct {
	Profiles        map[string]Profile
	Protocol        string
	Timeout         string
	ConnectTimeout  string
	Format          string
	Sensitive       []string
	Version         int
	VersionSet      bool
	Insecure        bool
	InsecureSet     bool
	AllowHTTPSSE    bool
	AllowHTTPSSESet bool
}

// Profile is one named target. Command is an argv array, never a shell string.
type Profile struct {
	Headers     map[string]string
	HeaderEnv   map[string]string
	Env         map[string]string
	URL         string
	Connector   string
	Origin      string
	Dir         string
	Protocol    string
	Command     []string
	TLS         TLS
	Insecure    bool
	InsecureSet bool
}

// TLS holds certificate file references. PEM blobs are rejected.
type TLS struct {
	CAFile      string
	CertFile    string
	KeyFile     string
	Insecure    bool
	InsecureSet bool
}

// Defaults are built-in values (protocol auto, human format, insecure off).
func Defaults() Settings {
	return Settings{
		Protocol:       "auto",
		Format:         model.FormatHuman,
		Timeout:        30 * time.Second,
		ConnectTimeout: 15 * time.Second,
		Profiles:       map[string]Profile{},
	}
}
