package modbus

import (
	"fmt"
	"io"
)

// IoSimPort one end of a IoSim
type IoSimPort struct {
	name  string
	tx    chan byte
	rx    chan byte
	debug bool
}

// NewIoSimPort returns a new port of an IoSim
func NewIoSimPort(name string, tx chan byte, rx chan byte, debug bool) *IoSimPort {
	return &IoSimPort{
		name:  name,
		tx:    tx,
		rx:    rx,
		debug: debug,
	}
}

// Read reads data from IoSimPort
func (isp *IoSimPort) Read(data []byte) (int, error) {
	data[0] = <-isp.rx
	if isp.debug {
		packet := data[:1]
		fmt.Printf("%v Read (%v): %v\n", isp.name, 1, HexDump(packet))
	}
	return 1, nil
}

// Write reads data from IoSimPort
func (isp *IoSimPort) Write(data []byte) (int, error) {
	for _, b := range data {
		isp.tx <- b
	}
	c := len(data)
	if isp.debug {
		packet := data[:c]
		fmt.Printf("%v Write: %v\n", isp.name, HexDump(packet))
	}

	return c, nil
}

// IoSim simulates a serial port and provides a io.ReadWriter
// for both ends
type IoSim struct {
	aToB  chan byte
	bToA  chan byte
	debug bool
}

// NewIoSim creates a new IO simulator
func NewIoSim(debug bool) *IoSim {
	return &IoSim{
		aToB:  make(chan byte, 500),
		bToA:  make(chan byte, 500),
		debug: debug,
	}
}

// GetA returns the A port from a IoSim
func (is *IoSim) GetA() io.ReadWriter {
	return NewIoSimPort("A", is.aToB, is.bToA, is.debug)
}

// GetB returns the B port from a IoSim
func (is *IoSim) GetB() io.ReadWriter {
	return NewIoSimPort("B", is.bToA, is.aToB, is.debug)
}
