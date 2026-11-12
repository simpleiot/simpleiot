package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// signInDefault signs in with the account a new instance starts with,
// retrying while the store is still creating it.
func signInDefault(t *testing.T, base string, want int) (data.Auth, string) {
	t.Helper()

	var status int
	var body []byte
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		http.DefaultClient.CloseIdleConnections()
		resp, err := http.PostForm(base+"/v1/auth", url.Values{
			"email":    {data.DefaultUserEmail},
			"password": {data.DefaultUserPass},
		})
		if err != nil {
			t.Fatal("sign-in:", err)
		}
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		status = resp.StatusCode
		if status == want {
			var auth data.Auth
			_ = json.Unmarshal(body, &auth)
			return auth, string(body)
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("sign-in: got %v %s, want %v", status, body, want)
	return data.Auth{}, ""
}

// TestDefaultPassword: an instance with no token accepts the default
// password and tells the UI about both defaults; one with a token refuses
// it and says why.
func TestDefaultPassword(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		opts := TestServerOptions
		_, _, stop, err := TestServerOpts(opts)
		if err != nil {
			t.Fatal("Error starting test server: ", err)
		}
		defer stop()

		auth, _ := signInDefault(t, "http://localhost:"+opts.HTTPPort, http.StatusOK)
		if auth.Token == "" || !auth.DefaultPassword || !auth.NoAuthToken {
			t.Fatalf("want a token and both notices, got %+v", auth)
		}
	})

	t.Run("token set", func(t *testing.T) {
		opts := TestServerOptions
		opts.AuthToken = "tok"
		nc, _, stop, err := TestServerOpts(opts)
		if err != nil {
			t.Fatal("Error starting test server: ", err)
		}
		defer stop()

		base := "http://localhost:" + opts.HTTPPort
		_, body := signInDefault(t, base, http.StatusForbidden)
		if !strings.Contains(body, data.ErrDefaultPassword.Error()) {
			t.Fatalf("want the reason in the reply, got %q", body)
		}

		// once the password is changed, sign-in works and carries no notice
		token := apiLogin(t, base, nc)
		if token == "" {
			t.Fatal("no token after the password was changed")
		}
	})
}
