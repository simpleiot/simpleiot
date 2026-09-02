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
	"time"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// apiLogin logs the default admin user in and returns the JWT. The admin
// user is created shortly after the store starts, so a login right after
// start-up is retried.
func apiLogin(t *testing.T, base string) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	var last string

	for time.Now().Before(deadline) {
		resp, err := http.PostForm(base+"/v1/auth",
			url.Values{"email": {"admin"}, "password": {"admin"}})
		if err != nil {
			t.Fatal("login:", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var auth data.Auth
		if resp.StatusCode == http.StatusOK &&
			json.Unmarshal(body, &auth) == nil && auth.Token != "" {
			return auth.Token
		}

		last = fmt.Sprintf("%v: %s", resp.StatusCode, body)
		http.DefaultClient.CloseIdleConnections()
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatal("login never succeeded, last reply:", last)
	return ""
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

func TestAPIEnrollTokenKey(t *testing.T) {
	// with a token configured, a request with no credentials is refused
	opts := TestServerOptions
	opts.AuthToken = "tok"

	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	base := "http://localhost:" + TestServerOptions.HTTPPort
	// a keep-alive connection to a previous test's server would reach a
	// server whose NATS client is closed
	http.DefaultClient.CloseIdleConnections()
	token := apiLogin(t, base)

	et := client.EnrollToken{ID: "et-1", Parent: root.ID, Description: "fleet"}
	if err := client.SendNodeType(nc, et, "test"); err != nil {
		t.Fatal(err)
	}
	cred := client.DeviceCred{ID: "cred-1", Parent: root.ID, Description: "cred"}
	if err := client.SendNodeType(nc, cred, "test"); err != nil {
		t.Fatal(err)
	}

	code, body := apiDo(t, http.MethodPost, base+"/v1/nodes/et-1/key", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %v: %s", code, body)
	}

	code, body = apiDo(t, http.MethodPost, base+"/v1/nodes/et-1/key", token, nil)
	if code != http.StatusOK {
		t.Fatalf("generate token: %v: %s", code, body)
	}

	var gen api.KeyResponse
	if err := json.Unmarshal(body, &gen); err != nil {
		t.Fatal(err)
	}
	if gen.Token == "" {
		t.Fatalf("expected a token, got %+v", gen)
	}

	got, err := client.GetNodesType[client.EnrollToken](nc, "all", "et-1")
	if err != nil || len(got) == 0 || got[0].TokenHash != client.HashEnrollToken(gen.Token) {
		t.Fatalf("token hash not stored on the node: %+v", got)
	}

	// only the hash is in the tree
	all, err := client.GetNodes(nc, "all", "et-1", "", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range all[0].Points {
		if strings.Contains(p.Txt(), gen.Token) {
			t.Fatal("token stored on the node")
		}
	}

	// only enrollment token nodes generate anything
	code, body = apiDo(t, http.MethodPost, base+"/v1/nodes/cred-1/key", token, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a credential node, got %v: %s", code, body)
	}

	fmt.Println("api enroll token test finished")
}
