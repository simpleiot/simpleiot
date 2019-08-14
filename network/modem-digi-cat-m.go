package network

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/svent/go-nbreader"
)

const (
	commandMode = "+++"
)

// Modem is a typethat defines a modem
type Modem struct {
	portNb io.Reader
	port   io.ReadWriter
}

// NewModem creates a new modem type
func NewModem(port io.ReadWriter) *Modem {
	portNb := nbreader.NewNBReader(port, 100,
		nbreader.Timeout(time.Second),
		nbreader.ChunkTimeout(time.Millisecond*50))

	return &Modem{
		portNb: portNb,
		port:   port,
	}
}

// ModemState describes the current state of the modem
type ModemState struct {
	Detected  bool
	Connected bool
	APN       string
}

// ModemSettings describe the current modem settings
type ModemSettings struct {
	APN               string
	CarrierProfile    int
	NetworkTechnology int
}

// GetSettings are used to fetch the modem settings
func (m *Modem) GetSettings() (ret ModemSettings, err error) {

	return
}

// Connect is used to set up and connect the modem
func (m *Modem) Connect() error {
	return nil
}

// Flush is used to clear any data out of the serial port buffer before
// executing a command.
func (m *Modem) Flush() error {
	// flush any data
	readString := make([]byte, 100)
	n, err := m.portNb.Read(readString)
	readString = readString[:n]
	if n > 0 {
		fmt.Print("modem flush: ", hex.Dump(readString))
	}
	return err
}

// Send a command to modem and read response with 200ms max delay
func (m *Modem) Send(cmd string) (string, error) {
	return m.SendDelay(cmd, 200*time.Millisecond)
}

// SendDelay a command to modem and read response
func (m *Modem) SendDelay(cmd string, delay time.Duration) (string, error) {
	readString := make([]byte, 100)

	_, err := m.port.Write([]byte(cmd + "\r"))
	if err != nil {
		return "", err
	}

	time.Sleep(delay)
	n, err := m.portNb.Read(readString)

	readString = readString[:n]
	//fmt.Print("Digi read: ", hex.Dump(readString))

	if err != nil {
		return "", err
	}

	readStringS := strings.TrimSpace(string(readString))

	return readStringS, nil
}

// SwitchCmdMode switches the mode modem to command mode

// GetState is used to return modem state
func (m *Modem) GetState() (ret ModemState, err error) {
	_, err = m.port.Write([]byte(commandMode))
	if err != nil {
		return
	}

	err = m.Flush()

	if err != nil {
		return
	}

	return
}

// SetAPN is used to set the modem APN
func (m *Modem) SetAPN() error {
	return nil
}
