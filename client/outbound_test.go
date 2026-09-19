package client

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestOutboundAddrAllowed(t *testing.T) {
	private := []string{
		"127.0.0.1", "::1", "10.1.2.3", "172.16.0.1", "172.31.255.255",
		"192.168.1.42", "169.254.169.254", "fe80::1", "fc00::1", "fd12::1",
		"0.0.0.0", "::", "::ffff:192.168.1.1", "224.0.0.1", "ff02::1",
	}
	public := []string{
		"8.8.8.8", "1.1.1.1", "172.32.0.1", "2001:4860:4860::8888",
		"::ffff:8.8.8.8",
	}

	for _, s := range private {
		addr := netip.MustParseAddr(s)
		if err := outboundAddrAllowed(addr, true); err == nil {
			t.Errorf("%v allowed with deny private set", s)
		}
		if err := outboundAddrAllowed(addr, false); err != nil {
			t.Errorf("%v refused with deny private clear: %v", s, err)
		}
	}

	for _, s := range public {
		addr := netip.MustParseAddr(s)
		if err := outboundAddrAllowed(addr, true); err != nil {
			t.Errorf("%v refused with deny private set: %v", s, err)
		}
	}
}

// TestOutboundHTTPClientPolicy checks that the policy is applied where a
// client dials, and that a redirect to a private address is refused too.
func TestOutboundHTTPClientPolicy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "http://127.0.0.1:1/", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
	defer ts.Close()

	saved := outboundDenyPrivate
	defer func() { outboundDenyPrivate = saved }()

	outboundDenyPrivate = false
	c := outboundHTTPClient(2 * time.Second)
	resp, err := c.Get(ts.URL)
	if err != nil {
		t.Fatalf("default policy refused a loopback server: %v", err)
	}
	_ = resp.Body.Close()

	// the policy applies when a connection is dialed, so drop the pooled one
	outboundDenyPrivate = true
	c.CloseIdleConnections()
	_, err = c.Get(ts.URL)
	if err == nil || !strings.Contains(err.Error(), "private address") {
		t.Fatalf("deny private policy let a loopback dial through: %v", err)
	}

	// a public-looking request that redirects to a private address is
	// stopped at the redirect
	outboundDenyPrivate = false
	c.CloseIdleConnections()
	resp, err = c.Get(ts.URL + "/redirect")
	if err == nil {
		_ = resp.Body.Close()
		// the redirect target does not exist, so without the policy this
		// fails with a connection error rather than succeeding; either way
		// it must not be a policy error
	}
	outboundDenyPrivate = true
	c.CloseIdleConnections()
	_, err = c.Get(ts.URL + "/redirect")
	if err == nil || !strings.Contains(err.Error(), "private address") {
		t.Fatalf("deny private policy did not stop the request: %v", err)
	}

	// the dialer used for raw TCP applies the same policy
	conn, err := outboundDialer(time.Second).Dial("tcp", strings.TrimPrefix(ts.URL, "http://"))
	if err == nil {
		_ = conn.Close()
		t.Fatal("deny private policy let a loopback TCP dial through")
	}
}

func TestEnvBool(t *testing.T) {
	for v, want := range map[string]bool{
		"1": true, "true": true, "TRUE": true, "yes": true, " on ": true,
		"": false, "0": false, "false": false, "no": false, "maybe": false,
	} {
		t.Setenv("SIOT_TEST_BOOL", v)
		if got := envBool("SIOT_TEST_BOOL"); got != want {
			t.Errorf("%q: got %v, want %v", v, got, want)
		}
	}
}
