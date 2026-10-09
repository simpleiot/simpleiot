package client

import (
	"net"
	"sync/atomic"
)

// ConnCounter totals the bytes that cross the network connections dialed
// through it. It counts at the socket, below TLS, so the totals include
// TLS records and handshakes, NATS protocol, and pings: what a metered
// link carries apart from TCP/IP headers.
type ConnCounter struct {
	sent atomic.Uint64
	recv atomic.Uint64
}

// Sent returns the bytes written so far.
func (c *ConnCounter) Sent() uint64 {
	return c.sent.Load()
}

// Recv returns the bytes read so far.
func (c *ConnCounter) Recv() uint64 {
	return c.recv.Load()
}

// SetSent replaces the sent total, for starting from a total saved
// earlier or for a reset.
func (c *ConnCounter) SetSent(v uint64) {
	c.sent.Store(v)
}

// SetRecv replaces the received total, for starting from a total saved
// earlier or for a reset.
func (c *ConnCounter) SetRecv(v uint64) {
	c.recv.Store(v)
}

// countingDialer dials like Dialer and counts the traffic of every
// connection it returns in Counter. It satisfies nats.CustomDialer.
type countingDialer struct {
	Dialer  *net.Dialer
	Counter *ConnCounter
}

func (d *countingDialer) Dial(network, address string) (net.Conn, error) {
	conn, err := d.Dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &countingConn{Conn: conn, counter: d.Counter}, nil
}

type countingConn struct {
	net.Conn
	counter *ConnCounter
}

func (c *countingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.counter.recv.Add(uint64(n))
	return n, err
}

func (c *countingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.counter.sent.Add(uint64(n))
	return n, err
}
