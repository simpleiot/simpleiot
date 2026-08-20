// Package mqtttest is a small MQTT 3.1.1 client used by the Simple IoT tests
// to exercise the built-in broker. It speaks only the packets the tests need
// -- CONNECT, PUBLISH, SUBSCRIBE, PINGREQ and DISCONNECT, with their
// acknowledgements -- which keeps the module free of an MQTT client
// dependency and checks the broker against the wire format rather than
// against another library's idea of it.
package mqtttest

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// MQTT control packet types, in the high nibble of the fixed header
const (
	pktConnect     = 0x10
	pktConnAck     = 0x20
	pktPublish     = 0x30
	pktPubAck      = 0x40
	pktSubscribe   = 0x80
	pktSubAck      = 0x90
	pktPingReq     = 0xC0
	pktPingResp    = 0xD0
	pktDisconnect  = 0xE0
	pktTypeMask    = 0xF0
	pubFlagRetain  = 0x01
	pubFlagQoSMask = 0x06
)

// DefaultTimeout bounds how long a test waits for the broker to answer.
const DefaultTimeout = 5 * time.Second

// Message is a message delivered to a subscriber.
type Message struct {
	Topic   string
	Payload []byte
	QoS     byte
	Retain  bool
}

// ConnAckError reports a CONNECT that the broker refused, carrying the return
// code from the CONNACK packet so a test can tell "bad password" from
// "not authorized".
type ConnAckError struct {
	Code byte
}

func (e *ConnAckError) Error() string {
	desc := map[byte]string{
		1: "unacceptable protocol version",
		2: "identifier rejected",
		3: "server unavailable",
		4: "bad user name or password",
		5: "not authorized",
	}[e.Code]

	if desc == "" {
		desc = "unknown"
	}

	return fmt.Sprintf("MQTT connection refused: %v (%v)", desc, e.Code)
}

// Client is a connection to an MQTT broker.
type Client struct {
	conn net.Conn
	r    *bufio.Reader

	writeMu sync.Mutex

	msgs   chan Message
	acks   chan uint16
	subs   chan uint16
	pings  chan struct{}
	closed chan struct{}

	readErrMu sync.Mutex
	readErr   error

	idMu   sync.Mutex
	nextID uint16
}

type options struct {
	clientID     string
	username     string
	password     string
	cleanSession bool
	keepAlive    uint16
}

// Option configures a connection.
type Option func(*options)

// ClientID sets the MQTT client identifier, which is what the broker keys a
// session to.
func ClientID(id string) Option {
	return func(o *options) { o.clientID = id }
}

// Auth sets the user name and password sent in the CONNECT packet. The
// built-in broker takes the Simple IoT auth token as the password; MQTT
// requires a user name alongside it, and its value is not checked.
func Auth(username, password string) Option {
	return func(o *options) {
		o.username = username
		o.password = password
	}
}

// CleanSession asks the broker to start a fresh session rather than resume a
// stored one. It is on by default.
func CleanSession(clean bool) Option {
	return func(o *options) { o.cleanSession = clean }
}

// Dial connects to a broker at addr and completes the MQTT handshake.
func Dial(addr string, opts ...Option) (*Client, error) {
	o := options{
		clientID:     "mqtttest",
		cleanSession: true,
		keepAlive:    60,
	}

	for _, opt := range opts {
		opt(&o)
	}

	conn, err := net.DialTimeout("tcp", addr, DefaultTimeout)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn:   conn,
		r:      bufio.NewReader(conn),
		msgs:   make(chan Message, 64),
		acks:   make(chan uint16, 8),
		subs:   make(chan uint16, 8),
		pings:  make(chan struct{}, 8),
		closed: make(chan struct{}),
		nextID: 1,
	}

	if err := c.connect(o); err != nil {
		_ = conn.Close()
		return nil, err
	}

	go c.readLoop()

	return c, nil
}

