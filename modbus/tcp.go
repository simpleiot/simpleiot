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
	sock         net.Conn
	txID         uint16
	timeout      time.Duration
	clientServer TransportClientServer
}

// NewTCP creates a new TCP transport
func NewTCP(sock net.Conn, timeout time.Duration, clientServer TransportClientServer) *TCP {
	return &TCP{
		sock:         sock,
		timeout:      timeout,
		clientServer: clientServer,
	}
}

func (t *TCP) Read(p []byte) (int, error) {
	err := t.sock.SetDeadline(time.Now().Add(t.timeout))
	if err != nil {
		return 0, err
	}
	return t.sock.Read(p)
}

func (t *TCP) Write(p []byte) (int, error) {
	err := t.sock.SetDeadline(time.Now().Add(t.timeout))
	if err != nil {
		return 0, err
	}
	return t.sock.Write(p)
}

// Close connection
func (t *TCP) Close() error {
	return t.sock.Close()
}

// Encode encodes a TCP packet
func (t *TCP) Encode(id byte, pdu PDU) ([]byte, error) {
	// increment transaction ID
	if t.clientServer == TransportClient {
		t.txID++
	}

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
func (t *TCP) Decode(packet []byte) (byte, PDU, error) {
	if len(packet) < 9 {
		return 0, PDU{}, fmt.Errorf("not enough data for TCP packet: %v", len(packet))
	}

	txID := binary.BigEndian.Uint16(packet[:2])

	switch t.clientServer {
	case TransportClient:
		// need to check that echo'd tx is correct
		if txID != t.txID {
			return 0, PDU{}, fmt.Errorf("transaction id not correct, expected: 0x%x, got 0x%x", t.txID, txID)
		}
	case TransportServer:
		// need to store tx to echo back to client on Encode
		t.txID = txID
	}

	id := packet[6]

	pdu := PDU{}
	pdu.FunctionCode = FunctionCode(packet[7])
	pdu.Data = packet[8:]

	return id, pdu, nil
}

// Type returns TransportType
func (t *TCP) Type() TransportType {
	return TransportTypeTCP
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
	servers  []*Server
	lock     sync.Mutex
	stopped  bool
}

// NewTCPServer starts a new TCP modbus server
func NewTCPServer(id, maxClients int, port string, regs *Regs, debug int) (*TCPServer, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	return &TCPServer{
		id:         id,
		maxClients: maxClients,
		port:       port,
		regs:       regs,
		listener:   listener,
		debug:      debug,
	}, nil
}

// Listen starts the server and listens for modbus requests
// this function does not return unless an error occurs
// The listen function supports various debug levels:
// 1 - dump packets
// 9 - dump raw data
func (ts *TCPServer) Listen(errorCallback func(error),
	changesCallback func(), done func()) {
	for {
		sock, err := ts.listener.Accept()
		if err != nil {
			if ts.stopped {
				if ts.debug > 0 {
					log.Println("Modbus TCPServer, stopping listen")
				}
				done()
				return
			}
			log.Println("Modbus TCP server: failed to accept connection:", err)
		}

		if ts.debug > 0 {
			log.Println("New Modbus TCP connection")
		}

		ts.lock.Lock()
		if len(ts.servers) < ts.maxClients {
			transport := NewTCP(sock, 500*time.Millisecond, TransportServer)
			server := NewServer(byte(ts.id), transport, ts.regs, ts.debug)
			ts.servers = append(ts.servers, server)
			go server.Listen(errorCallback,
				changesCallback, func() {
					// TCP server client has disconnected, remove from list
					ts.lock.Lock()
					for i := range ts.servers {
						if ts.servers[i] == server {
							ts.servers[i] = ts.servers[len(ts.servers)-1]
							ts.servers = ts.servers[:len(ts.servers)-1]
							break
						}
					}
					ts.lock.Unlock()
				})
		} else {
			log.Println("Modbus TCP server: warning reached max conn")
		}
		ts.lock.Unlock()
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

	for _, server := range ts.servers {
		err := server.Close()
		if err != nil {
			retErr = err
		}
	}

	err := ts.listener.Close()

	if err != nil {
		retErr = err
	}

	return retErr
}
