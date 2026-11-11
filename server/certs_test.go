package server

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
)

// leafOf returns the DER of the first certificate in a PEM.
func leafOf(t *testing.T, pem []byte) []byte {
	t.Helper()
	der, err := client.LeafCert(pem)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// serves checks which certificate the source hands a handshake.
func serves(t *testing.T, src certSource, want []byte) {
	t.Helper()
	c, err := src.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if string(c.Certificate[0]) != string(want) {
		t.Fatal("source served the wrong certificate")
	}
	leaf, err := src.Leaf()
	if err != nil {
		t.Fatal(err)
	}
	if string(leaf) != string(want) {
		t.Fatal("Leaf differs from the served certificate")
	}
}

// TestFileCertsReload: a rewritten pair is served after the check
// interval, including one whose modification time did not change, and a
// pair that does not load keeps the old one.
func TestFileCertsReload(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, first := selfSigned(t, dir, "server")

	f, err := newFileCerts(certFile, keyFile, "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	f.now = func() time.Time { return now }
	serves(t, f, leafOf(t, first))

	// rewritten, but the interval has not passed
	second := writeSelfSigned(t, certFile, keyFile, "server", time.Now())
	serves(t, f, leafOf(t, first))

	now = now.Add(f.interval)
	serves(t, f, leafOf(t, second))

	// rewritten with the old modification time; the name is longer so
	// the size differs
	info, err := os.Stat(certFile)
	if err != nil {
		t.Fatal(err)
	}
	third := writeSelfSigned(t, certFile, keyFile, "server-renewed", time.Now())
	for _, p := range []string{certFile, keyFile} {
		if err := os.Chtimes(p, info.ModTime(), info.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(f.interval)
	serves(t, f, leafOf(t, third))

	// a key that does not match (half a renewal) keeps the pair loaded
	_, otherKey, _ := selfSigned(t, dir, "other")
	key, err := os.ReadFile(otherKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, key, 0o600); err != nil {
		t.Fatal(err)
	}
	now = now.Add(f.interval)
	serves(t, f, leafOf(t, third))
}

// TestFindPair: the directory search finds the layouts Caddy and certbot
// write, prefers the certificate issued most recently, and follows a
// renewal written under another issuer.
func TestFindPair(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := findPair(dir); err == nil ||
		!strings.Contains(err.Error(), "no certificate and key pair") {
		t.Fatal("empty directory not refused:", err)
	}

	// Caddy's layout: one issuer so far
	le := filepath.Join(dir, "acme-v02.api.letsencrypt.org-directory", "siot.example.com")
	leCert := filepath.Join(le, "siot.example.com.crt")
	leKey := filepath.Join(le, "siot.example.com.key")
	lePEM := writeSelfSigned(t, leCert, leKey, "le", time.Now().Add(-48*time.Hour))
	if err := os.WriteFile(filepath.Join(le, "siot.example.com.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := newFileCerts("", "", dir)
	if err != nil {
		t.Fatal(err)
	}
	f.interval = 0
	serves(t, f, leafOf(t, lePEM))

	// a renewal under the other issuer, issued later, wins
	zs := filepath.Join(dir, "acme.zerossl.com-v2-dv90", "siot.example.com")
	zsPEM := writeSelfSigned(t, filepath.Join(zs, "siot.example.com.crt"),
		filepath.Join(zs, "siot.example.com.key"), "zerossl", time.Now().Add(-time.Hour))
	serves(t, f, leafOf(t, zsPEM))

	// a certificate with no key beside it is not a pair
	stray := filepath.Join(dir, "stray")
	writeSelfSigned(t, filepath.Join(stray, "a.crt"), filepath.Join(stray, "b.key"),
		"stray", time.Now())
	serves(t, f, leafOf(t, zsPEM))

	// certbot's layout
	cb := t.TempDir()
	cbPEM := writeSelfSigned(t, filepath.Join(cb, "live", "siot.example.com", "fullchain.pem"),
		filepath.Join(cb, "live", "siot.example.com", "privkey.pem"), "certbot", time.Now())
	g, err := newFileCerts("", "", cb)
	if err != nil {
		t.Fatal(err)
	}
	serves(t, g, leafOf(t, cbPEM))
}

// TestNewCertSource: the option combinations.
func TestNewCertSource(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := selfSigned(t, dir, "server")

	if src, err := newCertSource(Options{}); err != nil || src != nil {
		t.Fatal("no settings should give no source:", src, err)
	}
	for _, o := range []Options{
		{NatsTLSCert: certFile},
		{NatsTLSCert: certFile, NatsTLSKey: keyFile, NatsTLSDir: dir},
		{NatsTLSDir: filepath.Join(dir, "missing")},
	} {
		if _, err := newCertSource(o); err == nil {
			t.Fatalf("%+v not refused", o)
		}
	}
	for _, o := range []Options{
		{NatsTLSCert: certFile, NatsTLSKey: keyFile},
		{NatsTLSDir: dir},
	} {
		if _, err := newCertSource(o); err != nil {
			t.Fatalf("%+v: %v", o, err)
		}
	}
}

// TestClientTLSConfig: the loopback configuration accepts the certificate
// the source serves now, including after a renewal, and nothing else.
func TestClientTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, first := selfSigned(t, dir, "server")
	_, _, other := selfSigned(t, dir, "other")

	f, err := newFileCerts(certFile, keyFile, "")
	if err != nil {
		t.Fatal(err)
	}
	f.interval = 0
	cfg := clientTLSConfig(f)
	verify := func(pem []byte) error {
		return cfg.VerifyPeerCertificate([][]byte{leafOf(t, pem)}, nil)
	}

	if err := verify(first); err != nil {
		t.Fatal("served certificate refused:", err)
	}
	if err := verify(other); err == nil {
		t.Fatal("another certificate accepted")
	}

	second := writeSelfSigned(t, certFile, keyFile, "server", time.Now())
	if err := verify(second); err != nil {
		t.Fatal("renewed certificate refused:", err)
	}
	if err := verify(first); err == nil {
		t.Fatal("replaced certificate still accepted")
	}
	if err := cfg.VerifyPeerCertificate(nil, nil); err == nil {
		t.Fatal("no certificate accepted")
	}

	// the parsed leaf is kept with the pair
	c, err := f.GetCertificate(nil)
	if err != nil || c.Leaf == nil {
		t.Fatal("leaf not parsed:", err)
	}
	if _, err := x509.ParseCertificate(c.Certificate[0]); err != nil {
		t.Fatal(err)
	}
}
