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
	debug     int
}

// NewServer creates a new server instance
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewServer(id byte, transport Transport, regs *Regs, debug int) *Server {
	return &Server{
		id:        id,
		transport: transport,
		regs:      regs,
		chDone:    make(chan bool),
		debug:     debug,
	}
}

// Close stops the listening channel
func (s *Server) Close() error {
	s.transport.Close()
	s.chDone <- true
	return nil
}

// Listen starts the server and listens for modbus requests
// this function does not return unless an error occurs
// The listen function supports various debug levels:
// 1 - dump packets
// 9 - dump raw data
func (s *Server) Listen(errorCallback func(error),
	changesCallback func(), done func()) {
	for {
		select {
		case <-s.chDone:
			// FIXME is there a way to detect closed port with serial so
			// we don't need this channel any more?
			log.Println("Exiting modbus server listen")
			done()
			return
		default:
		}
		buf := make([]byte, 200)
		cnt, err := s.transport.Read(buf)
		if err != nil {
			if err != io.EOF && s.transport.Type() == TransportTypeRTU {
				// only print errors for RTU for now as we get timeout
				// errors with TCP
				log.Println("Error reading modbus port: ", err)
			}

			if err == io.EOF && s.transport.Type() == TransportTypeTCP {
				// with TCP, EOF means we are done with this connection
				log.Println("Modbus TCP client disconnected")
				done()
				return
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

		if s.debug >= 9 {
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

		fmt.Println("CLIFF: debug: ", s.debug)

		if s.debug >= 1 {
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

		if s.debug >= 1 {
			fmt.Println("Modbus server resp: ", resp)
		}

		respRtu, err := s.transport.Encode(s.id, resp)
		if err != nil {
			errorCallback(err)
			continue
		}

		if s.debug >= 9 {
			fmt.Println("Modbus server tx: ", HexDump(respRtu))
		}

		_, err = s.transport.Write(respRtu)
		if err != nil {
			errorCallback(err)
			continue
		}
	}
}
