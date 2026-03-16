package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/kjx98/crc16"
	"github.com/simpleiot/simpleiot/data"
)

// Packet format is:
//   - sequence #: 1 byte
//   - subject (16 bytes)
//   - binary encoded points payload
//   - crc: 2 bytes (CCITT)

// SerialEncode can be used in a client to encode points sent over a serial link.
func SerialEncode(seq byte, subject string, points data.Points) ([]byte, error) {
	var ret bytes.Buffer
	ret.WriteByte(seq)

	sub := make([]byte, 16)

	if len(subject) > 16 {
		return []byte{},
			fmt.Errorf("SerialEncode Error: length of subject %v is longer than 20 bytes", subject)
	}

	copy(sub, []byte(subject))

	_, err := ret.Write(sub)
	if err != nil {
		return []byte{}, fmt.Errorf("SerialEncode: error writing to buffer: %v", err)
	}

	ret.Write(points.Encode())

	crc := crc16.ChecksumCCITT(ret.Bytes())

	err = binary.Write(&ret, binary.LittleEndian, crc)

	return ret.Bytes(), err
}

// SerialDecode can be used to decode serial data in a client.
// returns seq, subject, payload
func SerialDecode(d []byte) (byte, string, []byte, error) {
	l := len(d)

	if l < 1 {
		return 0, "", nil, errors.New("not enough data")
	}

	if l < (1 + 16) {
		return d[0], "", nil, errors.New("not enough data")
	}

	// try to extract subject
	subject := string(bytes.Trim(d[1:17], "\x00"))
	end := l

	if subject != "log" {
		// check CRC
		end -= 2
		if len(d) < (1 + 2 + 16) {
			return d[0], "", nil, errors.New("not enough data")
		}

		crc := binary.LittleEndian.Uint16(d[l-2:])
		crcCalc := crc16.ChecksumCCITT(d[:l-2])
		if crc != crcCalc {
			return d[0], "", nil, errors.New("CRC check failed")
		}
	}

	// extract payload
	payload := d[17:end]

	return d[0], subject, payload, nil
}
