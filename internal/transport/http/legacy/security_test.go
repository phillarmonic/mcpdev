package legacy

import (
	"strings"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/testkit"
)

func TestAuthFailureDoesNotLogSecret(t *testing.T) {
	const secret = "super-secret-token"
	d, err := testkit.Named("legacy-http")
	if err != nil {
		t.Fatal(err)
	}
	d.AuthRequired = true
	d.AuthToken = secret
	lab := testkit.StartHTTP(t, d)
	conn := dialLab(t, lab, Config{Headers: map[string]string{"Authorization": "Bearer wrong"}})
	_, err = conn.Post(t.Context(), rpc(1, "initialize", initParamsMap()))
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("class %d err %v", errclass.ExitCode(err), err)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "Bearer wrong") {
		t.Fatalf("secret leaked: %v", err)
	}
}

func TestTLSFailureAndInsecureOptIn(t *testing.T) {
	d, err := testkit.Named("legacy-http")
	if err != nil {
		t.Fatal(err)
	}
	untrusted := testkit.StartHTTPS(t, d)
	reject, err := Dial(Config{Endpoint: untrusted.Server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reject.Post(t.Context(), rpc(1, "initialize", initParamsMap()))
	if errclass.ExitCode(err) != errclass.ExitConnection {
		t.Fatalf("untrusted TLS class %d err %v", errclass.ExitCode(err), err)
	}

	trustedLab := testkit.StartHTTPS(t, d)
	trusted := dialLab(t, trustedLab, Config{HTTPClient: trustedLab.Server.Client()})
	handshake(t, trusted)

	insecureLab := testkit.StartHTTPS(t, d)
	insecure := dialLab(t, insecureLab, Config{AllowInsecure: true})
	handshake(t, insecure)
}

func TestNonLoopbackHTTPRequiresInsecure(t *testing.T) {
	_, err := Dial(Config{Endpoint: "http://example.com/mcp"})
	if errclass.ExitCode(err) != errclass.ExitUsage {
		t.Fatalf("class %d err %v", errclass.ExitCode(err), err)
	}
	if _, err := Dial(Config{Endpoint: "http://example.com/mcp", AllowInsecure: true}); err != nil {
		t.Fatal(err)
	}
}
