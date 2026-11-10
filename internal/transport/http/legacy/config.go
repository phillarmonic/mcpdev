package legacy

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/trace"
	"github.com/phillarmonic/mcpdev/internal/transport/tlsclient"
)

// ProtocolVersion is the 2025-11-25 Streamable HTTP era this adapter speaks.
const ProtocolVersion = "2025-11-25"

const defaultMaxReconnects = 5

// Config is a 2025-11-25 Streamable HTTP endpoint.
type Config struct {
	Headers        map[string]string
	Clock          port.Clock
	HTTPClient     *http.Client
	Tracer         *trace.Recorder
	Endpoint       string
	Origin         string
	CAFile         string
	CertFile       string
	KeyFile        string
	ReconnectDelay time.Duration
	MaxReconnects  int
	AllowInsecure  bool
}

func (c Config) clock() port.Clock {
	if c.Clock != nil {
		return c.Clock
	}
	return clock.Wall{}
}

func (c Config) maxReconnects() int {
	switch {
	case c.MaxReconnects < 0:
		return 0
	case c.MaxReconnects == 0:
		return defaultMaxReconnects
	default:
		return c.MaxReconnects
	}
}

func parseEndpoint(raw string, allowInsecure bool) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errclass.Usage{Msg: "http: endpoint URL is required"}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errclass.Usage{Err: err, Msg: "http: endpoint URL"}
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !isLoopback(u.Hostname()) && !allowInsecure {
			return nil, errclass.Usage{Msg: "http: non-loopback http URL requires AllowInsecure"}
		}
	default:
		return nil, errclass.Usage{Msg: "http: unsupported URL scheme " + u.Scheme}
	}
	if u.Host == "" {
		return nil, errclass.Usage{Msg: "http: endpoint URL is missing a host"}
	}
	return u, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func originFor(cfg Config, u *url.URL) string {
	if cfg.Origin != "" {
		return cfg.Origin
	}
	return u.Scheme + "://" + u.Host
}

func buildClient(cfg Config, u *url.URL) (*http.Client, error) {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient, nil
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errclass.Internal{Err: errDefaultTransport}
	}
	tr := base.Clone()
	skip := u.Scheme == "https" && cfg.AllowInsecure
	if err := tlsclient.Apply(tr, skip, cfg.CAFile, cfg.CertFile, cfg.KeyFile); err != nil {
		return nil, err
	}
	return &http.Client{Transport: tr}, nil
}
