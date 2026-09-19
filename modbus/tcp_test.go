package modbus

import (
	"errors"
	"io"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestTCPEncodeDecode(t *testing.T) {
	pdu := PDU{
		FunctionCode: FuncCodeWriteMultipleCoils,
		Data:         []byte{1, 2, 3},
	}

	tport := NewTCP(nil, 500*time.Millisecond, TransportClient)
	data, err := tport.Encode(1, pdu)

	if err != nil {
		t.Fail()
	}

	_, pdu2, err := tport.Decode(data)

	if err != nil {
		t.Fail()
	}

	if pdu2.FunctionCode != pdu.FunctionCode {
		t.Error("Function code not the same")
	}

	if !reflect.DeepEqual(pdu2.Data, pdu.Data) {
		t.Error("Data compare failed")
	}
}

func dialServer(t *testing.T, ts *TCPServer) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", ts.listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// waitServers waits until the server is serving n connections.
func waitServers(t *testing.T, ts *TCPServer, n int) {
	t.Helper()
	for i := 0; ; i++ {
		ts.lock.Lock()
		got := len(ts.servers)
		ts.lock.Unlock()
		if got == n {
			return
		}
		if i > 200 {
			t.Fatalf("server has %v connections, want %v", got, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestTCPServerMaxClients checks that a connection over maxClients is closed
// instead of left open and unserved, and that a slot frees up when a client
// leaves.
func TestTCPServerMaxClients(t *testing.T) {
	regs := &Regs{}
	regs.AddReg(0, 1)
	_ = regs.WriteReg(0, 0x1234)
	ts, err := NewTCPServer(1, 5, "0", regs, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ts.Close() })
	go ts.Listen(func(err error) { t.Log(err) }, func() {}, func() {})

	var conns []net.Conn
	for range 5 {
		conns = append(conns, dialServer(t, ts))
	}
	waitServers(t, ts, 5)

	sixth := dialServer(t, ts)
	_ = sixth.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := sixth.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("sixth connection: got %v, want EOF", err)
	}

	// closing one of the five frees a slot
	_ = conns[0].Close()
	waitServers(t, ts, 4)

	client := NewClient(NewTCP(dialServer(t, ts), time.Second, TransportClient), 0)
	vals, err := client.ReadHoldingRegs(1, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0] != 0x1234 {
		t.Fatalf("got %v, want [0x1234]", vals)
	}
}

// errListener fails Accept a set number of times and then serves from the
// wrapped listener.
type errListener struct {
	net.Listener
	failures int
	calls    int
}

func (l *errListener) Accept() (net.Conn, error) {
	l.calls++
	if l.calls <= l.failures {
		return nil, errors.New("accept: too many open files")
	}
	return l.Listener.Accept()
}

// TestTCPServerAcceptError checks that an Accept error does not start a
// server on a nil socket and that the listener keeps going afterwards.
func TestTCPServerAcceptError(t *testing.T) {
	regs := &Regs{}
	regs.AddReg(0, 1)
	_ = regs.WriteReg(0, 0x1234)
	ts, err := NewTCPServer(1, 5, "0", regs, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ts.Close() })
	ts.listener = &errListener{Listener: ts.listener, failures: 3}
	go ts.Listen(func(err error) { t.Log(err) }, func() {}, func() {})

	client := NewClient(NewTCP(dialServer(t, ts), time.Second, TransportClient), 0)
	vals, err := client.ReadHoldingRegs(1, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0] != 0x1234 {
		t.Fatalf("got %v, want [0x1234]", vals)
	}
}