func (c *Client) connect(o options) error {
	var flags byte

	if o.cleanSession {
		flags |= 0x02
	}

	if o.username != "" {
		flags |= 0x80
	}

	if o.password != "" {
		flags |= 0x40
	}

	var vh bytes.Buffer
	writeString(&vh, "MQTT")
	vh.WriteByte(4) // protocol level 4 == MQTT 3.1.1
	vh.WriteByte(flags)
	_ = binary.Write(&vh, binary.BigEndian, o.keepAlive)
	writeString(&vh, o.clientID)

	if o.username != "" {
		writeString(&vh, o.username)
	}

	if o.password != "" {
		writeString(&vh, o.password)
	}

	if err := c.writePacket(pktConnect, vh.Bytes()); err != nil {
		return err
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return err
	}

	hdr, payload, err := c.readPacket()
	if err != nil {
		return err
	}

	if err := c.conn.SetReadDeadline(time.Time{}); err != nil {
		return err
	}

	if hdr&pktTypeMask != pktConnAck {
		return fmt.Errorf("expected CONNACK, got packet type 0x%X", hdr&pktTypeMask)
	}

	if len(payload) < 2 {
		return fmt.Errorf("short CONNACK: %v bytes", len(payload))
	}

	if payload[1] != 0 {
		return &ConnAckError{Code: payload[1]}
	}

	return nil
}

// Publish sends a message. QoS 1 waits for the broker's PUBACK, so a test that
// publishes at QoS 1 knows the broker has the message when this returns.
func (c *Client) Publish(topic string, payload []byte, qos byte, retain bool) error {
	if qos > 1 {
		return fmt.Errorf("mqtttest publishes at QoS 0 or 1, not %v", qos)
	}

	hdr := byte(pktPublish) | qos<<1

	if retain {
		hdr |= pubFlagRetain
	}

	var b bytes.Buffer
	writeString(&b, topic)

	var id uint16

	if qos > 0 {
		id = c.packetID()
		_ = binary.Write(&b, binary.BigEndian, id)
	}

	b.Write(payload)

	if err := c.writePacket(hdr, b.Bytes()); err != nil {
		return err
	}

	if qos == 0 {
		return nil
	}

	return c.await(c.acks, id, "PUBACK")
}

// Subscribe subscribes to a topic filter and waits for the SUBACK.
func (c *Client) Subscribe(filter string, qos byte) error {
	id := c.packetID()

	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, id)
	writeString(&b, filter)
	b.WriteByte(qos)

	// the SUBSCRIBE fixed header carries reserved bits that must be 0b0010
	if err := c.writePacket(pktSubscribe|0x02, b.Bytes()); err != nil {
		return err
	}

	return c.await(c.subs, id, "SUBACK")
}

// Ping sends a PINGREQ and waits for the PINGRESP.
func (c *Client) Ping() error {
	if err := c.writePacket(pktPingReq, nil); err != nil {
		return err
	}

	select {
	case <-c.pings:
		return nil
	case <-c.closed:
		return c.err()
	case <-time.After(DefaultTimeout):
		return fmt.Errorf("timeout waiting for PINGRESP")
	}
}

// Next returns the next message delivered to this client, or an error if none
// arrives within timeout.
func (c *Client) Next(timeout time.Duration) (Message, error) {
	select {
	case m := <-c.msgs:
		return m, nil
	case <-time.After(timeout):
		return Message{}, fmt.Errorf("timeout waiting for a message")
	}
}

// Close sends DISCONNECT and closes the connection.
func (c *Client) Close() error {
	_ = c.writePacket(pktDisconnect, nil)
	return c.conn.Close()
}

func (c *Client) packetID() uint16 {
	c.idMu.Lock()
	defer c.idMu.Unlock()

	id := c.nextID
	c.nextID++

	if c.nextID == 0 {
		c.nextID = 1
	}

	return id
}

