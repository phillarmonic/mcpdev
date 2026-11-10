// Package tlsclient applies operator TLS file references to HTTP transports.
package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Apply clones default transport settings onto tr: optional CA pool, client cert, skip-verify.
func Apply(tr *http.Transport, insecure bool, caFile, certFile, keyFile string) error {
	cfg := tr.TLSClientConfig
	if cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		cfg = cfg.Clone()
	}
	if caFile != "" {
		pem, err := os.ReadFile(caFile) //nolint:gosec // G304: path comes from operator config
		if err != nil {
			return errclass.Usage{Err: err, Msg: "tls: ca_file"}
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return errclass.Usage{Msg: "tls: ca_file: no certificates"}
		}
		cfg.RootCAs = pool
	}
	if certFile != "" || keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return errclass.Usage{Err: err, Msg: "tls: client certificate"}
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	if insecure {
		cfg.InsecureSkipVerify = true // #nosec G402 -- AllowInsecure is an explicit operator opt-in
	}
	if caFile != "" || certFile != "" || keyFile != "" || insecure {
		tr.TLSClientConfig = cfg
	}
	return nil
}
