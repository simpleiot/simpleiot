package network

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// Ethernet implements the Interface interface
type Ethernet struct {
	iface string
}

// NewEthernet contructor
func NewEthernet(iface string) *Ethernet {
	return &Ethernet{
		iface: iface,
	}
}

// Desc returns a description of the interface
func (e *Ethernet) Desc() string {
	return fmt.Sprintf("Eth(%v)", e.iface)
}

// Connect network interface
func (e *Ethernet) Connect() error {
	// this is handled by system so no-op
	return nil
}

func (e *Ethernet) detected() bool {
	cnt, err := ioutil.ReadFile("/sys/class/net/" + e.iface + "/carrier")
	if err != nil {
		return false
	}

	if !strings.Contains(string(cnt), "1") {
		return false
	}

	cnt, err = ioutil.ReadFile("/sys/class/net/" + e.iface + "/operstate")
	if err != nil {
		return false
	}

	if !strings.Contains(string(cnt), "up") {
		return false
	}

	return true
}

// Connected returns true if connected
func (e *Ethernet) connected() bool {
	if !e.detected() {
		return false
	}

	_, err := GetIP(e.iface)
	if err == nil {
		return true
	}

	return false
}

// GetStatus returns ethernet interface status
func (e *Ethernet) GetStatus() (InterfaceStatus, error) {
	ip, _ := GetIP(e.iface)
	return InterfaceStatus{
		Detected:  e.detected(),
		Connected: e.connected(),
		IP:        ip,
	}, nil
}

// Reset interface. Currently no-op for ethernet
func (e *Ethernet) Reset() error {
	return nil
}
