package needsported

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Modem is a typethat defines a modem
type Modem struct {
	port  io.ReadWriter
	debug bool
	apn   string
}

// NewModem creates a new modem type
//
// port should be a respreader
func NewModem(port io.ReadWriter, apn string, debug bool) *Modem {
	return &Modem{
		port:  port,
		debug: debug,
		apn:   apn,
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
	Mode           Mode
}

func (ms ModemSettings) String() string {
	return fmt.Sprintf("APN: %v\nCarrier Profile: %v\nTechnology: %v\nMode: %v",
		ms.APN, ms.CarrierProfile, ms.Technology, ms.Mode)
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
	var err error
	changed := false

	for try := 0; try < 3; try++ {
		err := m.SwitchCmdMode()
		if err != nil {
			continue
		}
		var settings ModemSettings
		settings, err = m.GetSettings()
		if err != nil {
			continue
		}

		if settings.APN != m.apn {
			log.Println("Modem: updating APN")
			changed = true
			err = m.SetAPN()
			if err != nil {
				continue
			}
		}

		if settings.CarrierProfile != CarrierProfileNone {
			log.Println("Modem: updating Carrier profile")
			changed = true
			err = m.SetCarrierProfile(CarrierProfileNone)
			if err != nil {
				continue
			}
		}

		if settings.Technology != TechnologyLTEM {
			log.Println("Modem: updating technology")
			changed = true
			err = m.SetTechnology(TechnologyLTEM)
			if err != nil {
				continue
			}
		}

		if settings.Mode != ModeTransparent {
			log.Println("Modem: updating mode to transparent")
			changed = true
			err = m.SetMode(ModeTransparent)
			if err != nil {
				continue
			}
		}

		if changed {
			err = m.Write()
			if err != nil {
				continue
			}

			err = m.Reset()
			if err != nil {
				continue
			}
		}

		// if we got this far, everything must have worked so
		// we're done
		break
	}

	return err
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

		if err != nil {
			continue
		}

		readString = readString[:n]

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

	db, err := strconv.ParseInt(resp, 16, 32)
	if err != nil {
		return
	}

	ret.Signal = int(db)

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

	resp, err = m.Cmd("ATAP")
	if err != nil {
		return
	}

	ap, err := strconv.Atoi(resp)

	if err != nil {
		return
	}

	ret.Mode = Mode(ap)

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
// assumes modem is in command mode
func (m *Modem) SetAPN() error {
	resp, err := m.Cmd("ATAN " + m.apn)
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	return nil
}

// Write stores changes to non-voltatile memory
func (m *Modem) Write() error {
	resp, err := m.Cmd("ATWR")
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	return nil
}

// Reset modem
func (m *Modem) Reset() error {
	resp, err := m.Cmd("ATFR")
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	return nil
}

// SetCarrierProfile updates the carrier profile in the modem
func (m *Modem) SetCarrierProfile(profile CarrierProfile) error {
	resp, err := m.Cmd("ATCP" + strconv.Itoa(int(profile)))
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	return nil
}

// SetTechnology updates the network technology setting in the modem
func (m *Modem) SetTechnology(tech Technology) error {
	resp, err := m.Cmd("ATN#" + strconv.Itoa(int(tech)))
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	return nil
}

// Mode defines operating mode
type Mode int

// define valid modes
const (
	ModeTransparent   Mode = 0
	ModeAPI                = 1
	ModeAPIWithEscape      = 2
	ModePython             = 4
	ModeBypass             = 5
)

func (m Mode) String() string {
	switch m {
	case ModeTransparent:
		return "transparent"
	case ModeAPI:
		return "API"
	case ModeAPIWithEscape:
		return "API with escape"
	case ModePython:
		return "Python"
	case ModeBypass:
		return "Bypass"
	default:
		return "unknown"

	}
}

// SetMode updates the operating mode
func (m *Modem) SetMode(mode Mode) error {
	resp, err := m.Cmd("ATAP" + strconv.Itoa(int(mode)))
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	return nil
}

// HTTPGet executes a http get command
func (m *Modem) HTTPGet(urlIn string) ([]byte, error) {
	err := m.SwitchCmdMode()
	if err != nil {
		return []byte{}, err
	}

	u, err := url.Parse(urlIn)

	if err != nil {
		return []byte{}, nil
	}

	resp, err := m.Cmd("ATDL " + u.Hostname())

	if err != nil {
		return []byte{}, err
	}

	if resp != "OK" {
		return []byte{}, errors.New("Expected OK to ATDL command")
	}

	resp, err = m.Cmd("ATDE 50")

	if err != nil {
		return []byte{}, err
	}

	if resp != "OK" {
		return []byte{}, errors.New("Expected OK to ATDE command")
	}

	resp, err = m.Cmd("ATIP 1")

	if err != nil {
		return []byte{}, err
	}

	if resp != "OK" {
		return []byte{}, errors.New("Expected OK to ATIP command")
	}

	resp, err = m.Cmd("ATCN")

	if err != nil {
		return []byte{}, err
	}

	if resp != "OK" {
		return []byte{}, errors.New("Expected OK to ATCN command")
	}

	r, err := http.NewRequest(http.MethodGet, urlIn, nil)
	if err != nil {
		return []byte{}, err
	}

	headbuf := bytes.NewBuffer([]byte{})
	r.Write(headbuf)
	fmt.Println(hex.Dump(headbuf.Bytes()))

	err = r.Write(m.port)
	if err != nil {
		return []byte{}, err
	}

	buf := make([]byte, 500)
	c, err := m.port.Read(buf)

	if err != nil {
		return []byte{}, err
	}

	buf = buf[0:c]

	return buf, nil
}
