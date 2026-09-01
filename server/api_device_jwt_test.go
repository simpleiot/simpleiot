package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// TestAPIDeviceJWT checks that a token signed with an enrolled device key
// reaches the node API, and only the device's own subtree.
func TestAPIDeviceJWT(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = "tok"

	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	base := "http://localhost:" + opts.HTTPPort
	// a keep-alive connection to a previous test's server would reach a
	// server whose NATS client is closed
	http.DefaultClient.CloseIdleConnections()

	// device X with a variable under it, and an unrelated node
	seed, pubKey, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []any{
		client.Device{ID: "dev-x", Parent: root.ID, Description: "device x"},
		client.DeviceCred{ID: "cred-x", Parent: "dev-x", PubKey: pubKey},
		client.Variable{ID: "var-x", Parent: "dev-x", Description: "under x"},
		client.Variable{ID: "var-other", Parent: root.ID, Description: "elsewhere"},
	} {
		if err := client.SendNodeType(nc, n, "test"); err != nil {
			t.Fatal(err)
		}
	}

	// the authorizer indexes the credential from tree changes
	deadline := time.Now().Add(5 * time.Second)
	for {
		code, _ := apiGetNode(t, base+"/v1/nodes/var-x", deviceToken(t, seed))
		if code == http.StatusOK || time.Now().After(deadline) {
			if code != http.StatusOK {
				t.Fatalf("device token refused: %v", code)
			}
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	tok := deviceToken(t, seed)

	// own subtree: read, write points, notify
	code, body := apiGetNode(t, base+"/v1/nodes/dev-x", tok)
	if code != http.StatusOK {
		t.Fatalf("get own device node: %v: %s", code, body)
	}
	var nodes []data.NodeEdge
	if err := json.Unmarshal(body, &nodes); err != nil || len(nodes) == 0 || nodes[0].ID != "dev-x" {
		t.Fatalf("unexpected node reply: %s", body)
	}

	code, body = apiDo(t, http.MethodPost, base+"/v1/nodes/var-x/points", tok,
		data.Points{data.NewPointFloat(data.PointTypeValue, "", 42)})
	if code != http.StatusOK {
		t.Fatalf("post points under own device: %v: %s", code, body)
	}
	vars, err := client.GetNodesType[client.Variable](nc, "all", "var-x")
	if err != nil || len(vars) == 0 || vars[0].Value["0"] != 42 {
		t.Fatalf("point not written: %+v %v", vars, err)
	}

	// anything else is forbidden
	for _, c := range []struct {
		method, path string
	}{
		{http.MethodGet, "/v1/nodes/var-other"},
		{http.MethodPost, "/v1/nodes/var-other/points"},
		{http.MethodGet, "/v1/nodes/" + root.ID},
		{http.MethodGet, "/v1/nodes"},
		{http.MethodPost, "/v1/nodes"},
		{http.MethodDelete, "/v1/nodes/var-x"},
		{http.MethodPost, "/v1/nodes/var-x/parents"},
		{http.MethodPost, "/v1/nodes/cred-x/key"},
	} {
		code, body := apiDo(t, c.method, base+c.path, tok, data.Points{})
		if code != http.StatusForbidden {
			t.Fatalf("%v %v: expected 403, got %v: %s", c.method, c.path, code, body)
		}
	}

	// the device key endpoint is not for devices
	code, _ = apiDo(t, http.MethodGet, base+"/v1/deviceKey", tok, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("device token reached the device key endpoint: %v", code)
	}

	// a key that is not enrolled, a disabled credential, and an expired
	// token are all refused
	strangerSeed, _, _ := client.GenerateDeviceKey()
	code, _ = apiGetNode(t, base+"/v1/nodes/var-x", deviceToken(t, strangerSeed))
	if code != http.StatusUnauthorized {
		t.Fatalf("unknown key accepted: %v", code)
	}

	p := data.NewPointFloat(data.PointTypeDisabled, "", 1)
	p.Origin = "test"
	if err := client.SendNodePoint(nc, "cred-x", p, true); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for {
		code, _ = apiGetNode(t, base+"/v1/nodes/var-x", deviceToken(t, seed))
		if code == http.StatusUnauthorized || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if code != http.StatusUnauthorized {
		t.Fatalf("disabled credential still accepted: %v", code)
	}

	if _, err := client.VerifyDeviceJWT(expiredDeviceToken(t, seed)); err == nil {
		t.Fatal("expired token verified")
	}
}

func deviceToken(t *testing.T, seed string) string {
	t.Helper()
	tok, err := client.DeviceJWT(seed)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// expiredDeviceToken signs a token that expired a minute ago.
func expiredDeviceToken(t *testing.T, seed string) string {
	t.Helper()
	old := client.DeviceJWTLifetime
	defer func() { client.DeviceJWTLifetime = old }()
	client.DeviceJWTLifetime = -time.Minute
	return deviceToken(t, seed)
}

// apiGetNode reads a node; the endpoint takes the parent in the body, and
// "all" means any.
func apiGetNode(t *testing.T, url, token string) (int, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader("all"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, out
}
