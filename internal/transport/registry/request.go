package registry

import (
	"context"
	"io"
	"net/http"

	"github.com/phillarmonic/mcpdev/internal/port"
	"github.com/phillarmonic/mcpdev/internal/trace"
)

// Request is the configuration used to select and construct a connector.
// Selection is a pure function of these fields; it does not probe the network.
type Request struct {
	Supervisor    port.ProcessSupervisor
	Clock         port.Clock
	Headers       map[string]string
	Env           map[string]string
	HTTPClient    *http.Client
	Tracer        *trace.Recorder
	Connector     string
	Protocol      string
	URL           string
	Origin        string
	Dir           string
	CAFile        string
	CertFile      string
	KeyFile       string
	Command       []string
	AllowInsecure bool
}

// Selection is the deterministic outcome of matching Request to a registered name.
type Selection struct {
	Constructed string
	EraPolicy   string
	Descriptor  Descriptor
}

// Handle is a live connector. Close owns process or HTTP client teardown.
type Handle interface {
	Name() string
	Selection() Selection
	Diagnostics() map[string]string
	ByteStream(ctx context.Context) (io.ReadCloser, io.WriteCloser)
	Close(ctx context.Context) error
}

// Factory constructs a Handle for one registered name.
type Factory func(ctx context.Context, req Request, sel Selection) (Handle, error)
