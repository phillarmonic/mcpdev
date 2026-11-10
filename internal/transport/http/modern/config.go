package modern

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/protocol/modern"
	"github.com/phillarmonic/mcpdev/internal/trace"
	"github.com/phillarmonic/mcpdev/internal/transport/tlsclient"
)

// ProtocolVersion is the 2026-07-28 Streamable HTTP era this adapter speaks.
const ProtocolVersion = modern.Era

// Config is a 2026-07-28 Streamable HTTP endpoint.
// There is no session, reconnect, or GET listen configuration.
type Config struct {
	Headers       map[string]string
	HTTPClient    *http.Client
	Tracer        *trace.Recorder
	Endpoint      string
	Origin        string
	CAFile        string
	CertFile      string
	KeyFile       string
	AllowInsecure bool
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
