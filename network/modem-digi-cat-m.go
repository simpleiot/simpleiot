package network

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/simpleiot/simpleiot/respreader"
)

// Modem is a typethat defines a modem
type Modem struct {
	port  io.ReadWriter
	debug bool
}

// NewModem creates a new modem type
func NewModem(port io.ReadWriter, debug bool) *Modem {
	port = respreader.NewResponseReadWriter(port, 2*time.Second,
		50*time.Millisecond)

	return &Modem{
		port:  port,
		debug: debug,
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

// Cmd a command to modem and read response
func (m *Modem) Cmd(cmd string) (string, error) {
	readString := make([]byte, 100)

	_, err := m.port.Write([]byte(cmd + "\r"))
	if err != nil {
		return "", err
	}

	n, err := m.port.Read(readString)

	readString = readString[:n]

	if err != nil {
		return "", err
	}

	readStringS := strings.TrimSpace(string(readString))

	return readStringS, nil
}

// SwitchCmdMode switches the mode modem to command mode
func (m *Modem) SwitchCmdMode() error {
	readString := make([]byte, 100)

	_, err := m.port.Write([]byte("+++"))
	if err != nil {
		return err
	}

	n, err := m.port.Read(readString)

	readString = readString[:n]

	if err != nil {
		return err
	}

	readStringS := strings.TrimSpace(string(readString))

	if readStringS != "OK" {
		return errors.New("did not receive OK string")
	}

	return nil
}

// GetState is used to return modem state
func (m *Modem) GetState() (ret ModemState, err error) {
	err = m.SwitchCmdMode()
	if err != nil {
		return
	}

	var resp string

	resp, err = m.Cmd("ATAI")
	if err != nil {
		return
	}

	ret.Connected = resp == "0"

	return
}

// SetAPN is used to set the modem APN
func (m *Modem) SetAPN() error {
	return nil
}
