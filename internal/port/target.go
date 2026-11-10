package port

import (
	"context"
	"time"
)

// Target is a live MCP destination after profile expansion.
// Kind is stdio or http. Command and URL stay mutually exclusive.
type Target struct {
	Headers        map[string]string
	Env            map[string]string
	Kind           string
	Profile        string
	URL            string
	Connector      string
	Origin         string
	Protocol       string
	Dir            string
	CAFile         string
	CertFile       string
	KeyFile        string
	Command        []string
	Sensitive      []string
	Timeout        time.Duration
	ConnectTimeout time.Duration
	AllowInsecure  bool
	SkipDiscover   bool
}

// Connector opens a Session for Target. Implementations live outside app.
type Connector interface {
	Connect(ctx context.Context, target Target) (Session, error)
}
