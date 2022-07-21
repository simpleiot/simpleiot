package client

import (
	"bytes"
	"errors"
	"io"

	"github.com/dim13/cobs"
)

type cobsWrapper struct {
	dev          io.ReadWriteCloser
	readLeftover bytes.Buffer
}

func newCobsWrapper(dev io.ReadWriteCloser) *cobsWrapper {
	return &cobsWrapper{dev: dev}
}

// Read a COBS encoded data stream. The stream must start and end with a NULL byte.
// We don't attempt to decode until we see that pattern. This Read blocks until we
// get an entire packet or an error.
func (cw *cobsWrapper) Read(b []byte) (int, error) {
	errCh := make(chan error)
	packetCh := make(chan []byte)

	go func() {
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
					errCh <- errors.New("COBS decode error")
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
			// FIXME the +50 below is probably not great
			buf := make([]byte, len(b)+50)
			c, err := cw.dev.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			buf = buf[0:c]

			for _, b := range buf {
				if !ret {
					ret = processByte(b)
				} else {
					cw.readLeftover.WriteByte(b)
				}
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

func (cw *cobsWrapper) Write(b []byte) (int, error) {
	return cw.dev.Write(append([]byte{0}, cobs.Encode(b)...))
}

func (cw *cobsWrapper) Close() error {
	return cw.dev.Close()
}