package modbus

import (
	"errors"
	"fmt"
	"io"
	"log"
)

// Server defines a server (slave)
// Current Server only supports Modbus RTU,
// but could be expanded to do ASCII and TCP.
type Server struct {
	id     byte
	port   io.ReadWriter
	Regs   Regs
	chDone chan bool
}

// NewServer creates a new server instance
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewServer(id byte, port io.ReadWriter) *Server {
	return &Server{
		id:     id,
		port:   port,
		chDone: make(chan bool),
	}
}

// Close stops the listening channel
func (s *Server) Close() {
	s.chDone <- true
}

// Listen starts the server and listens for modbus requests
// this function does not return unless an error occurs
// The listen function supports various debug levels:
// 1 - dump packets
// 9 - dump raw data
func (s *Server) Listen(debug int, errorCallback func(error),
	changesCallback func([]RegChange)) {
	for {
		select {
		case <-s.chDone:
			log.Println("Exiting modbus server listen")
			return
		default:
		}
		buf := make([]byte, 200)
		cnt, err := s.port.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading serial port: ", err)
			}

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

		if packet[0] != s.id {
			// packet is not for this device
			continue
		}

		err = CheckRtuCrc(packet)
		if err != nil {
			errorCallback(errors.New("CRC error"))
			continue
		}

		req, err := RtuDecode(packet)
		if err != nil {
			errorCallback(err)
			continue
		}

		if debug >= 1 {
			fmt.Println("Modbus server req: ", req)
		}

		changes, resp, err := req.ProcessRequest(&s.Regs)
		if len(changes) > 0 {
			changesCallback(changes)
		}

		if err != nil {
			errorCallback(err)
			continue
		}

		if debug >= 1 {
			fmt.Println("Modbus server resp: ", resp)
		}

		respRtu, err := RtuEncode(s.id, resp)
		if err != nil {
			errorCallback(err)
			continue
		}

		if debug >= 9 {
			fmt.Println("Modbus server tx: ", HexDump(respRtu))
		}

		_, err = s.port.Write(respRtu)
		if err != nil {
			errorCallback(err)
			continue
		}
	}
}
