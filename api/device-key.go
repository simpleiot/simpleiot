package api

import (
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
)

// DeviceKeyResponse is the reply to key requests. Seed is present only when a
// key was generated, and is never stored.
type DeviceKeyResponse struct {
	PubKey string `json:"pubKey,omitempty"`
	Seed   string `json:"seed,omitempty"`
	// Token is present when an enrollment token was generated, once.
	Token string `json:"token,omitempty"`
}

// DeviceKeyInstall is the body of a key install.
type DeviceKeyInstall struct {
	Seed string `json:"seed"`
}

// DeviceKey handles /v1/deviceKey: the key this instance connects to
// upstreams with. GET returns the public key; POST installs a seed issued by
// an upstream, which is how a key reaches a device through its UI without
// ever being a point.
type DeviceKey struct {
	check     RequestValidator
	nc        *nats.Conn
	authToken string
}

// NewDeviceKeyHandler returns a handler for the device key.
func NewDeviceKeyHandler(v RequestValidator, authToken string, nc *nats.Conn) http.Handler {
	return &DeviceKey{check: v, nc: nc, authToken: authToken}
}

func (h *DeviceKey) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != h.authToken {
		if valid, _ := h.check.Valid(req); !valid {
			http.Error(res, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	switch req.Method {
	case http.MethodGet:
		_, pubKey, err := client.GetDeviceKey(h.nc)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := encode(res, DeviceKeyResponse{PubKey: pubKey}); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		var in DeviceKeyInstall
		if err := decode(req.Body, &in); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		pubKey, err := client.InstallDeviceKey(h.nc, in.Seed)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if err := encode(res, DeviceKeyResponse{PubKey: pubKey}); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

	default:
		http.Error(res, "invalid method", http.StatusMethodNotAllowed)
	}
}
