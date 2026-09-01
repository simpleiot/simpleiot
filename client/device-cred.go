package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// Subjects the server answers for this instance's device key. The bus is
// trusted to the same degree as the shared token: whoever can ask here can
// also read the key file.
const (
	// SubjectDeviceKey replies with the key as a DeviceKey.
	SubjectDeviceKey = "auth.deviceKey"
	// SubjectInstallDeviceKey takes a seed and replaces the key.
	SubjectInstallDeviceKey = "auth.installDeviceKey"
)

// DeviceKey is this instance's key, as answered on SubjectDeviceKey.
type DeviceKey struct {
	Seed   string `json:"seed,omitempty"`
	PubKey string `json:"pubKey,omitempty"`
	Error  string `json:"error,omitempty"`
}

// DeviceCred is a credential for one device, kept under the device node on
// the upstream. Only the public key is stored; the device holds the seed
// on its sync node. The authorizer maintains LastConnect and Connected.
type DeviceCred struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	PubKey      string  `point:"pubKey"`
	Disabled    bool    `point:"disabled"`
	LastConnect float64 `point:"lastConnect"`
	Connected   bool    `point:"connected"`
}

// GetDeviceKey returns this instance's device key.
func GetDeviceKey(nc *nats.Conn) (seed, pubKey string, err error) {
	msg, err := nc.Request(SubjectDeviceKey, nil, 5*time.Second)
	if err != nil {
		return "", "", err
	}

	var k DeviceKey
	if err := json.Unmarshal(msg.Data, &k); err != nil {
		return "", "", err
	}
	if k.Error != "" {
		return "", "", errors.New(k.Error)
	}

	return k.Seed, k.PubKey, nil
}

// InstallDeviceKey replaces this instance's device key with one issued by
// an upstream and returns its public key. Sync nodes that use the key
// reconnect with it.
func InstallDeviceKey(nc *nats.Conn, seed string) (pubKey string, err error) {
	msg, err := nc.Request(SubjectInstallDeviceKey, []byte(seed), 5*time.Second)
	if err != nil {
		return "", err
	}

	var k DeviceKey
	if err := json.Unmarshal(msg.Data, &k); err != nil {
		return "", err
	}
	if k.Error != "" {
		return "", errors.New(k.Error)
	}

	return k.PubKey, nil
}

// ParseDeviceKey checks a seed and returns its public key.
func ParseDeviceKey(seed string) (pubKey string, err error) {
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return "", fmt.Errorf("not a device key seed: %w", err)
	}

	pubKey, err = kp.PublicKey()
	if err != nil {
		return "", err
	}

	if !nkeys.IsValidPublicUserKey(pubKey) {
		return "", errors.New("not a user key")
	}

	return pubKey, nil
}

// GenerateDeviceKey creates a new device credential and returns the seed,
// which the device keeps, and the public key, which the upstream stores.
func GenerateDeviceKey() (seed, pubKey string, err error) {
	kp, err := nkeys.CreateUser()
	if err != nil {
		return "", "", fmt.Errorf("error creating key: %w", err)
	}

	s, err := kp.Seed()
	if err != nil {
		return "", "", fmt.Errorf("error reading seed: %w", err)
	}

	pubKey, err = kp.PublicKey()
	if err != nil {
		return "", "", fmt.Errorf("error reading public key: %w", err)
	}

	return string(s), pubKey, nil
}
