package client

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// DeviceJWTLifetime is how long a device's HTTP token is good for. A device
// signs a fresh one per request or per few minutes; a stolen token is
// useful for no longer than this. Tests shorten it.
var DeviceJWTLifetime = 5 * time.Minute

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
	// Pending is set on a credential a device enrolled itself with, until
	// an operator approves it.
	Pending bool `point:"pending"`
}

// EnrollToken lets devices that hold the token ask an upstream for a
// credential. Only a hash of the token is stored.
type EnrollToken struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	TokenHash   string `point:"tokenHash"`
	Disabled    bool   `point:"disabled"`
	// AutoApprove creates enrolled credentials live rather than pending.
	AutoApprove bool `point:"autoApprove"`
	// Expires is a Unix time after which the token is refused; zero never
	// expires.
	Expires float64 `point:"expires"`
}

// SubjectEnrollRequest is where a device asks for a credential. A
// connection made with an enrollment token may publish here and nowhere
// else.
const SubjectEnrollRequest = "enroll.request"

// EnrollRequest is what a device sends to enroll.
type EnrollRequest struct {
	Token       string `json:"token"`
	DeviceID    string `json:"deviceID"`
	PubKey      string `json:"pubKey"`
	Description string `json:"description,omitempty"`
}

// Enrollment outcomes.
const (
	EnrollApproved = "approved"
	EnrollPending  = "pending"
)

// EnrollReply is the upstream's answer.
type EnrollReply struct {
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// GenerateEnrollToken makes a new enrollment token and its hash.
func GenerateEnrollToken() (token, hash string, err error) {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}

	token = "ET" + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])

	return token, HashEnrollToken(token), nil
}

// HashEnrollToken is what an enrollToken node stores.
func HashEnrollToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Enroll connects to an upstream with an enrollment token and asks for a
// credential for this device's key.
func Enroll(uri string, req EnrollRequest) (EnrollReply, error) {
	uri, err := sanitizeURI(uri)
	if err != nil {
		return EnrollReply{}, err
	}

	nc, err := nats.Connect(uri, nats.Token(req.Token), nats.Timeout(30*time.Second),
		nats.NoReconnect())
	if err != nil {
		return EnrollReply{}, fmt.Errorf("error connecting with enrollment token: %w", err)
	}
	defer nc.Close()

	body, err := json.Marshal(req)
	if err != nil {
		return EnrollReply{}, err
	}

	msg, err := nc.Request(SubjectEnrollRequest, body, 30*time.Second)
	if err != nil {
		return EnrollReply{}, fmt.Errorf("enrollment request: %w", err)
	}

	var reply EnrollReply
	if err := json.Unmarshal(msg.Data, &reply); err != nil {
		return EnrollReply{}, err
	}
	if reply.Error != "" {
		return reply, errors.New(reply.Error)
	}

	return reply, nil
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

// DeviceJWT signs a short-lived token with the device key for the HTTP API.
// The upstream verifies the signature against the issuer, which is the
// device's public key, and looks that key up the same way it does for a
// NATS connection.
func DeviceJWT(seed string) (string, error) {
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return "", fmt.Errorf("not a device key seed: %w", err)
	}

	pubKey, err := kp.PublicKey()
	if err != nil {
		return "", err
	}

	claims := jwt.NewGenericClaims(pubKey)
	claims.Expires = time.Now().Add(DeviceJWTLifetime).Unix()

	return claims.Encode(kp)
}

// VerifyDeviceJWT checks a device token's signature and expiry and returns
// the public key that signed it. Whether that key is enrolled is the
// caller's question.
func VerifyDeviceJWT(token string) (pubKey string, err error) {
	claims, err := jwt.DecodeGeneric(token)
	if err != nil {
		return "", err
	}

	if !nkeys.IsValidPublicUserKey(claims.Issuer) {
		return "", errors.New("token not signed by a device key")
	}

	now := time.Now().Unix()
	if claims.Expires == 0 || claims.Expires > now+int64(DeviceJWTLifetime.Seconds())+60 {
		return "", errors.New("token has no expiry or expires too far out")
	}
	if now > claims.Expires {
		return "", errors.New("token expired")
	}

	return claims.Issuer, nil
}
