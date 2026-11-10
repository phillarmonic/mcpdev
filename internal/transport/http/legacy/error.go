package legacy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

var (
	errDefaultTransport = errors.New("http: DefaultTransport is not an *http.Transport")
	errSessionNotFound  = errors.New("http: session not found")
)

func classifyStatus(status int, body []byte, contentType string) error {
	switch status {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return errclass.Connection{Err: errSessionNotFound}
	case http.StatusUnauthorized, http.StatusForbidden:
		return errclass.Connection{Err: errors.New("http: authentication failed")}
	case http.StatusBadRequest:
		return errclass.Protocol{Err: fmt.Errorf("http: bad request (%s)", previewBody(body, contentType))}
	case http.StatusMethodNotAllowed:
		return errclass.Protocol{Err: errors.New("http: method not allowed on this era")}
	default:
		if status >= 400 {
			return errclass.Connection{Err: fmt.Errorf("http: endpoint status %d", status)}
		}
		return nil
	}
}

func previewBody(body []byte, contentType string) string {
	if len(body) == 0 {
		return "empty body"
	}
	if !strings.Contains(contentType, "json") {
		return "non-json body"
	}
	s := string(body)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func wrapRoundTrip(err error) error {
	if err == nil {
		return nil
	}
	var coded interface{ ExitClass() int }
	if errors.As(err, &coded) {
		return err
	}
	return errclass.Connection{Err: fmt.Errorf("http: %w", err)}
}
