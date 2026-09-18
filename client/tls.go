package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// PinnedTLSConfig returns a TLS client configuration that accepts exactly
// the certificate in pemBytes and nothing else. It is for a connection an
// instance makes to its own NATS server over loopback: the certificate is
// issued for the public name, so a name check would fail, and the server
// is ours, so no other certificate is acceptable.
func PinnedTLSConfig(pemBytes []byte) (*tls.Config, error) {
	leaf, err := LeafCert(pemBytes)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		// verification is done by VerifyPeerCertificate below, against
		// the one certificate that is acceptable
		InsecureSkipVerify: true, //nolint:gosec
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 || !bytes.Equal(rawCerts[0], leaf) {
				return errors.New("server presented an unexpected certificate")
			}
			return nil
		},
	}, nil
}

// LeafCert returns the first certificate in a PEM file, in DER form.
func LeafCert(pemBytes []byte) ([]byte, error) {
	for {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			return nil, errors.New("no certificate in file")
		}
		if block.Type == "CERTIFICATE" {
			if _, err := x509.ParseCertificate(block.Bytes); err != nil {
				return nil, fmt.Errorf("error parsing certificate: %w", err)
			}
			return block.Bytes, nil
		}
	}
}
