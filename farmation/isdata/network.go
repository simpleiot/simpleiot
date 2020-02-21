package isdata

import "github.com/simpleiot/simpleiot/network"

// NetworkInterface defines the current network interface
type NetworkInterface int

// define valid network interfaces
const (
	NetworkInterfaceEthernet NetworkInterface = iota
	NetworkInterfaceCellular
)

// NetworkConfig defines the current network configuration
type NetworkConfig struct {
	Interface  NetworkInterface `json:"interface"`
	DHCP       bool             `json:"dhcp"`
	StaticIP   string           `json:"staticIP"`
	SubnetMask string           `json:"subnetMask"`
	DefaultGw  string           `json:"defaultGw"`
}

// NetworkState defines the current network state
type NetworkState struct {
	//EthernetDetected bool
	//ModemDetected    bool
	//Connected        bool
	//ModemRSRQ        float64
	Description     string
	InterfaceStatus network.InterfaceStatus
	ErrorCnt        int
}
