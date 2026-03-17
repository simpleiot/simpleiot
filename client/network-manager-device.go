//go:build linux

package client

import (
	"encoding/json"

	nm "github.com/Wifx/gonetworkmanager/v2"
)

// NetworkManagerDevice is a device managed by NetworkManager
type NetworkManagerDevice struct {
	ID         string `node:"id"`
	Parent     string `node:"parent"`
	Path       string `point:"path"`
	Interface  string `point:"interface"`
	State      string `point:"state"`
	DeviceType string `point:"deviceType"`
	// ActiveConnectionID
	IPv4Addresses   []IPv4Address `point:"ipv4Addresses"`
	IPv4Netmasks    []IPv4Netmask `point:"ipv4Netmasks"`
	IPv4Gateway     IPv4Address   `point:"ipv4Gateway"`
	IPv4Nameservers []IPv4Address `point:"ipv4Nameservers"`
	IPv6Addresses   []IPv6Address `point:"ipv6Addresses"`
	IPv6Prefixes    []uint8       `point:"ipv6Prefixes"`
	IPv6Gateway     IPv6Address   `point:"ipv6Gateway"`
	IPv6Nameservers []IPv6Address `point:"ipv6Nameservers"`
	HardwareAddress string        `point:"hardwareAddress"`
	Managed         bool
	// Wi-Fi specific properties
	ActiveAccessPoint *AccessPoint `point:"activeAccessPoint"`
	AccessPoints      []string     `point:"accessPoints"` // JSON-encoded strings
}

// ResolveDevice returns a NetworkManagerDevice from a NetworkManager Device
func ResolveDevice(parent string, device nm.Device) (
	dev NetworkManagerDevice, err error,
) {
	dev.Parent = parent

	// Read device info via D-Bus
	dev.Path = string(device.GetPath())
	dev.ID = dev.Path

	dev.Interface, err = device.GetPropertyInterface()
	if err != nil {
		return dev, err
	}
	if dev.Interface != "" {
		dev.ID = dev.Interface
	}

	ipIface, err := device.GetPropertyIpInterface()
	if err != nil {
		return dev, err
	}
	if ipIface != "" {
		dev.ID = ipIface
	}

	state, err := device.GetPropertyState()
	if err != nil {
		return dev, err
	}
	dev.State = state.String()

	// Populate IPv4 state
	ipv4, err := device.GetPropertyIP4Config()
	if err != nil {
		return dev, err
	}
	if ipv4 != nil {
		ip4Addresses, err := ipv4.GetPropertyAddressData()
		if err != nil {
			return dev, err
		}
		for _, addr := range ip4Addresses {
			dev.IPv4Addresses = append(dev.IPv4Addresses,
				IPv4Address(addr.Address),
			)
			dev.IPv4Netmasks = append(dev.IPv4Netmasks,
				IPv4NetmaskPrefix(addr.Prefix),
			)
		}

		gateway, err := ipv4.GetPropertyGateway()
		if err != nil {
			return dev, err
		}
		dev.IPv4Gateway = IPv4Address(gateway)

		nameservers, err := ipv4.GetPropertyNameserverData()
		if err != nil {
			return dev, err
		}
		for _, addr := range nameservers {
			if addr.Address != "" {
				dev.IPv4Nameservers = append(dev.IPv4Nameservers,
					IPv4Address(addr.Address),
				)
			}
		}
	}

	ipv6, err := device.GetPropertyIP6Config()
	if err != nil {
		return dev, err
	}
	if ipv6 != nil {
		// Populate IPv6 state
		ip6Addresses, err := ipv6.GetPropertyAddressData()
		if err != nil {
			return dev, err
		}
		for _, addr := range ip6Addresses {
			dev.IPv6Addresses = append(dev.IPv6Addresses, IPv6Address(addr.Address))
			dev.IPv6Prefixes = append(dev.IPv6Prefixes, addr.Prefix)
		}

		gateway, err := ipv6.GetPropertyGateway()
		if err != nil {
			return dev, err
		}
		dev.IPv6Gateway = IPv6Address(gateway)

		nameservers, err := ipv6.GetPropertyNameservers()
		if err != nil {
			return dev, err
		}
		for _, addr := range nameservers {
			dev.IPv6Nameservers = append(dev.IPv6Nameservers,
				NewIPv6Address(addr),
			)
		}
	}

	dev.Managed, err = device.GetPropertyManaged()
	if err != nil {
		return dev, err
	}

	deviceType, err := device.GetPropertyDeviceType()
	if err != nil {
		return dev, err
	}
	dev.DeviceType = deviceType.String()

	if devHwAddr, ok := device.(interface {
		GetPropertyHwAddress() (string, error)
	}); ok {
		dev.HardwareAddress, err = devHwAddr.GetPropertyHwAddress()
		if err != nil {
			return dev, err
		}
		if dev.HardwareAddress != "" && dev.HardwareAddress != "00:00:00:00:00:00" {
			dev.ID = dev.HardwareAddress
		}
	}

	// Add WiFi access point
	if devAP, ok := device.(interface {
		GetPropertyActiveAccessPoint() (nm.AccessPoint, error)
	}); ok {
		ap, err := devAP.GetPropertyActiveAccessPoint()
		if err != nil {
			return dev, err
		}
		if ap == nil {
			dev.ActiveAccessPoint = nil
		} else {
			resolvedAP, err := ResolveAccessPoint(ap)
			if err != nil {
				return dev, err
			}
			dev.ActiveAccessPoint = &resolvedAP
		}
	}

	// Add device type prefix to ID
	dev.ID = deviceType.String() + "_" + dev.ID
	return dev, err
}

// RescanTimeoutSeconds is the maximum number of seconds since LastScan that can
// elapse before scanning for access points is requested
const RescanTimeoutSeconds = 10

// AccessPoint describes a network access point
type AccessPoint struct {
	SSID     string `json:"ssid"`
	BSSID    string `json:"bssid"`
	Strength uint8  `json:"strength"`
	Flags    uint32 `json:"flags"`
	WPAFlags uint32 `json:"wpaFlags"`
	RSNFlags uint32 `json:"rsnFlags"`
}

// ResolveAccessPoint returns an AccessPoint from a NetworkManager AccessPoint
func ResolveAccessPoint(ap nm.AccessPoint) (apOut AccessPoint, err error) {
	ssid, err := ap.GetPropertySSID()
	if err != nil {
		return apOut, err
	}
	apOut.SSID = string(ssid)

	apOut.BSSID, err = ap.GetPropertyHWAddress()
	if err != nil {
		return apOut, err
	}

	apOut.Strength, err = ap.GetPropertyStrength()
	if err != nil {
		return apOut, err
	}

	apOut.Flags, err = ap.GetPropertyFlags()
	if err != nil {
		return apOut, err
	}

	apOut.WPAFlags, err = ap.GetPropertyWPAFlags()
	if err != nil {
		return apOut, err
	}

	apOut.RSNFlags, err = ap.GetPropertyRSNFlags()
	if err != nil {
		return apOut, err
	}

	return apOut, err
}

// MarshallJSON returns a JSON representation of the AP
func (ap AccessPoint) MarshallJSON() ([]byte, error) {
	return json.Marshal(ap)
}
