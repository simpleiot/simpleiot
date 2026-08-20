package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/internal/mqtttest"
)

// mqttAddr is where the test server listens for MQTT
func mqttAddr(o Options) string {
	return fmt.Sprintf("localhost:%v", o.NatsMQTTPort)
}

func TestMqttDisabledByDefault(t *testing.T) {
	opts := TestServerOptions
	opts.NatsMQTTPort = 0

	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	c, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%v", TestServerOptions.NatsMQTTPort),
		500*time.Millisecond)
	if err == nil {
		_ = c.Close()
		t.Fatal("MQTT port is listening even though it was not configured")
	}
}

func TestMqttToNats(t *testing.T) {
	nc, _, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	sub, err := nc.SubscribeSync("plant.line3.tank")
	if err != nil {
		t.Fatal("Error subscribing: ", err)
	}

	c, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("pub"))
	if err != nil {
		t.Fatal("Error connecting to MQTT: ", err)
	}
	defer c.Close()

	if err := c.Publish("plant/line3/tank", []byte(`{"value":42.1}`), 0, false); err != nil {
		t.Fatal("Error publishing: ", err)
	}

	msg, err := sub.NextMsg(time.Second * 5)
	if err != nil {
		t.Fatal("Error waiting for NATS message: ", err)
	}

	if string(msg.Data) != `{"value":42.1}` {
		t.Fatal("Payload did not survive the trip: ", string(msg.Data))
	}
}

func TestMqttFromNats(t *testing.T) {
	nc, _, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	c, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("sub"))
	if err != nil {
		t.Fatal("Error connecting to MQTT: ", err)
	}
	defer c.Close()

	if err := c.Subscribe("plant/line3/#", 0); err != nil {
		t.Fatal("Error subscribing: ", err)
	}

	// this direction is what a Sparkplug rebirth request depends on
	if err := nc.Publish("plant.line3.tank", []byte("rebirth")); err != nil {
		t.Fatal("Error publishing to NATS: ", err)
	}

	msg, err := c.Next(time.Second * 5)
	if err != nil {
		t.Fatal("Error waiting for MQTT message: ", err)
	}

	if msg.Topic != "plant/line3/tank" {
		t.Fatal("Unexpected topic: ", msg.Topic)
	}

	if string(msg.Payload) != "rebirth" {
		t.Fatal("Unexpected payload: ", string(msg.Payload))
	}
}

func TestMqttQoS1(t *testing.T) {
	nc, _, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	sub, err := nc.SubscribeSync("qos.one")
	if err != nil {
		t.Fatal("Error subscribing: ", err)
	}

	c, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("qos1"))
	if err != nil {
		t.Fatal("Error connecting to MQTT: ", err)
	}
	defer c.Close()

	// Publish returns once the broker has sent PUBACK
	if err := c.Publish("qos/one", []byte("ack me"), 1, false); err != nil {
		t.Fatal("Error publishing at QoS 1: ", err)
	}

	msg, err := sub.NextMsg(time.Second * 5)
	if err != nil {
		t.Fatal("Error waiting for NATS message: ", err)
	}

	if string(msg.Data) != "ack me" {
		t.Fatal("Unexpected payload: ", string(msg.Data))
	}
}

func TestMqttAuth(t *testing.T) {
	opts := TestServerOptions
	opts.AuthToken = "mqtt-test-token"

	_, _, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	addr := mqttAddr(opts)

	if _, err := mqtttest.Dial(addr, mqtttest.ClientID("noauth")); err == nil {
		t.Fatal("Broker accepted a connection with no credentials")
	}

	_, err = mqtttest.Dial(addr, mqtttest.ClientID("badauth"),
		mqtttest.Auth("siot", "wrong-token"))

	var connErr *mqtttest.ConnAckError
	if !errors.As(err, &connErr) {
		t.Fatal("Expected a CONNACK refusal for a wrong password, got: ", err)
	}

	c, err := mqtttest.Dial(addr, mqtttest.ClientID("goodauth"),
		mqtttest.Auth("siot", opts.AuthToken))
	if err != nil {
		t.Fatal("Error connecting with the auth token: ", err)
	}
	defer c.Close()

	if err := c.Ping(); err != nil {
		t.Fatal("Error pinging the broker: ", err)
	}
}

func TestMqttAnonymousWithoutToken(t *testing.T) {
	_, _, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	c, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("anon"))
	if err != nil {
		t.Fatal("Error connecting without credentials: ", err)
	}
	defer c.Close()
}

// TestMqttRetainedSurvivesRestart also exercises the stability of the NATS
// server name, since MQTT state is keyed to it.
func TestMqttRetainedSurvivesRestart(t *testing.T) {
	_, _, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	c, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("retainer"))
	if err != nil {
		t.Fatal("Error connecting to MQTT: ", err)
	}

	if err := c.Publish("plant/line3/state", []byte("running"), 1, true); err != nil {
		t.Fatal("Error publishing retained message: ", err)
	}

	c.Close()
	stop()

	_, _, stop, err = TestServerOptsKeepStore(TestServerOptions)
	if err != nil {
		t.Fatal("Error restarting test server: ", err)
	}
	defer stop()

	c2, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("retained-reader"))
	if err != nil {
		t.Fatal("Error reconnecting to MQTT: ", err)
	}
	defer c2.Close()

	if err := c2.Subscribe("plant/line3/state", 1); err != nil {
		t.Fatal("Error subscribing: ", err)
	}

	msg, err := c2.Next(time.Second * 5)
	if err != nil {
		t.Fatal("Retained message did not survive the restart: ", err)
	}

	if string(msg.Payload) != "running" {
		t.Fatal("Unexpected retained payload: ", string(msg.Payload))
	}
}

// TestMqttStoreIsolation checks that the JetStream streams NATS creates for
// MQTT sessions and retained messages stay out of the way of the streams
// Simple IoT enumerates.
func TestMqttStoreIsolation(t *testing.T) {
	nc, _, stop, err := TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	c, err := mqtttest.Dial(mqttAddr(TestServerOptions), mqtttest.ClientID("isolation"))
	if err != nil {
		t.Fatal("Error connecting to MQTT: ", err)
	}
	defer c.Close()

	if err := c.Publish("plant/line3/level", []byte("1"), 1, true); err != nil {
		t.Fatal("Error publishing: ", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal("Error getting JetStream: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	mqttStreams := 0

	for name := range js.StreamNames(ctx).Name() {
		if strings.HasPrefix(name, "$MQTT_") {
			mqttStreams++
		}
	}

	if mqttStreams == 0 {
		t.Fatal("Expected the broker to have created its own JetStream streams")
	}

	// this is the enumeration every SIOT component uses -- store, sync, and
	// the siot store tooling
	lister := js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))

	for si := range lister.Info() {
		if !strings.HasPrefix(si.Config.Name, "inst_") {
			t.Fatal("SIOT stream enumeration picked up: ", si.Config.Name)
		}
	}

	if err := lister.Err(); err != nil {
		t.Fatal("Error listing streams: ", err)
	}
}
