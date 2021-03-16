package modbus

import (
	"fmt"
	"io"
	"log"
	"time"
)

// Server defines a server (slave)
// Current Server only supports Modbus RTU,
// but could be expanded to do ASCII and TCP.
type Server struct {
	id        byte
	transport Transport
	regs      *Regs
	chDone    chan bool
}

// NewServer creates a new server instance
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewServer(id byte, transport Transport, regs *Regs) *Server {
	return &Server{
		id:        id,
		transport: transport,
		regs:      regs,
		chDone:    make(chan bool),
	}
}

// Close stops the listening channel
func (s *Server) Close() error {
	s.chDone <- true
	return nil
}

// Listen starts the server and listens for modbus requests
// this function does not return unless an error occurs
// The listen function supports various debug levels:
// 1 - dump packets
// 9 - dump raw data
func (s *Server) Listen(debug int, errorCallback func(error),
	changesCallback func()) {
	for {
		select {
		case <-s.chDone:
			log.Println("Exiting modbus server listen")
			return
		default:
		}
		buf := make([]byte, 200)
		cnt, err := s.transport.Read(buf)
		fmt.Println("CLIFF: Listen read returned: ", cnt, err)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading modbus port: ", err)
			}

			// FIXME -- do we want to keep this long term?
			// to keep the system from spinning if a connection is destroyed
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if cnt <= 0 {
			continue
		}

		// parse packet from server
		packet := buf[:cnt]

		if debug >= 9 {
			fmt.Println("Modbus server rx: ", HexDump(packet))
		}

		fmt.Printf("CLIFF: packet[0]: %v, s.id: %v\n", packet[0], s.id)

		if packet[0] != s.id {
			// packet is not for this device
			continue
		}

		req, err := s.transport.Decode(packet)
		fmt.Println("CLIFF: Decode returned: ", req, err)
		if err != nil {
			errorCallback(err)
			continue
		}

		fmt.Println("CLIFF: debug: ", debug)

		if debug >= 1 {
			fmt.Println("Modbus server req: ", req)
		}

		regsChanged, resp, err := req.ProcessRequest(s.regs)
		if regsChanged {
			changesCallback()
		}

		if err != nil {
			errorCallback(err)
			continue
		}

		if debug >= 1 {
			fmt.Println("Modbus server resp: ", resp)
		}

		respRtu, err := s.transport.Encode(s.id, resp)
		if err != nil {
			errorCallback(err)
			continue
		}

		if debug >= 9 {
			fmt.Println("Modbus server tx: ", HexDump(respRtu))
		}

		_, err = s.transport.Write(respRtu)
		if err != nil {
			errorCallback(err)
			continue
		}
	}
}
