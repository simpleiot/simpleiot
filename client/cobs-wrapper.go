package client

import (
	"io"

	"github.com/dim13/cobs"
)

type cobsWrapper struct {
	dev io.ReadWriteCloser
}

func newCobsWrapper(dev io.ReadWriteCloser) *cobsWrapper {
	return &cobsWrapper{dev: dev}
}

func (cw *cobsWrapper) Read(b []byte) (int, error) {
	// FIXME the +50 below is probably not great
	buf := make([]byte, len(b)+50)
	c, err := cw.dev.Read(buf)
	if err != nil {
		return 0, err
	}
	buf = buf[0:c]
	dec := cobs.Decode(buf)
	return copy(b, dec), nil
}

func (cw *cobsWrapper) Write(b []byte) (int, error) {
	return cw.dev.Write(cobs.Encode(b))
}

func (cw *cobsWrapper) Close() error {
	return cw.dev.Close()
}
