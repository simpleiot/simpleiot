package client

import (
	"bytes"
	"reflect"
	"sync"
	"testing"

	"github.com/dim13/cobs"
)

type ioSim struct {
	out *bytes.Buffer
	in  *bytes.Buffer
	m   *sync.Mutex
}

// newIoSim creates a new IO sim and returns the A and B side of an IO simulator
// that implements a ReadWriteCloser
func newIoSim() (*ioSim, *ioSim) {
	var a2b bytes.Buffer
	var b2a bytes.Buffer
	var m sync.Mutex

	a := ioSim{&a2b, &b2a, &m}
	b := ioSim{&b2a, &a2b, &m}

	return &a, &b
}

func (ios *ioSim) Write(d []byte) (int, error) {
	ios.m.Lock()
	defer ios.m.Unlock()
	return ios.in.Write(d)
}

func (ios *ioSim) Read(d []byte) (int, error) {
	ios.m.Lock()
	defer ios.m.Unlock()
	return ios.out.Read(d)
}

func (ios *ioSim) Close() error {
	return nil
}

func TestCobsWrapper(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := newIoSim()

	cw := newCobsWrapper(b)

	a.Write(cobs.Encode(d))

	buf := make([]byte, 500)

	c, err := cw.Read(buf)
	if err != nil {
		t.Fatal("Error reading cw: ", err)
	}
	buf = buf[0:c]

	if !reflect.DeepEqual(buf, d) {
		t.Fatal("Read data does not match")
	}

	_, err = cw.Write(d)
	if err != nil {
		t.Fatal("Error write: ", err)
	}

	buf = make([]byte, 500)

	c, err = a.Read(buf)
	if err != nil {
		t.Fatal("Error read: ", err)
	}
	buf = buf[0:c]

	if !reflect.DeepEqual(cobs.Decode(buf), d) {
		t.Fatal("cw.Write, buf is not same")
	}
}
