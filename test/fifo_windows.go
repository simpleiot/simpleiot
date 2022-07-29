//go:build windows

package test

// Fifo is a no-op to allow project to compile on windows
type Fifo struct {
}

// NewFifoA creates the A side interface. This must be called first to create the fifo files.
func NewFifoA(name string) (*Fifo, error) {
	ret := &Fifo{}
	return ret, nil
}

// NewFifoB creates the B side interface. This must be called after NewFifoB
func NewFifoB(name string) (*Fifo, error) {
	ret := &Fifo{}
	return ret, nil
}

func (f *Fifo) Read(b []byte) (int, error) {
	return 0, nil
}

func (f *Fifo) Write(b []byte) (int, error) {
	return 0, nil
}

// Close and delete fifos
func (f *Fifo) Close() error {
	return nil
}
