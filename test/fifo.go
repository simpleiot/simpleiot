//go:build !windows

package test

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
)

// Fifo uses unix named pipes or fifos to emulate a UART type channel
// the A side manages the channel, creates the fifos, cleans up, etc. The
// B side only opens the fifos for read/write. Fifo implements the io.ReadWriteCloser interface.
// The Close() on the B side does not do anything.
type Fifo struct {
	fread  io.ReadCloser
	fwrite io.WriteCloser
	a2b    string
	b2a    string
}

// NewFifoA creates the A side interface. This must be called first to create the fifo files.
func NewFifoA(name string) (*Fifo, error) {
	ret := &Fifo{}
	ret.a2b = name + "a2b"
	ret.b2a = name + "b2a"

	os.Remove(ret.a2b)
	os.Remove(ret.b2a)

	err := syscall.Mknod(ret.a2b, syscall.S_IFIFO|0666, 0)
	if err != nil {
		return nil, fmt.Errorf("mknod a2b failed: %v", err)
	}
	err = syscall.Mknod(ret.b2a, syscall.S_IFIFO|0666, 0)
	if err != nil {
		return nil, fmt.Errorf("mknod b2a failed: %v", err)
	}

	ret.fread, err = os.OpenFile(ret.b2a, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("error opening read file: %v", err)
	}

	ret.fwrite, err = os.OpenFile(ret.a2b, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("error opening write file: %v", err)
	}

	return ret, nil
}

// NewFifoB creates the B side interface. This must be called after NewFifoB
func NewFifoB(name string) (*Fifo, error) {
	ret := &Fifo{}

	a2b := name + "a2b"
	b2a := name + "b2a"

	var err error

	ret.fread, err = os.OpenFile(a2b, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("error opening read file: %v", err)
	}

	ret.fwrite, err = os.OpenFile(b2a, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("error opening write file: %v", err)
	}

	return ret, nil
}

func (f *Fifo) Read(b []byte) (int, error) {
	return f.fread.Read(b)
}

func (f *Fifo) Write(b []byte) (int, error) {
	return f.fwrite.Write(b)
}

// Close and delete fifos
func (f *Fifo) Close() error {
	if err := f.fwrite.Close(); err != nil {
		log.Println("Error closing write file")
	}

	if err := f.fread.Close(); err != nil {
		log.Println("Error closing read file")
	}

	if f.a2b != "" {
		os.Remove(f.a2b)
	}

	if f.b2a != "" {
		os.Remove(f.b2a)
	}

	return nil
}
