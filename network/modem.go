package network

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/simpleiot/simpleiot/file"
)

// Modem is an interface that always reports detected/connected
type Modem struct {
	iface      string
	chatScript string
	reset      func() error
	atCmdPort  io.ReadWriter
	debug      bool
}

// NewModem constructor
func NewModem(chatScript string, reset func() error) *Modem {
	return &Modem{
		iface:      "ppp0",
		chatScript: chatScript,
		reset:      reset,
	}
}

// Desc returns description
func (m *Modem) Desc() string {
	return "modem"
}

// cmd send a command to modem and read response
// retry 3 times
func (m *Modem) cmd(cmd string) (string, error) {
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
var reQcsq = regexp.MustCompile(`\+QCSQ:\s*\"(.+)\",(\d+),(\d+),(\d+),(\d+)`)

func (m *Modem) qcsq() (service bool, rssi, rsrp, rsrq int, err error) {
	var resp string
	resp, err = m.cmd("AT+QCSQ")
	if err != nil {
		return
	}

	matches := reQcsq.FindStringSubmatch(resp)

	if len(matches) < 6 {
		err = errors.New("Error parsing cmd response")
		return
	}

	serviceS := matches[1]
	rssi, _ = strconv.Atoi(matches[2])
	rsrq, _ = strconv.Atoi(matches[4])
	rsrp, _ = strconv.Atoi(matches[5])

	service = serviceS == "CAT-M1"

	return
}

// detected returns true if modem detected
func (m *Modem) detected() bool {
	return file.Exists("/dev/ttyUSB2") && file.Exists("/dev/ttyUSB3")
}

// Connected returns true if connected
func (m *Modem) connected() bool {
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
	return exec.Command("pon", m.chatScript).Run()
}

// GetStatus return interface status
func (m *Modem) GetStatus() (InterfaceStatus, error) {
	ip, _ := GetIP(m.iface)
	return InterfaceStatus{
		Detected:  m.detected(),
		Connected: m.connected(),
		IP:        ip,
	}, nil
}

// Reset stub
func (m *Modem) Reset() error {
	exec.Command("poff").Run()
	return m.reset()
}
