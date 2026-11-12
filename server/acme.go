package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// acmeCerts obtains the certificate for one domain from an ACME CA and
// renews it, through autocert. Every handshake gets that certificate
// whatever name it asks for: the loopback client and the WebSocket proxy
// dial 127.0.0.1 with no server name, and a scanner hitting the address
// gets the real certificate rather than an error.
//
// The HTTP-01 challenge is answered on the HTTP port (see Handler), which
// a web proxy in front of the instance passes through like any other
// request. autocert tries TLS-ALPN-01 first and cannot be told not to; the
// proxy answers that on 443 with its own certificate, so each order
// records one failed validation at the CA before HTTP-01 succeeds.
type acmeCerts struct {
	m      *autocert.Manager
	domain string
}

// acmeCacheDirName is the directory under the data directory that holds
// the account key, the certificate, and the key.
const acmeCacheDirName = "certs"

// newACMECerts sets up the manager; nothing is requested until the first
// handshake or warm.
func newACMECerts(domain, email, caURL, cacheDir string) *acmeCerts {
	return &acmeCerts{
		m: &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(cacheDir),
			HostPolicy: autocert.HostWhitelist(domain),
			Email:      email,
			Client:     &acme.Client{DirectoryURL: caURL},
		},
		domain: domain,
	}
}

// hello is the handshake every caller is served as. The cipher suite
// marks the client as supporting ECDSA, so autocert keeps one ECDSA
// certificate rather than a second RSA one for a client that lists no
// ECDSA suites, and Leaf agrees with what the listeners serve.
func (a *acmeCerts) hello() *tls.ClientHelloInfo {
	return &tls.ClientHelloInfo{
		ServerName:   a.domain,
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}
}

// GetCertificate implements certSource.
func (a *acmeCerts) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return a.m.GetCertificate(a.hello())
}

// Leaf implements certSource.
func (a *acmeCerts) Leaf() ([]byte, error) {
	c, err := a.GetCertificate(nil)
	if err != nil {
		return nil, err
	}
	return c.Certificate[0], nil
}

// Handler answers the HTTP-01 challenge under /.well-known/acme-challenge/
// and nothing else. The host check inside autocert compares the request's
// Host with the domain, so a port a proxy may leave on it is removed.
func (a *acmeCerts) Handler() http.Handler {
	h := a.m.HTTPHandler(http.NotFoundHandler())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if host, _, err := net.SplitHostPort(r.Host); err == nil {
			r = r.Clone(r.Context())
			r.Host = host
		}
		h.ServeHTTP(w, r)
	})
}

// warm asks for the certificate in the background so the first order, or
// a renewal that is due, starts now rather than on the first connection,
// and its outcome is in the log. A handshake that arrives while an order
// is in progress waits for it, which can exceed the listeners' handshake
// timeout; the client retries and the next attempt is served from the
// cache.
func (a *acmeCerts) warm() {
	go func() {
		start := time.Now()
		c, err := a.GetCertificate(nil)
		if err != nil {
			log.Printf("TLS certificate for %v not obtained: %v", a.domain, err)
			return
		}
		log.Printf("TLS certificate for %v ready in %v, expires %v",
			a.domain, time.Since(start).Round(time.Millisecond),
			c.Leaf.NotAfter.Format(time.RFC3339))
	}()
}

// describe is the start-up log line.
func (a *acmeCerts) describe(cacheDir string) string {
	return fmt.Sprintf("TLS certificate for %v from %v, kept in %v; "+
		"the CA records one failed TLS-ALPN-01 attempt per order before HTTP-01 "+
		"succeeds through the proxy", a.domain, a.m.Client.DirectoryURL, cacheDir)
}
