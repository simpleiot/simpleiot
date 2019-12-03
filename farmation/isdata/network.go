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
	Interface  NetworkInterface
	DHCP       bool
	StaticIP   string
	SubnetMask string
	DefaultGw  string
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
