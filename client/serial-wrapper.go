package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/kjx98/crc16"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Packet format is:
//   - sequence #: 1 byte
//   - subject (16 bytes)
//   - protobuf (serial) payload
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

	pbPoints := make([]*pb.SerialPoint, len(points))
	for i, p := range points {
		pPb, err := p.ToSerial()
		if err != nil {
			return nil, err
		}
		pbPoints[i] = &pPb
	}

	pbSerial := &pb.SerialPoints{
		Points: pbPoints,
	}

	pbSerialBytes, err := proto.Marshal(pbSerial)
	if err != nil {
		return nil, err
	}

	ret.Write(pbSerialBytes)

	crc := crc16.ChecksumCCITT(ret.Bytes())

	err = binary.Write(&ret, binary.LittleEndian, crc)

	return ret.Bytes(), err
}

// SerialDecode can be used to decode serial data in a client.
// returns seq, subject, payload
func SerialDecode(d []byte) (byte, string, []byte, error) {
	l := len(d)

	if l < 1 {
		return 0, "", nil, errors.New("Not enough data")
	}

	if l < 3 {
		return d[0], "", nil, errors.New("Not enough data")
	}

	// check CRC

	crc := binary.LittleEndian.Uint16(d[l-2:])
	crcCalc := crc16.ChecksumCCITT(d[:l-2])
	if crc != crcCalc {
		return d[0], "", nil, errors.New("CRC check failed")
	}

	if l == 3 {
		return d[0], "", []byte{}, nil
	}

	if len(d) < 19 {
		return d[0], "", nil, errors.New("Not enough data")
	}

	// try to extract subject
	subject := string(bytes.Trim(d[1:17], "\x00"))

	// try to extract protobuf
	payload := d[17 : l-2]

	return d[0], subject, payload, nil
}
