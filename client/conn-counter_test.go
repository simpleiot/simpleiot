package client

import (
	"io"
	"net"
	"testing"
)

func TestCountingDialer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// echo the first 5 bytes back, then 3 more of our own
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 5)
		if _, err := io.ReadFull(c, buf); err != nil {
			return
		}
		_, _ = c.Write(append(buf, "abc"...))
	}()

	counter := &ConnCounter{}
	d := &countingDialer{Dialer: &net.Dialer{}, Counter: counter}
	c, err := d.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err := c.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(c); err != nil {
		t.Fatal(err)
	}

	if counter.Sent() != 5 || counter.Recv() != 8 {
		t.Fatalf("got sent %v, recv %v; want 5, 8", counter.Sent(), counter.Recv())
	}
}
