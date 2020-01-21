package network

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/simpleiot/simpleiot/file"
	"github.com/simpleiot/simpleiot/respreader"
)

const apnVerizon = "vzwinternet"
const apnHologram = "hologram"

// Modem is an interface that always reports detected/connected
type Modem struct {
	iface      string
	atCmdPort  io.ReadWriteCloser
	lastPPPRun time.Time
	config     ModemConfig
}

// ModemConfig describes the configuration for a modem
type ModemConfig struct {
	ChatScript    string
	AtCmdPortName string
	Reset         func() error
	Debug         bool
	APN           string
}

// NewModem constructor
func NewModem(config ModemConfig) *Modem {
	ret := &Modem{
		iface:  "ppp0",
		config: config,
	}

	DebugAtCommands = config.Debug

	return ret
}

func (m *Modem) openCmdPort() error {
	if m.atCmdPort != nil {
		return nil
	}

	options := serial.OpenOptions{
		PortName:          m.config.AtCmdPortName,
		BaudRate:          115200,
		DataBits:          8,
		StopBits:          1,
		MinimumReadSize:   1,
		RTSCTSFlowControl: true,
	}

	port, err := serial.Open(options)

	if err != nil {
		return err
	}

	m.atCmdPort = respreader.NewResponseReadWriteCloser(port, 10*time.Second,
		50*time.Millisecond)

	return nil
}

// Desc returns description
func (m *Modem) Desc() string {
	return "modem"
}

// detected returns true if modem detected
func (m *Modem) detected() bool {
	return file.Exists("/dev/ttyUSB2") && file.Exists("/dev/ttyUSB3")
}

func (m *Modem) pppActive() bool {
	if !m.detected() {
		return false
	}

	_, err := GetIP(m.iface)
	if err == nil {
		return true
	}

	return false
}

// Configure modem interface
func (m *Modem) Configure() (InterfaceConfig, error) {
	ret := InterfaceConfig{
		Apn: m.config.APN,
	}

	// current sets APN and configures for internal SIM
	if err := m.openCmdPort(); err != nil {
		return ret, err
	}

	// disable echo as it messes up the respreader in that it
	// echos the command, which is not part of the response

	err := CmdOK(m.atCmdPort, "ATE0")
	if err != nil {
		return ret, err
	}

	err = CmdSetApn(m.atCmdPort, m.config.APN)
	if err != nil {
		return ret, err
	}

	mode, err := CmdBg96GetScanMode(m.atCmdPort)
	fmt.Println("BG96 scan mode: ", mode)
	if err != nil {
		return ret, fmt.Errorf("Error getting scan mode: %v", err.Error())
	}

	if mode != BG96ScanModeLTE {
		fmt.Println("Setting BG96 scan mode ...")
		err := CmdBg96ForceLTE(m.atCmdPort)
		if err != nil {
			return ret, fmt.Errorf("Error setting scan mode: %v", err.Error())
		}
	}

	err = CmdFunMin(m.atCmdPort)
	if err != nil {
		return ret, fmt.Errorf("Error setting fun Min: %v", err.Error())
	}

	err = CmdOK(m.atCmdPort, "AT+QCFG=\"gpio\",1,26,1,0,0,1")
	if err != nil {
		return ret, fmt.Errorf("Error setting GPIO: %v", err.Error())
	}

	if m.config.APN == apnVerizon {
		err = CmdOK(m.atCmdPort, "AT+QCFG=\"gpio\",3,26,1,1")
		if err != nil {
			return ret, fmt.Errorf("Error setting GPIO: %v", err.Error())
		}

	} else {
		err = CmdOK(m.atCmdPort, "AT+QCFG=\"gpio\",3,26,0,1")
		if err != nil {
			return ret, fmt.Errorf("Error setting GPIO: %v", err.Error())
		}

	}

	err = CmdFunFull(m.atCmdPort)
	if err != nil {
		return ret, fmt.Errorf("Error setting fun full: %v", err.Error())
	}

	// enable GPS
	/* for some reason this is failing -- likely a timing issue
	err = CmdOK(m.atCmdPort, "AT+QGPS=1")
	if err != nil {
		return fmt.Errorf("Error enabling GPS: %v", err.Error())
	}

	err = CmdOK(m.atCmdPort, "AT+QGPSCFG=\"nmeasrc\",1")
	if err != nil {
		return fmt.Errorf("Error settings GPS source: %v", err.Error())
	}
	*/

	sim, err := CmdGetSimBg96(m.atCmdPort)

	if err != nil {
		return ret, fmt.Errorf("Error getting SIM #: %v", err.Error())
	}

	ret.Sim = sim

	imei, err := CmdGetImei(m.atCmdPort)

	if err != nil {
		return ret, fmt.Errorf("Error getting IMEI #: %v", err.Error())
	}

	ret.Imei = imei

	version, err := CmdGetFwVersionBG96(m.atCmdPort)

	if err != nil {
		return ret, fmt.Errorf("Error getting fw version #: %v", err.Error())
	}

	ret.Version = version

	return ret, nil
}

// Connect stub
func (m *Modem) Connect() error {
	if err := m.openCmdPort(); err != nil {
		return err
	}

	mode, err := CmdBg96GetScanMode(m.atCmdPort)

	if err != nil {
		return err
	}

	log.Println("BG96 scan mode: ", mode)

	if mode != BG96ScanModeLTE {
		log.Println("Setting BG96 scan mode")
		err := CmdBg96ForceLTE(m.atCmdPort)
		if err != nil {
			return err
		}
	}

	service, _, _, _, err := CmdQcsq(m.atCmdPort)
	if err != nil {
		return err
	}

	// TODO need to set APN, etc before we do this
	// but eventually want to make sure we have service
	// before running PPP
	if !service {

	}

	if time.Since(m.lastPPPRun) < 30*time.Second {
		return errors.New("only run PPP once every 30s")
	}

	m.lastPPPRun = time.Now()

	log.Println("Modem: starting PPP")
	return exec.Command("pon", m.config.ChatScript).Run()
}

// GetStatus return interface status
func (m *Modem) GetStatus() (InterfaceStatus, error) {
	if err := m.openCmdPort(); err != nil {
		return InterfaceStatus{}, err
	}

	var retError error
	ip, _ := GetIP(m.iface)

	service, rssi, rsrp, rsrq, err := CmdQcsq(m.atCmdPort)
	if err != nil {
		retError = err
	}

	var network string

	if service {
		network, err = CmdCops(m.atCmdPort)
		if err != nil {
			retError = err
		}
	}

	return InterfaceStatus{
		Detected:  m.detected(),
		Connected: m.pppActive() && service,
		Operator:  network,
		IP:        ip,
		Signal:    rssi,
		Rsrp:      rsrp,
		Rsrq:      rsrq,
	}, retError
}

// Reset stub
func (m *Modem) Reset() error {
	if m.atCmdPort != nil {
		m.atCmdPort.Close()
		m.atCmdPort = nil
	}

	exec.Command("poff").Run()
	return m.config.Reset()
}
