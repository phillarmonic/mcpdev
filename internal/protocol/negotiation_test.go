package protocol_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/protocol"
	"github.com/phillarmonic/mcpdev/internal/protocol/modern"
)

func TestHTTP400RecognizedModernDoesNotFallback(t *testing.T) {
	header := protocol.HTTP400(modern.Era, http.StatusBadRequest, []byte(`{"jsonrpc":"2.0","error":{"code":-32020,"message":"HeaderMismatch"}}`))
	if header.Fallback {
		t.Fatalf("header mismatch: %+v", header)
	}
	miss := protocol.HTTP400(modern.Era, http.StatusBadRequest, []byte(`{"jsonrpc":"2.0","error":{"code":-32021,"message":"MissingRequiredClientCapabilityError"}}`))
	if miss.Fallback {
		t.Fatalf("missing capability: %+v", miss)
	}
	unsup := protocol.HTTP400(modern.Era, http.StatusBadRequest, []byte(`{"jsonrpc":"2.0","error":{"code":-32022,"message":"UnsupportedProtocolVersionError","data":{"supported":["2026-07-28"]}}}`))
	if unsup.Fallback || unsup.Negotiated != modern.Era {
		t.Fatalf("unsupported: %+v", unsup)
	}
}

func TestHTTP400EmptyAndUnrecognizedFallback(t *testing.T) {
	empty := protocol.HTTP400(modern.Era, http.StatusBadRequest, nil)
	if !empty.Fallback || empty.Reason != "empty_http_body" {
		t.Fatalf("empty: %+v", empty)
	}
	unrec := protocol.HTTP400(modern.Era, http.StatusBadRequest, []byte(`{"jsonrpc":"2.0","error":{"code":-1,"message":"nope"}}`))
	if !unrec.Fallback || unrec.Reason != "unrecognized_error" {
		t.Fatalf("unrecognized: %+v", unrec)
	}
}

func TestConnectModernRecordsNegotiation(t *testing.T) {
	lab := startNamed(t, "modern-stdio")
	tap := newMethodTap(lab.ClientIn)
	sess, err := protocol.Connect(t.Context(), protocol.Stream{Reader: lab.ClientOut, Writer: tap}, protocol.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sess.Close(t.Context()) })
	if slices.Contains(tap.methods(), "initialize") {
		t.Fatalf("modern must not initialize: %v", tap.methods())
	}
	info := sess.Info()
	if info.RequestedVersion != modern.Era || !slices.Contains(info.DiscoveredVersions, modern.Era) {
		t.Fatalf("negotiation: %+v", info)
	}
	if info.Fallback || info.ProtocolVersion != modern.Era {
		t.Fatalf("negotiated: %+v", info)
	}
	if info.Handshake != model.HandshakeDiscover {
		t.Fatalf("handshake: %q", info.Handshake)
	}
}

func TestConnectVersionRetryStaysModern(t *testing.T) {
	lab := startNamed(t, "modern-version-retry")
	sess, err := protocol.Connect(t.Context(), protocol.Stream{Reader: lab.ClientOut, Writer: lab.ClientIn}, protocol.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sess.Close(t.Context()) })
	info := sess.Info()
	if info.Fallback || info.ProtocolVersion != modern.Era || info.DiscoverState != model.DiscoverOK {
		t.Fatalf("retry: %+v", info)
	}
}

func TestConnectMissingCapabilityDoesNotFallback(t *testing.T) {
	lab := startNamed(t, "modern-need-cap")
	tap := newMethodTap(lab.ClientIn)
	_, err := protocol.Connect(t.Context(), protocol.Stream{Reader: lab.ClientOut, Writer: tap}, protocol.Options{})
	if err == nil {
		t.Fatal("expected missing capability error")
	}
	if errclass.ExitCode(err) != errclass.ExitProtocol {
		t.Fatalf("class: %d err=%v", errclass.ExitCode(err), err)
	}
	if slices.Contains(tap.methods(), "initialize") {
		t.Fatalf("must not fall back: %v", tap.methods())
	}
}
