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
	id   byte
	port io.ReadWriter
	Regs Regs
}

// NewServer creates a new server instance
// port must return an entire packet for each Read().
// github.com/simpleiot/simpleiot/respreader is a good
// way to do this.
func NewServer(id byte, port io.ReadWriter) *Server {
	return &Server{
		id:   id,
		port: port,
	}
}

// Listen starts the server and listens for modbus requests
// this function does not return unless and error occurs
// The listen function supports various debug levels:
// 1 - dump packets
// 9 - dump raw data
func (s *Server) Listen(debug int, errorCallback func(error),
	changesCallback func([]RegChange)) {
	for {
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
			fmt.Println("Modbus server raw data received: ", HexDump(packet))
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

		pdu, err := RtuDecode(packet)
		if err != nil {
			errorCallback(err)
			continue
		}

		changes, resp, err := pdu.ProcessRequest(&s.Regs)
		if len(changes) > 0 {
			changesCallback(changes)
		}

		if err != nil {
			errorCallback(err)
			continue
		}

		respRtu, err := RtuEncode(s.id, resp)
		if err != nil {
			errorCallback(err)
			continue
		}

		_, err = s.port.Write(respRtu)
		if err != nil {
			errorCallback(err)
			continue
		}
	}
}
