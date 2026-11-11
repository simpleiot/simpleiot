package client

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

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
