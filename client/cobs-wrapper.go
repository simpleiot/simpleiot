package client

import (
	"bytes"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/dim13/cobs"
	"github.com/simpleiot/simpleiot/test"
)

// CobsWrapper can be used to wrap an io.ReadWriteCloser to COBS encode/decode data
type CobsWrapper struct {
	dev              io.ReadWriteCloser
	readLeftover     bytes.Buffer
	readLock         sync.Mutex
	debug            int
	maxMessageLength int
}

// NewCobsWrapper creates a new cobs wrapper
func NewCobsWrapper(dev io.ReadWriteCloser, maxMessageLength int) *CobsWrapper {
	return &CobsWrapper{dev: dev, maxMessageLength: maxMessageLength}
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

// Read a COBS encoded data stream. The stream may optionally start with one or more NULL
// bytes and must end with a NULL byte. This Read blocks until we
// get an entire packet or an error.
func (cw *CobsWrapper) Read(b []byte) (int, error) {
	errCh := make(chan error)
	packetCh := make(chan []byte)

	// only let one Read read at a time
	go func() {
		cw.readLock.Lock()
		defer cw.readLock.Unlock()

		var decodeBuf bytes.Buffer
		foundNonZero := false
		ret := false

		// returns done if we hit error or decoded packet
		processByte := func(b byte) bool {
			if b == 0 {
				// throw away leading zeros or if we have
				// data, try to decode it
				if foundNonZero {
					decodeBuf.WriteByte(b)
					dec := cobs.Decode(decodeBuf.Bytes())
					if len(dec) > 0 {
						packetCh <- dec
						return true
					}
					// we did not decode a packet, return a decode error
					errCh <- ErrCobsDecodeError
					return true
				}
			} else {
				decodeBuf.WriteByte(b)
				foundNonZero = true
			}

			return false
		}

		// First, process any leftover data
		for cw.readLeftover.Len() > 0 {
			b, _ := cw.readLeftover.ReadByte()
			if processByte(b) {
				return
			}
		}

		for {
			// FIXME the +50 below is probably overkill
			buf := make([]byte, len(b)+50)
			c, err := cw.dev.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			buf = buf[0:c]

			if cw.debug >= 9 {
				log.Println("SER RX RAW: ", test.HexDump(buf))
			}

			for _, b := range buf {
				if !ret {
					ret = processByte(b)
				} else {
					cw.readLeftover.WriteByte(b)
				}
			}

			if ret {
				return
			}

			if decodeBuf.Len() > cw.maxMessageLength {
				errCh <- ErrCobsTooMuchData
				return
			}

			if cw.readLeftover.Len() > cw.maxMessageLength {
				errCh <- ErrCobsLeftoverBufferFull
				return
			}
		}
	}()

	select {
	case err := <-errCh:
		return 0, err
	case d := <-packetCh:
		return copy(b, d), nil
	}
}

func (cw *CobsWrapper) Write(b []byte) (int, error) {
	if cw.debug >= 9 {
		log.Println("SER TX RAW: ", test.HexDump(b))
	}

	return cw.dev.Write(append([]byte{0}, cobs.Encode(b)...))
}

// Close the device wrapped.
func (cw *CobsWrapper) Close() error {
	return cw.dev.Close()
}
