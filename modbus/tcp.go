package modbus

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// TCPADU defines an ADU for TCP packets
type TCPADU struct {
	PDU
	Address byte
}

// TCP defines an TCP connection
type TCP struct {
	sock    net.Conn
	txID    uint16
	timeout time.Duration
}

// NewTCP creates a new TCP transport
func NewTCP(sock net.Conn, timeout time.Duration) *TCP {
	return &TCP{
		sock:    sock,
		timeout: timeout,
	}
}

func (t *TCP) Read(p []byte) (int, error) {
	t.sock.SetDeadline(time.Now().Add(t.timeout))
	return t.sock.Read(p)
}

func (t *TCP) Write(p []byte) (int, error) {
	t.sock.SetDeadline(time.Now().Add(t.timeout))
	return t.sock.Write(p)
}

// Encode encodes a TCP packet
func (t *TCP) Encode(id byte, pdu PDU) ([]byte, error) {
	// increment transaction ID
	t.txID++
	// bytes 0,1 transaction ID
	ret := make([]byte, len(pdu.Data)+8)
	binary.BigEndian.PutUint16(ret[0:], t.txID)

	// bytes 2,3 protocol identifier

	// bytes 4,5 length
	binary.BigEndian.PutUint16(ret[4:], uint16(len(pdu.Data)+2))

	// byte 6 unit identifier
	ret[6] = id

	// byte 7 function code
	ret[7] = byte(pdu.FunctionCode)

	// byte 8: data
	copy(ret[8:], pdu.Data)
	return ret, nil
}

// Decode decodes a TCP packet
func (t *TCP) Decode(packet []byte) (PDU, error) {
	if len(packet) < 9 {
		return PDU{}, fmt.Errorf("Not enough data for TCP packet: %v", len(packet))
	}

	// FIXME check txID
	ret := PDU{}

	ret.FunctionCode = FunctionCode(packet[7])

	ret.Data = packet[8:]

	return ret, nil
}

// TCPServer listens for new connections and then starts a modbus listener
// on the port.
type TCPServer struct {
	// config
	id         int
	maxClients int
	port       string
	regs       *Regs
	debug      int

	// state
	listener net.Listener
	clients  []net.Conn
	servers  []*Server
	lock     sync.Mutex
	stopped  bool
}

// NewTCPServer starts a new TCP modbus server
func NewTCPServer(id, maxClients int, port string, regs *Regs) (*TCPServer, error) {
	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return nil, err
	}

	return &TCPServer{
		id:         id,
		maxClients: maxClients,
		port:       port,
		regs:       regs,
		listener:   listener,
	}, nil
}

// Listen starts the server and listens for modbus requests
// this function does not return unless an error occurs
// The listen function supports various debug levels:
// 1 - dump packets
// 9 - dump raw data
func (ts *TCPServer) Listen(debug int, errorCallback func(error),
	changesCallback func()) {

	ts.debug = debug

	for {
		sock, err := ts.listener.Accept()
		if err != nil {
			if ts.stopped {
				return
			}
			log.Println("Modbus TCP server: failed to accept connection: ", err)
		}

		log.Println("New Modbus TCP connection")

		if len(ts.clients) < ts.maxClients {
			ts.lock.Lock()
			ts.clients = append(ts.clients, sock)
			transport := NewTCP(sock, 500*time.Millisecond)
			server := NewServer(byte(ts.id), transport, ts.regs)
			go server.Listen(debug, errorCallback,
				changesCallback)
			ts.servers = append(ts.servers, server)
			ts.lock.Unlock()
		} else {
			log.Println("Modbus TCP server: warning reached max conn")
		}
	}
}

// Close stops the server and closes all connections
func (ts *TCPServer) Close() error {
	if ts.debug > 0 {
		log.Println("Modbus TCPServer closing ...")
	}

	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.stopped = true

	var retErr error

	err := ts.listener.Close()

	if err != nil {
		retErr = err
	}

	for _, server := range ts.servers {
		err := server.Close()
		if err != nil {
			retErr = err
		}
	}

	for _, sock := range ts.clients {
		err := sock.Close()
		if err != nil {
			retErr = err
		}
	}

	return retErr
}
