package client

import (
	"bytes"
	"errors"
	"io"
	"log"

	"github.com/dim13/cobs"
	"github.com/simpleiot/simpleiot/test"
)

// CobsWrapper can be used to wrap an io.ReadWriteCloser to COBS encode/decode data
type CobsWrapper struct {
	dev              io.ReadWriteCloser
	readLeftover     bytes.Buffer
	debug            int
	maxMessageLength int
}

// NewCobsWrapper creates a new cobs wrapper
func NewCobsWrapper(dev io.ReadWriteCloser, maxMessageLength int) *CobsWrapper {
	ret := CobsWrapper{dev: dev, maxMessageLength: maxMessageLength}
	// grow buffer to minimize allocations
	ret.readLeftover.Grow(maxMessageLength)
	return &ret
}

// ErrCobsDecodeError indicates we got an error decoding a COBS packet
var ErrCobsDecodeError = errors.New("COBS decode error")

// ErrCobsTooMuchData indicates we received too much data without a null in it
// to delineate packets
var ErrCobsTooMuchData = errors.New("COBS decode: too much data without null")

// ErrCobsLeftoverBufferFull indicates our leftover buffer is too full to process
var ErrCobsLeftoverBufferFull = errors.New("COBS leftover buffer too full")

// SetDebug sets the debug level. If >= 9, then it dumps the raw data
// received.
func (cw *CobsWrapper) SetDebug(debug int) {
	cw.debug = debug
}

// Decode a null-terminated cobs frame to a slice of bytes
// data is shifted down to start at beginning of buffer
func cobsDecodeInplace(b []byte) (int, error) {
	if b == nil || len(b) <= 2 {
		return 0, errors.New("Not enough data for cobs decode")
	}

	// used to skip leading zeros
	foundStart := false

	// define input and output indicies
	var iIn, iOut int
	var off, iOff uint8

	for iIn = 0; iIn < len(b); iIn++ {
		bCur := b[iIn]

		if !foundStart {
			if bCur == 0 {
				continue
			}

			foundStart = true
			off = bCur
			iOff = 0
			continue
		}

		iOff++

		if iOff == off {
			if bCur == 0 {
				// we've reached the end folks
				return iOut, nil
			}

			if off != 0xff {
				b[iOut] = 0
				iOut++
			}
			off = bCur
			iOff = 0
		} else {
			if bCur == 0 {
				return 0, ErrCobsDecodeError
			}
			b[iOut] = bCur
			iOut++
		}
	}

	return iOut, nil
}

// Read a COBS encoded data stream. The stream may optionally start with one or more NULL
// bytes and must end with a NULL byte. This Read blocks until we
// get an entire packet or an error. b must be large enough to hold the entire packet.
func (cw *CobsWrapper) Read(b []byte) (int, error) {
	// we read data until we see a zero or hit the size of the b buffer
	// current location in read buffer
	var cur int

	// first, process any leftover bytes looking for packets
	if cw.readLeftover.Len() > 0 {
		foundStart := false

		lb := cw.readLeftover.Bytes()
		for i := 0; i < len(lb); i++ {
			if !foundStart {
				if lb[i] == 0 {
					continue
				}
				foundStart = true
			}
			if lb[i] == 0 {
				// found end of packet, copy to read buffer and process
				_, _ = cw.readLeftover.Read(b[0:i])
				return cobsDecodeInplace(b[0:i])
			}
		}

		// write leftover bytes to beginning of buffer
		bBuf := bytes.NewBuffer(b)
		c, _ := bBuf.Write(cw.readLeftover.Bytes())

		cur += c
	}

	foundStart := false

	for {
		c, err := cw.dev.Read(b[cur:])
		if err != nil {
			return 0, err
		}

		if c > 0 {
			// look for zero in buffer
			for i := 0; i < c; i++ {
				if !foundStart {
					if b[cur+i] == 0 {
						continue
					}
					foundStart = true
				}
				if b[cur+i] == 0 {
					// found end of packet, decode in place
					// first save off extra bytes
					cw.readLeftover.Write(b[cur+i+1 : cur+c])

					return cobsDecodeInplace(b[0 : cur+i+1])
				}
			}
		}

		cur += c

		if cur >= len(b) || cur > cw.maxMessageLength {
			return 0, ErrCobsTooMuchData
		}
	}
}

func (cw *CobsWrapper) Write(b []byte) (int, error) {
	if cw.debug >= 8 {
		log.Println("SER TX RAW: ", test.HexDump(b))
	}

	w := append([]byte{0}, cobs.Encode(b)...)

	if cw.debug >= 9 {
		log.Println("SER TX COBS: ", test.HexDump(w))
	}

	return cw.dev.Write(w)
}

// Close the device wrapped.
func (cw *CobsWrapper) Close() error {
	return cw.dev.Close()
}
