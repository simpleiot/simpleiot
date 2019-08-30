package network

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Modem is a typethat defines a modem
type Modem struct {
	port  io.ReadWriter
	debug bool
}

// NewModem creates a new modem type
//
// port should be a respreader
func NewModem(port io.ReadWriter, debug bool) *Modem {
	return &Modem{
		port:  port,
		debug: debug,
	}
}

// ModemState describes the current state of the modem
type ModemState struct {
	Detected  bool
	Connected bool
	Operator  string
	Signal    int
}

func (ms ModemState) String() string {
	return fmt.Sprintf("Detected: %v\nConnected: %v\nOperator: %v\nSignal: %v",
		ms.Detected, ms.Connected, ms.Operator, ms.Signal)
}

// CarrierProfile is used to lock modem to a particular carrier
type CarrierProfile int

// define value carrier profiles
const (
	CarrierProfileAuto    CarrierProfile = 0
	CarrierProfileNone                   = 1
	CarrierProfileATT                    = 2
	CarrierProfileVerizon                = 3
)

func (cp CarrierProfile) String() string {
	switch cp {
	case CarrierProfileAuto:
		return "Autodetect"
	case CarrierProfileNone:
		return "No profile"
	case CarrierProfileATT:
		return "AT&T"
	case CarrierProfileVerizon:
		return "Verizon"
	default:
		return "unknown"
	}
}

// Technology is used to define Cat-M or NB-Iot operation
type Technology int

// define valid technologies
const (
	TechnologyLTEMWithNBIOTFallback Technology = 0
	TechnologyNBIOTWithLTEMFallback            = 1
	TechnologyLTEM                             = 2
	TechnologyNBIOT                            = 3
)

func (nt Technology) String() string {
	switch nt {
	case TechnologyLTEMWithNBIOTFallback:
		return "LTE-M with NB-IoT fallback"
	case TechnologyNBIOTWithLTEMFallback:
		return "NB-IoT with LTE-M fallback"
	case TechnologyLTEM:
		return "LTE-M only"
	case TechnologyNBIOT:
		return "NB-IoT only"
	default:
		return "Unknown"

	}
}

// ModemSettings describe the current modem settings
type ModemSettings struct {
	APN            string
	CarrierProfile CarrierProfile
	Technology     Technology
}

func (ms ModemSettings) String() string {
	return fmt.Sprintf("APN: %v\nCarrier Profile: %v\nTechnology: %v",
		ms.APN, ms.CarrierProfile, ms.Technology)
}

// ModemInfo describes information about the modem that is fairly static
type ModemInfo struct {
	ICCID     string
	IMEI      string
	FWVersion string
}

func (mi ModemInfo) String() string {
	return fmt.Sprintf("ICCID: %v\nIMEI: %v\nFWVersion: %v",
		mi.ICCID, mi.IMEI, mi.FWVersion)
}

// Configure is used to set up the modem
func (m *Modem) Configure() error {
	// try 3 times
	for try := 0; try < 3; try++ {
	}
	return nil
}

// Cmd a command to modem and read response
// retry 3 times
func (m *Modem) Cmd(cmd string) (string, error) {
	var err error
	for try := 0; try < 3; try++ {
		readString := make([]byte, 100)

		_, err = m.port.Write([]byte(cmd + "\r"))
		if err != nil {
			continue
		}

		var n int
		n, err = m.port.Read(readString)

		readString = readString[:n]

		if err != nil {
			continue
		}

		readStringS := strings.TrimSpace(string(readString))

		if m.debug {
			fmt.Printf("Modem: %v -> %v\n", cmd, readStringS)
		}

		return readStringS, nil
	}

	return "", err
}

// SwitchCmdMode switches the mode modem to command mode
// try 3 times
func (m *Modem) SwitchCmdMode() error {
	var err error
	for try := 0; try < 3; try++ {
		readString := make([]byte, 100)

		_, err = m.port.Write([]byte("+++"))
		if err != nil {
			continue
		}

		var n int
		n, err = m.port.Read(readString)

		readString = readString[:n]

		if err != nil {
			continue
		}

		readStringS := strings.TrimSpace(string(readString))

		if readStringS != "OK" {
			err = errors.New("did not receive OK string")
			continue
		}
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

	ret.Detected = true

	ret.Connected = resp == "0"

	ret.Operator, err = m.Cmd("ATMN")
	if err != nil {
		return
	}

	resp, err = m.Cmd("ATDB")
	if err != nil {
		return
	}

	db, err := strconv.Atoi(resp)
	if err != nil {
		return
	}

	ret.Signal = db

	return
}

// GetSettings are used to fetch the modem settings
func (m *Modem) GetSettings() (ret ModemSettings, err error) {
	err = m.SwitchCmdMode()
	if err != nil {
		return
	}

	var resp string

	ret.APN, err = m.Cmd("ATAN")
	if err != nil {
		return
	}

	resp, err = m.Cmd("ATCP")
	if err != nil {
		return
	}

	cp, err := strconv.Atoi(resp)

	if err != nil {
		return
	}

	ret.CarrierProfile = CarrierProfile(cp)

	resp, err = m.Cmd("ATN#")
	if err != nil {
		return
	}

	nt, err := strconv.Atoi(resp)

	if err != nil {
		return
	}

	ret.Technology = Technology(nt)

	return
}

// GetInfo is used to get static info from modem
func (m *Modem) GetInfo() (ret ModemInfo, err error) {
	err = m.SwitchCmdMode()
	if err != nil {
		return
	}

	ret.ICCID, err = m.Cmd("ATS#")
	if err != nil {
		return
	}

	ret.IMEI, err = m.Cmd("ATIM")
	if err != nil {
		return
	}

	ret.FWVersion, err = m.Cmd("ATMV")
	if err != nil {
		return
	}

	return

}

// SetAPN is used to set the modem APN
func (m *Modem) SetAPN() error {
	return nil
}
