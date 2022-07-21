package test

import (
	"bytes"
	"sync"
	"time"
)

// IoSim is used to simulate an io channel -- provides both sides so you can easily
// test code that uses an io.ReadWriter interface, etc
type IoSim struct {
	out  *bytes.Buffer
	in   *bytes.Buffer
	m    *sync.Mutex
	stop chan struct{}
}

// NewIoSim creates a new IO sim and returns the A and B side of an IO simulator
// that implements a ReadWriteCloser
func NewIoSim() (*IoSim, *IoSim) {
	var a2b bytes.Buffer
	var b2a bytes.Buffer
	var m sync.Mutex

	a := IoSim{&a2b, &b2a, &m, make(chan struct{})}
	b := IoSim{&b2a, &a2b, &m, make(chan struct{})}

	return &a, &b
}

func (ios *IoSim) Write(d []byte) (int, error) {
	ios.m.Lock()
	defer ios.m.Unlock()
	return ios.in.Write(d)
}

// Read blocks until there is some data in the out buffer or the ioSim is closed.
func (ios *IoSim) Read(d []byte) (int, error) {
	ret := make(chan struct{})

	go func() {
		for {
			ios.m.Lock()
			if ios.out.Len() > 0 {
				close(ret)
				ios.m.Unlock()
				return
			}
			ios.m.Unlock()
			select {
			case <-time.After(time.Millisecond):
				// continue
			case <-ios.stop:
				close(ret)
				return
			}
		}
	}()

	// block until we have data
	<-ret
	ios.m.Lock()
	defer ios.m.Unlock()
	return ios.out.Read(d)
}

// Close simulator
func (ios *IoSim) Close() error {
	close(ios.stop)
	return nil
}
