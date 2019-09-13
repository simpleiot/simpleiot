package network

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cbrake/go-serial/serial"
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

// cmd send a command to modem and read response
// retry 3 times
func (m *Modem) cmd(cmd string) (string, error) {
	if err := m.openCmdPort(); err != nil {
		return "", err
	}

	var err error
	for try := 0; try < 3; try++ {
		readString := make([]byte, 100)

		_, err = m.atCmdPort.Write([]byte(cmd + "\r"))
		if err != nil {
			continue
		}

		var n int
		n, err = m.atCmdPort.Read(readString)

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

// service, rssi, rsrp, sinr, rsrq
// +QCSQ: "CAT-M1",-52,-81,195,-10
var reQcsq = regexp.MustCompile(`\+QCSQ:\s*"(.+)",(-*\d+),(-*\d+),(\d+),(-*\d+)`)

func (m *Modem) qcsq() (service bool, rssi, rsrp, rsrq int, err error) {
	var resp string
	resp, err = m.cmd("AT+QCSQ")
	if err != nil {
		return
	}

	found := false

	for _, line := range strings.Split(string(resp), "\n") {

		matches := reQcsq.FindStringSubmatch(line)

		if len(matches) < 6 {
			continue
		}

		found = true

		serviceS := matches[1]
		rssi, _ = strconv.Atoi(matches[2])
		rsrq, _ = strconv.Atoi(matches[3])
		rsrp, _ = strconv.Atoi(matches[5])

		service = serviceS == "CAT-M1"
	}

	if !found {
		err = fmt.Errorf("Error parsing QCSQ response: %v", resp)
	}

	return
}

// service, rssi, rsrp, sinr, rsrq
// +QSPN: "CHN-UNICOM","UNICOM","",0,"46001"
// +QSPN: "Verizon Wireless","VzW","Hologram",0,"311480"

var reQspn = regexp.MustCompile(`\+QSPN:\s*"(.*)","(.*)","(.*)",(\d+),"(.*)"`)

func (m *Modem) qspn() (network string, err error) {
	var resp string
	resp, err = m.cmd("AT+QSPN")
	if err != nil {
		return
	}

	found := false

	for _, line := range strings.Split(string(resp), "\n") {

		matches := reQspn.FindStringSubmatch(line)

		if len(matches) < 6 {
			continue
		}

		found = true

		network = matches[1]
	}

	if !found {
		err = fmt.Errorf("Error parsing QSPN response: %v", resp)
	}

	return
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
	fmt.Println("Modem: starting PPP")
	service, _, _, _, err := m.qcsq()
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
	var retError error
	ip, _ := GetIP(m.iface)

	service, rssi, rsrp, rsrq, err := m.qcsq()
	if err != nil {
		retError = err
	}

	network, err := m.qspn()
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
