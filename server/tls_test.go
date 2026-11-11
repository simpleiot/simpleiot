package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
)

// selfSigned writes a self-signed certificate for localhost and returns
// the file paths and the certificate PEM.
func selfSigned(t *testing.T, dir, name string) (certFile, keyFile string, certPEM []byte) {
	t.Helper()
	certFile = filepath.Join(dir, name+".crt")
	keyFile = filepath.Join(dir, name+".key")
	certPEM = writeSelfSigned(t, certFile, keyFile, name, time.Now().Add(-time.Hour))
	return certFile, keyFile, certPEM
}

// writeSelfSigned writes a self-signed certificate for localhost, issued
// at notBefore, to the two paths and returns the certificate PEM.
func writeSelfSigned(t *testing.T, certFile, keyFile, name string, notBefore time.Time) []byte {
	t.Helper()
	certPEM, keyPEM := newSelfSigned(t, name, []string{"localhost"}, notBefore, notBefore.Add(2*time.Hour))
	if err := os.MkdirAll(filepath.Dir(certFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return certPEM
}

// newSelfSigned returns a self-signed certificate for the names and its
// key, both as PEM.
func newSelfSigned(t *testing.T, name string, dns []string, notBefore, notAfter time.Time) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              dns,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

// TestTLS: with a certificate configured, the NATS and WebSocket
// listeners serve TLS, the HTTP port's proxy still reaches the WebSocket
// listener, and an edge client that pins a certificate refuses a server
// presenting another.
func TestTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, certPEM := selfSigned(t, dir, "server")
	_, _, otherPEM := selfSigned(t, dir, "other")

	opts := TestServerOptions
	opts.AuthToken = "tok"
	opts.NatsTLSCert = certFile
	opts.NatsTLSKey = keyFile
	opts.NatsTLSTimeout = 5
	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	// plain NATS and the WebSocket listener, verified against the
	// certificate; and the proxy on the HTTP port, which is not TLS
	// itself but reaches the listener over it
	for _, tt := range []struct {
		url string
		ca  []nats.Option
	}{
		{fmt.Sprintf("tls://localhost:%v", opts.NatsPort), []nats.Option{nats.RootCAs(certFile)}},
		{fmt.Sprintf("wss://localhost:%v", opts.NatsWSPort), []nats.Option{nats.RootCAs(certFile)}},
		{fmt.Sprintf("ws://localhost:%v/", opts.HTTPPort), nil},
	} {
		u := tt.url
		c, err := nats.Connect(u, append(tt.ca, nats.Token("tok"), nats.NoReconnect())...)
		if err != nil {
			t.Fatalf("%v: %v", u, err)
		}
		if _, err := c.Request("nodes.root.all", nil, 2*time.Second); err != nil {
			t.Fatalf("%v: request: %v", u, err)
		}
		c.Close()
	}

	// the WebSocket listener no longer answers in the clear
	if c, err := nats.Connect(fmt.Sprintf("ws://localhost:%v", opts.NatsWSPort),
		nats.Token("tok"), nats.NoReconnect()); err == nil {
		c.Close()
		t.Fatal("WebSocket listener accepted a plain connection")
	}

	// an edge client pinning the right certificate connects; one pinning
	// another does not
	connected := func(caCert string) bool {
		nc, err := client.EdgeConnect(client.EdgeOptions{
			URI: fmt.Sprintf("tls://localhost:%v", opts.NatsPort), AuthToken: "tok",
			CACert: caCert,
		})
		if err != nil {
			return false
		}
		defer nc.Close()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if nc.IsConnected() {
				return true
			}
			time.Sleep(50 * time.Millisecond)
		}
		return false
	}
	if !connected(string(certPEM)) {
		t.Fatal("edge client with the server's certificate did not connect")
	}
	if connected(string(otherPEM)) {
		t.Fatal("edge client with another certificate connected")
	}
}

// TestTLSRenewal: a certificate rewritten while the server runs is served
// on the next handshake, and the HTTP port's proxy, which verifies against
// the certificate in use, still reaches the WebSocket listener.
func TestTLSRenewal(t *testing.T) {
	saved := fileCheckInterval
	fileCheckInterval = 0
	t.Cleanup(func() { fileCheckInterval = saved })

	dir := t.TempDir()
	certFile, keyFile, _ := selfSigned(t, dir, "server")
	oldCA := filepath.Join(dir, "old.pem")
	if err := os.Link(certFile, oldCA); err != nil {
		t.Fatal(err)
	}

	opts := TestServerOptions
	opts.AuthToken = "tok"
	opts.NatsTLSCert = certFile
	opts.NatsTLSKey = keyFile
	opts.NatsTLSTimeout = 5
	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	// renew: new certificate and key at the same paths
	if err := os.Remove(certFile); err != nil {
		t.Fatal(err)
	}
	writeSelfSigned(t, certFile, keyFile, "server", time.Now())

	connect := func(u string, ca string) error {
		o := []nats.Option{nats.Token("tok"), nats.NoReconnect()}
		if ca != "" {
			o = append(o, nats.RootCAs(ca))
		}
		c, err := nats.Connect(u, o...)
		if err != nil {
			return err
		}
		defer c.Close()
		_, err = c.Request("nodes.root.all", nil, 2*time.Second)
		return err
	}
	natsURL := fmt.Sprintf("tls://localhost:%v", opts.NatsPort)
	if err := connect(natsURL, certFile); err != nil {
		t.Fatal("renewed certificate not served:", err)
	}
	if err := connect(natsURL, oldCA); err == nil {
		t.Fatal("old certificate still served")
	}
	if err := connect(fmt.Sprintf("ws://localhost:%v/", opts.HTTPPort), ""); err != nil {
		t.Fatal("proxy after renewal:", err)
	}
}
