package server

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"golang.org/x/crypto/acme"
)

// certSource hands the current certificate to the NATS listeners and to the
// server's own connections back to those listeners, so a renewal is served
// without a restart.
type certSource interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
	// Leaf is the DER of the certificate GetCertificate returns right now.
	Leaf() ([]byte, error)
}

// newCertSource builds the source the options name, or nil when the
// listeners are in the clear.
func newCertSource(o Options) (certSource, error) {
	files := o.NatsTLSCert != "" || o.NatsTLSKey != ""
	sources := 0
	for _, set := range []bool{files, o.NatsTLSDir != "", o.NatsTLSDomain != ""} {
		if set {
			sources++
		}
	}
	switch {
	case sources > 1:
		return nil, errors.New("set one of SIOT_NATS_TLS_DOMAIN, SIOT_NATS_TLS_DIR, " +
			"or SIOT_NATS_TLS_CERT with SIOT_NATS_TLS_KEY")
	case files && (o.NatsTLSCert == "" || o.NatsTLSKey == ""):
		return nil, errors.New("SIOT_NATS_TLS_CERT and SIOT_NATS_TLS_KEY are set together")
	case o.NatsTLSDomain != "":
		caURL := o.NatsTLSCA
		if caURL == "" {
			caURL = acme.LetsEncryptURL
		}
		dir := filepath.Join(o.DataDir, acmeCacheDirName)
		a := newACMECerts(o.NatsTLSDomain, o.NatsTLSEmail, caURL, dir)
		log.Println(a.describe(dir))
		return a, nil
	case files || o.NatsTLSDir != "":
		src, err := newFileCerts(o.NatsTLSCert, o.NatsTLSKey, o.NatsTLSDir)
		if err != nil {
			return nil, fmt.Errorf("TLS certificate: %w", err)
		}
		return src, nil
	}
	return nil, nil
}

// serverTLSConfig is the configuration the NATS, MQTT, and WebSocket
// listeners serve with.
func serverTLSConfig(src certSource) *tls.Config {
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: src.GetCertificate,
	}
}

// clientTLSConfig is the configuration for a connection the instance makes
// to its own listeners over loopback. The certificate is issued for the
// public name, so a name check would fail, and the server is ours, so
// nothing but the certificate it serves right now is acceptable. Checking
// the source at handshake time, rather than a copy read at start-up, is
// what lets a renewal through.
func clientTLSConfig(src certSource) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		// verification is done by VerifyPeerCertificate below
		InsecureSkipVerify: true, //nolint:gosec
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			leaf, err := src.Leaf()
			if err != nil {
				return err
			}
			if len(rawCerts) == 0 || !bytes.Equal(rawCerts[0], leaf) {
				return errors.New("server presented an unexpected certificate")
			}
			return nil
		},
	}
}

// fileCheckInterval caps how often a handshake looks at the certificate
// files for a change. A renewal happens weeks before expiry, so a minute
// is prompt.
var fileCheckInterval = time.Minute

// fileStamp identifies a version of a certificate and key pair on disk. A
// copied file may keep its modification time, so size counts too.
type fileStamp struct {
	certFile, keyFile string
	certSize, keySize int64
	certMod, keyMod   time.Time
}

// fileCerts serves a certificate and key from files, reloading them when
// they change. The pair is either two fixed paths or found in a directory
// (see findPair).
type fileCerts struct {
	certFile, keyFile string
	dir               string
	now               func() time.Time
	interval          time.Duration

	mu      sync.Mutex
	cert    *tls.Certificate
	stamp   fileStamp
	checked time.Time
}

// newFileCerts loads a certificate and key from the two paths, or from a
// directory when dir is set, and fails if nothing usable is there.
func newFileCerts(certFile, keyFile, dir string) (*fileCerts, error) {
	f := &fileCerts{
		certFile: certFile,
		keyFile:  keyFile,
		dir:      dir,
		now:      time.Now,
		interval: fileCheckInterval,
	}
	if err := f.reload(); err != nil {
		return nil, err
	}
	return f, nil
}

// GetCertificate implements certSource.
func (f *fileCerts) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return f.current()
}

// Leaf implements certSource.
func (f *fileCerts) Leaf() ([]byte, error) {
	c, err := f.current()
	if err != nil {
		return nil, err
	}
	return c.Certificate[0], nil
}

