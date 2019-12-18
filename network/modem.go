package network

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/simpleiot/simpleiot/file"
	"github.com/simpleiot/simpleiot/respreader"
)

// Modem is an interface that always reports detected/connected
type Modem struct {
	iface         string
	chatScript    string
	reset         func() error
	atCmdPortName string
	atCmdPort     io.ReadWriteCloser
	debug         bool
	lastPPPRun    time.Time
}

// NewModem constructor
func NewModem(chatScript string, atCmdPortName string, reset func() error, debug bool) *Modem {
	ret := &Modem{
		iface:         "ppp0",
		chatScript:    chatScript,
		reset:         reset,
		atCmdPortName: atCmdPortName,
		debug:         debug,
	}

	return ret
}

func (m *Modem) openCmdPort() error {
	if m.atCmdPort != nil {
		return nil
	}

	options := serial.OpenOptions{
		PortName:          m.atCmdPortName,
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

// Connect stub
func (m *Modem) Connect() error {
	if err := m.openCmdPort(); err != nil {
		return err
	}

	fmt.Println("Modem: starting PPP")
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

	return exec.Command("pon", m.chatScript).Run()
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

	network, err := CmdQspn(m.atCmdPort)
	if err != nil {
		retError = err
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
	return m.reset()
}
