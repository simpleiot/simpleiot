package network

import (
	"fmt"
	"os/exec"

	"github.com/simpleiot/simpleiot/file"
)

// Modem is an interface that always reports detected/connected
type Modem struct {
	iface      string
	chatScript string
}

// NewModem constructor
func NewModem(chatScript string) *Modem {
	return &Modem{
		iface:      "ppp0",
		chatScript: chatScript,
	}
}

// Desc returns description
func (m *Modem) Desc() string {
	return "modem"
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

	ip, err := GetIP(m.iface)
	fmt.Println("CLIFF: modem connected, ip: ", ip)
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
	return nil
}