func (c *Client) await(ch chan uint16, id uint16, what string) error {
	deadline := time.After(DefaultTimeout)

	for {
		select {
		case got := <-ch:
			if got == id {
				return nil
			}
		case <-c.closed:
			return c.err()
		case <-deadline:
			return fmt.Errorf("timeout waiting for %v", what)
		}
	}
}

func (c *Client) err() error {
	c.readErrMu.Lock()
	defer c.readErrMu.Unlock()

	if c.readErr == nil {
		return io.ErrUnexpectedEOF
	}

	return c.readErr
}

func (c *Client) readLoop() {
	for {
		hdr, payload, err := c.readPacket()
		if err != nil {
			c.readErrMu.Lock()
			c.readErr = err
			c.readErrMu.Unlock()
			close(c.closed)
			return
		}

		switch hdr & pktTypeMask {
		case pktPublish:
			m, id, err := parsePublish(hdr, payload)
			if err != nil {
				continue
			}

			if m.QoS == 1 {
				var b bytes.Buffer
				_ = binary.Write(&b, binary.BigEndian, id)
				_ = c.writePacket(pktPubAck, b.Bytes())
			}

			select {
			case c.msgs <- m:
			default:
			}

		case pktPubAck:
			if len(payload) >= 2 {
				select {
				case c.acks <- binary.BigEndian.Uint16(payload):
				default:
				}
			}

		case pktSubAck:
			if len(payload) >= 2 {
				select {
				case c.subs <- binary.BigEndian.Uint16(payload):
				default:
				}
			}

		case pktPingResp:
			select {
			case c.pings <- struct{}{}:
			default:
			}
		}
	}
}

func parsePublish(hdr byte, payload []byte) (Message, uint16, error) {
	qos := (hdr & pubFlagQoSMask) >> 1

	m := Message{
		QoS:    qos,
		Retain: hdr&pubFlagRetain != 0,
	}

	if len(payload) < 2 {
		return m, 0, fmt.Errorf("short PUBLISH")
	}

	n := int(binary.BigEndian.Uint16(payload))
	if len(payload) < 2+n {
		return m, 0, fmt.Errorf("short PUBLISH topic")
	}

	m.Topic = string(payload[2 : 2+n])
	rest := payload[2+n:]

	var id uint16

	if qos > 0 {
		if len(rest) < 2 {
			return m, 0, fmt.Errorf("short PUBLISH packet ID")
		}
		id = binary.BigEndian.Uint16(rest)
		rest = rest[2:]
	}

	m.Payload = append([]byte(nil), rest...)

	return m, id, nil
}

func (c *Client) writePacket(hdr byte, payload []byte) error {
	var b bytes.Buffer
	b.WriteByte(hdr)
	writeLength(&b, len(payload))
	b.Write(payload)

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	_, err := c.conn.Write(b.Bytes())

	return err
}

func (c *Client) readPacket() (byte, []byte, error) {
	hdr, err := c.r.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	n, err := readLength(c.r)
	if err != nil {
		return 0, nil, err
	}

	payload := make([]byte, n)
	if _, err := io.ReadFull(c.r, payload); err != nil {
		return 0, nil, err
	}

	return hdr, payload, nil
}

func writeString(b *bytes.Buffer, s string) {
	_ = binary.Write(b, binary.BigEndian, uint16(len(s)))
	b.WriteString(s)
}

// writeLength writes an MQTT remaining-length field, a variable length integer
// of up to four bytes with the high bit marking a continuation.
func writeLength(b *bytes.Buffer, n int) {
	for {
		d := byte(n % 128)
		n /= 128

		if n > 0 {
			d |= 0x80
		}

		b.WriteByte(d)

		if n == 0 {
			return
		}
	}
}

func readLength(r *bufio.Reader) (int, error) {
	var n, mult int

	for i := 0; i < 4; i++ {
		d, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		n += int(d&0x7F) << mult

		if d&0x80 == 0 {
			return n, nil
		}

		mult += 7
	}

	return 0, fmt.Errorf("malformed remaining length")
}
