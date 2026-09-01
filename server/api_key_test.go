package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// apiLogin logs the default admin user in and returns the JWT.
func apiLogin(t *testing.T, base string) string {
	t.Helper()

	resp, err := http.PostForm(base+"/v1/auth",
		url.Values{"email": {"admin"}, "password": {"admin"}})
	if err != nil {
		t.Fatal("login:", err)
	}
	defer resp.Body.Close()

	var auth data.Auth
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		t.Fatal("decode login:", err)
	}
	if auth.Token == "" {
		t.Fatal("no token from login")
	}

	return auth.Token
}

func apiDo(t *testing.T, method, url, token string, body any) (int, []byte) {
	t.Helper()

	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, out
}

func TestAPIDeviceKey(t *testing.T) {
	// with a token configured, a request with no credentials is refused
	opts := TestServerOptions
	opts.AuthToken = "tok"

	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	base := "http://localhost:" + TestServerOptions.HTTPPort
	token := apiLogin(t, base)

	// ---- generating a key for a credential node
	dev := client.Device{ID: "dev-1", Parent: root.ID, Description: "dev 1"}
	if err := client.SendNodeType(nc, dev, "test"); err != nil {
		t.Fatal(err)
	}
	cred := client.DeviceCred{ID: "cred-1", Parent: "dev-1", Description: "cred"}
	if err := client.SendNodeType(nc, cred, "test"); err != nil {
		t.Fatal(err)
	}

	code, body := apiDo(t, http.MethodPost, base+"/v1/nodes/cred-1/key", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %v: %s", code, body)
	}

	code, body = apiDo(t, http.MethodPost, base+"/v1/nodes/cred-1/key", token, nil)
	if code != http.StatusOK {
		t.Fatalf("generate key: %v: %s", code, body)
	}

	var gen api.DeviceKeyResponse
	if err := json.Unmarshal(body, &gen); err != nil {
		t.Fatal(err)
	}
	if gen.Seed == "" || gen.PubKey == "" {
		t.Fatalf("expected seed and public key, got %+v", gen)
	}
	if pub, err := client.ParseDeviceKey(gen.Seed); err != nil || pub != gen.PubKey {
		t.Fatalf("seed does not match public key: %v", err)
	}

	got, err := client.GetNodesType[client.DeviceCred](nc, "all", "cred-1")
	if err != nil || len(got) == 0 || got[0].PubKey != gen.PubKey {
		t.Fatalf("public key not stored on the credential: %+v", got)
	}

	// only the public key is in the tree
	all, err := client.GetNodes(nc, "all", "cred-1", "", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range all[0].Points {
		if strings.Contains(p.Txt(), gen.Seed) {
			t.Fatal("seed stored on the credential")
		}
	}

	// not a credential node
	code, body = apiDo(t, http.MethodPost, base+"/v1/nodes/dev-1/key", token, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a device node, got %v: %s", code, body)
	}

	// ---- reading and installing this instance's key
	code, _ = apiDo(t, http.MethodGet, base+"/v1/deviceKey", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %v", code)
	}

	code, body = apiDo(t, http.MethodGet, base+"/v1/deviceKey", token, nil)
	if code != http.StatusOK {
		t.Fatalf("get device key: %v: %s", code, body)
	}
	var cur api.DeviceKeyResponse
	if err := json.Unmarshal(body, &cur); err != nil {
		t.Fatal(err)
	}
	_, pub, err := client.GetDeviceKey(nc)
	if err != nil || cur.PubKey != pub || cur.Seed != "" {
		t.Fatalf("device key reply %+v, expected public key %v and no seed", cur, pub)
	}

	code, body = apiDo(t, http.MethodPost, base+"/v1/deviceKey", token,
		api.DeviceKeyInstall{Seed: gen.Seed})
	if code != http.StatusOK {
		t.Fatalf("install device key: %v: %s", code, body)
	}
	_, pub, err = client.GetDeviceKey(nc)
	if err != nil || pub != gen.PubKey {
		t.Fatalf("installed key %v, expected %v", pub, gen.PubKey)
	}

	code, body = apiDo(t, http.MethodPost, base+"/v1/deviceKey", token,
		api.DeviceKeyInstall{Seed: "junk"})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a bad seed, got %v: %s", code, body)
	}

	fmt.Println("api device key test finished")
}
