package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
)

const testDomain = "siot.example.com"

// seededACME returns an ACME source whose cache already holds a
// certificate for the domain, so no CA is contacted. The certificate is
// long lived so autocert does not schedule a renewal during the test.
func seededACME(t *testing.T) (*acmeCerts, []byte) {
	t.Helper()
	dir := t.TempDir()
	certPEM, keyPEM := newSelfSigned(t, testDomain, []string{testDomain},
		time.Now().Add(-time.Hour), time.Now().Add(365*24*time.Hour))
	// autocert's cache file: the key, then the chain
	if err := os.WriteFile(filepath.Join(dir, testDomain),
		append(append([]byte{}, keyPEM...), certPEM...), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newACMECerts(testDomain, "", "https://127.0.0.1:1/unreachable", dir)
	return a, leafOf(t, certPEM)
}

// TestACMECertsServesDomain: every handshake gets the domain's certificate,
// with no server name, with the domain, and with another name, and Leaf
// agrees with it.
func TestACMECertsServesDomain(t *testing.T) {
	a, leaf := seededACME(t)

	for _, hello := range []*tls.ClientHelloInfo{
		nil,
		{},
		{ServerName: testDomain},
		{ServerName: "other.example.net"},
		// a client listing no ECDSA suite still gets the one certificate
		{ServerName: testDomain, CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_GCM_SHA256}},
	} {
		c, err := a.GetCertificate(hello)
		if err != nil {
			t.Fatalf("%+v: %v", hello, err)
		}
		if string(c.Certificate[0]) != string(leaf) {
			t.Fatalf("%+v: wrong certificate", hello)
		}
	}
	got, err := a.Leaf()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(leaf) {
		t.Fatal("Leaf differs from the served certificate")
	}
	if _, err := x509.ParseCertificate(got); err != nil {
		t.Fatal(err)
	}
}

// TestACMEChallengeRoute: the HTTP port answers only the challenge path
// for the domain, with a port a proxy may leave on the host removed, and
// nothing without ACME.
func TestACMEChallengeRoute(t *testing.T) {
	a, _ := seededACME(t)
	app := api.NewAppHandler(api.ServerArgs{ACMEHandler: a.Handler()})

	get := func(path, host string) int {
		req := httptest.NewRequest("GET", path, nil)
		req.Host = host
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code
	}
	// no token has been placed, so the right host and path is 404; the
	// wrong host is refused before the token lookup
	if got := get("/.well-known/acme-challenge/token", testDomain); got != http.StatusNotFound {
		t.Fatal("challenge path for the domain:", got)
	}
	if got := get("/.well-known/acme-challenge/token", testDomain+":443"); got != http.StatusNotFound {
		t.Fatal("challenge path with a port on the host:", got)
	}
	if got := get("/.well-known/acme-challenge/token", "other.example.net"); got != http.StatusForbidden {
		t.Fatal("challenge path for another host:", got)
	}
	if got := get("/.well-known/other", testDomain); got != http.StatusNotFound {
		t.Fatal("another well-known path:", got)
	}

	plain := api.NewAppHandler(api.ServerArgs{})
	req := httptest.NewRequest("GET", "/.well-known/acme-challenge/token", nil)
	rec := httptest.NewRecorder()
	plain.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatal("challenge path without ACME:", rec.Code)
	}
}

// TestACMESettings: a domain is a source on its own and not with the
// file settings.
func TestACMESettings(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := selfSigned(t, dir, "server")

	src, err := newCertSource(Options{DataDir: dir, NatsTLSDomain: testDomain})
	if err != nil {
		t.Fatal(err)
	}
	a, ok := src.(*acmeCerts)
	if !ok {
		t.Fatalf("domain gave %T", src)
	}
	if a.m.Client.DirectoryURL == "" {
		t.Fatal("no default CA")
	}
	src, err = newCertSource(Options{DataDir: dir, NatsTLSDomain: testDomain,
		NatsTLSCA: "https://acme-staging-v02.api.letsencrypt.org/directory"})
	if err != nil {
		t.Fatal(err)
	}
	if src.(*acmeCerts).m.Client.DirectoryURL != "https://acme-staging-v02.api.letsencrypt.org/directory" {
		t.Fatal("CA setting not used")
	}

	for _, o := range []Options{
		{NatsTLSDomain: testDomain, NatsTLSCert: certFile, NatsTLSKey: keyFile},
		{NatsTLSDomain: testDomain, NatsTLSDir: dir},
	} {
		if _, err := newCertSource(o); err == nil {
			t.Fatalf("%+v not refused", o)
		}
	}
}

// TestACMEOrder runs a full order against a local ACME test server, such
// as pebble (https://github.com/letsencrypt/pebble), when
// SIOT_TEST_ACME_DIR names its directory URL. The CA has to validate
// HTTP-01 on the test server's HTTP port, so run pebble with httpPort set
// to that port and a domain that resolves to this machine, and trust
// pebble's own certificate through SSL_CERT_FILE:
//
//	PEBBLE_VA_NOSLEEP=1 pebble -config pebble-config.json   # httpPort: 8901
//	SSL_CERT_FILE=test/certs/pebble.minica.pem \
//	SIOT_TEST_ACME_DIR=https://localhost:14000/dir \
//	SIOT_TEST_ACME_DOMAIN=siot.localhost go test ./server -run TestACMEOrder
func TestACMEOrder(t *testing.T) {
	dirURL := os.Getenv("SIOT_TEST_ACME_DIR")
	if dirURL == "" {
		t.Skip("SIOT_TEST_ACME_DIR not set")
	}
	domain := os.Getenv("SIOT_TEST_ACME_DOMAIN")
	if domain == "" {
		domain = "siot.localhost"
	}

	opts := TestServerOptions
	opts.AuthToken = "tok"
	opts.NatsTLSDomain = domain
	opts.NatsTLSCA = dirURL
	opts.NatsTLSTimeout = 30
	opts.DataDir = t.TempDir()
	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	deadline := time.Now().Add(2 * time.Minute)
	for {
		c, err := nats.Connect(fmt.Sprintf("tls://localhost:%v", opts.NatsPort),
			nats.Token("tok"), nats.NoReconnect(),
			nats.Secure(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})) //nolint:gosec
		if err == nil {
			state, err := c.TLSConnectionState()
			c.Close()
			if err != nil {
				t.Fatal(err)
			}
			leaf := state.PeerCertificates[0]
			if leaf.Issuer.String() == leaf.Subject.String() {
				t.Fatal("certificate is self-signed")
			}
			if err := leaf.VerifyHostname(domain); err != nil {
				t.Fatal(err)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no certificate obtained:", err)
		}
		time.Sleep(2 * time.Second)
	}
}
