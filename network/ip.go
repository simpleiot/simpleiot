package network

import (
	"errors"
	"net"
)

// GetIP returns the IP address for the itnerface
func GetIP(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if !v.IP.IsLoopback() && v.IP.To4() != nil {
				return addr.String(), nil
			}
		}
	}

	return "", errors.New("no IP address")
}