// current returns the loaded pair, looking for a change first when the
// last look was more than interval ago. A change that does not load (a
// renewal half written, for instance) keeps the loaded pair and is tried
// again next time.
func (f *fileCerts) current() (*tls.Certificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.now().Sub(f.checked) >= f.interval {
		if err := f.reload(); err != nil {
			log.Println("TLS certificate not reloaded:", err)
		}
	}
	return f.cert, nil
}

// reload loads the pair if it differs from the loaded one. The caller
// holds the lock, or is the constructor.
func (f *fileCerts) reload() error {
	f.checked = f.now()

	certFile, keyFile := f.certFile, f.keyFile
	if f.dir != "" {
		var err error
		certFile, keyFile, err = findPair(f.dir)
		if err != nil {
			return err
		}
	}
	stamp, err := stampFiles(certFile, keyFile)
	if err != nil {
		return err
	}
	if f.cert != nil && stamp == f.stamp {
		return nil
	}

	cert, err := loadPair(certFile, keyFile)
	if err != nil {
		return err
	}
	if f.cert != nil {
		log.Printf("TLS certificate reloaded from %v, expires %v",
			certFile, cert.Leaf.NotAfter.Format(time.RFC3339))
	}
	f.cert, f.stamp = cert, stamp
	return nil
}

// loadPair parses a certificate and key and checks that they belong
// together.
func loadPair(certFile, keyFile string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", certFile, err)
	}
	if cert.Leaf == nil {
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("%v: %w", certFile, err)
		}
	}
	return &cert, nil
}

func stampFiles(certFile, keyFile string) (fileStamp, error) {
	ci, err := os.Stat(certFile)
	if err != nil {
		return fileStamp{}, err
	}
	ki, err := os.Stat(keyFile)
	if err != nil {
		return fileStamp{}, err
	}
	return fileStamp{
		certFile: certFile, keyFile: keyFile,
		certSize: ci.Size(), keySize: ki.Size(),
		certMod: ci.ModTime(), keyMod: ki.ModTime(),
	}, nil
}

// findPair looks through dir and its subdirectories for certificate and
// key pairs and returns the one issued most recently. A certificate is a
// file named *.crt or fullchain.pem with its key beside it as the same
// stem with .key, or privkey.pem. That covers the directories Caddy and
// certbot write, so dir can be either tool's own store.
func findPair(dir string) (certFile, keyFile string, err error) {
	var newest time.Time
	var found int

	walk := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		key := pairKey(path)
		if key == "" {
			return nil
		}
		if _, err := os.Stat(key); err != nil {
			return nil
		}
		found++
		issued, err := issuedAt(path)
		if err != nil {
			log.Printf("TLS certificate %v skipped: %v", path, err)
			return nil
		}
		if certFile == "" || issued.After(newest) {
			certFile, keyFile, newest = path, key, issued
		}
		return nil
	}
	if err := filepath.WalkDir(dir, walk); err != nil {
		return "", "", fmt.Errorf("%v: %w", dir, err)
	}
	if certFile == "" {
		if found > 0 {
			return "", "", fmt.Errorf("no certificate in %v could be read", dir)
		}
		return "", "", fmt.Errorf("no certificate and key pair in %v "+
			"(looked for *.crt with *.key, or fullchain.pem with privkey.pem)", dir)
	}
	return certFile, keyFile, nil
}

// pairKey returns the path of the key that goes with a certificate file,
// or empty when the file is not a certificate.
func pairKey(certPath string) string {
	base := filepath.Base(certPath)
	switch {
	case base == "fullchain.pem":
		return filepath.Join(filepath.Dir(certPath), "privkey.pem")
	case strings.HasSuffix(base, ".crt"):
		return strings.TrimSuffix(certPath, ".crt") + ".key"
	}
	return ""
}

// issuedAt returns the NotBefore of the first certificate in a PEM file.
func issuedAt(certFile string) (time.Time, error) {
	pemBytes, err := os.ReadFile(certFile)
	if err != nil {
		return time.Time{}, err
	}
	der, err := client.LeafCert(pemBytes)
	if err != nil {
		return time.Time{}, err
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		return time.Time{}, err
	}
	return c.NotBefore, nil
}
