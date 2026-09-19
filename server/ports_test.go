package server

import (
	"flag"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
)

// TestPortsFollowNatsPort: SIOT_NATS_PORT alone moves the HTTP port, the
// monitoring port, and the server address the process connects to.
func TestPortsFollowNatsPort(t *testing.T) {
	t.Setenv("SIOT_DATA", t.TempDir())
	t.Setenv("SIOT_NATS_PORT", "5000")
	t.Setenv("SIOT_HTTP_PORT", "")
	t.Setenv("SIOT_NATS_MONITOR_PORT", "")
	t.Setenv("SIOT_NATS_SERVER", "")

	o, err := Args(nil, flag.NewFlagSet("test", flag.ContinueOnError))
	if err != nil {
		t.Fatal(err)
	}

	if o.NatsPort != 5000 || o.HTTPPort != "5001" || o.NatsMonitorPort != 5002 ||
		o.NatsServer != "nats://127.0.0.1:5000" {
		t.Errorf("got NATS %v, HTTP %v, monitor %v, server %v", o.NatsPort,
			o.HTTPPort, o.NatsMonitorPort, o.NatsServer)
	}

	t.Setenv("SIOT_HTTP_PORT", "8080")
	t.Setenv("SIOT_NATS_MONITOR_PORT", "0")

	o, err = Args(nil, flag.NewFlagSet("test", flag.ContinueOnError))
	if err != nil {
		t.Fatal(err)
	}

	if o.HTTPPort != "8080" || o.NatsMonitorPort != 0 {
		t.Errorf("overrides: got HTTP %v, monitor %v", o.HTTPPort, o.NatsMonitorPort)
	}
}

// TestWebSocketFreePort: with no WebSocket port given, the NATS server
// picks one and the HTTP port's proxy finds it.
func TestWebSocketFreePort(t *testing.T) {
	opts := TestServerOptions
	opts.NatsWSPort = 0

	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	c, err := nats.Connect(fmt.Sprintf("ws://localhost:%v/", opts.HTTPPort),
		nats.NoReconnect())
	if err != nil {
		t.Fatal("connect through the proxy:", err)
	}
	defer c.Close()

	if _, err := client.GetNodes(c, "root", "all", "", false); err != nil {
		t.Fatal("request through the proxy:", err)
	}
}
