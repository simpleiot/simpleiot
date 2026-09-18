package client

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"syscall"
	"time"
)

// Several clients dial an address taken from a point: the metrics scraper,
// a Shelly device, an ntfy server, a Modbus TCP server, and gpsd. Edge
// devices routinely talk to peers on their LAN, so every address is allowed
// by default. Setting SIOT_OUTBOUND_DENY_PRIVATE refuses loopback,
// link-local, and private addresses, which keeps an instance other people
// can configure, such as one in the cloud, from being used to probe the
// network it sits on. The check runs after name resolution, so a name that
// resolves to a private address is refused as well.
//
// The setting is read once when the process starts.
var outboundDenyPrivate = envBool("SIOT_OUTBOUND_DENY_PRIVATE")

// envBool reads a true/false environment variable.
func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// outboundAddrAllowed reports whether a resolved address may be dialed
// under the policy.
func outboundAddrAllowed(addr netip.Addr, denyPrivate bool) error {
	if !denyPrivate {
		return nil
	}

	addr = addr.Unmap()

	switch {
	case addr.IsLoopback(), addr.IsUnspecified(), addr.IsPrivate(),
		addr.IsLinkLocalUnicast(), addr.IsLinkLocalMulticast(),
		addr.IsInterfaceLocalMulticast(), addr.IsMulticast():
		return fmt.Errorf("%v is a private address and SIOT_OUTBOUND_DENY_PRIVATE is set", addr)
	}

	return nil
}

// outboundControl is the dialer hook that applies the policy. The dialer
// calls it with the address after name resolution and before connecting.
func outboundControl(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}
	return outboundAddrAllowed(ap.Addr(), outboundDenyPrivate)
}

// outboundDialer returns a dialer that applies the outbound policy.
func outboundDialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{Timeout: timeout, Control: outboundControl}
}

// outboundTransport returns an HTTP transport that dials under the
// outbound policy. Redirects go through the same transport, so a public
// server cannot redirect a request to a private address either.
func outboundTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = outboundDialer(30 * time.Second).DialContext
	return t
}

// outboundHTTPClient returns an HTTP client that dials under the outbound
// policy.
func outboundHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: outboundTransport()}
}
