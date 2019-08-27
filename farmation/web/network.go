package web

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
)

func getNetwork() http.HandlerFunc {
	type interfaceData struct {
		HardwareAddr string
		/* Array of of network address information, including:
		- cidr - string form of net.Addr (CIDR notation IP address)
		- ip - string form of IP address only (decimal or colon separated)
		- ipNet - string form of IP *network* (CIDR notation)
		- netmask - string form of IP network mask in hexadecimal
		*/
		Addresses []map[string]string
	}
	return func(rw http.ResponseWriter, req *http.Request) {
		data := make(map[string]interfaceData)
		// Get information from all network interfaces
		ifaces, err := net.Interfaces()
		// Error handling
		if err != nil {
			const msg = "Error getting network interfaces"
			log.Println("web:", msg)
			http.Error(rw, msg, 500)
			return
		}
		for _, iface := range ifaces {
			// Get IP addresses for each interface
			addrs, err := iface.Addrs()
			if err != nil {
				// Ensure `addrs` is nil (is this needed?)
				addrs = nil
			}
			// Load interface data into map
			addressStrings := make([]map[string]string, len(addrs))
			for i, addr := range addrs {
				addressMap := make(map[string]string)
				ip, ipNet, err := net.ParseCIDR(addr.String())
				addressMap["cidr"] = addr.String()
				if err == nil {
					addressMap["ip"] = ip.String()
					addressMap["ipNet"] = ipNet.String()
					addressMap["netmask"] = ipNet.Mask.String()
				}
				addressStrings[i] = addressMap
			}
			data[iface.Name] = interfaceData{
				iface.HardwareAddr.String(), addressStrings}
		}
		json.NewEncoder(rw).Encode(data)
	}
}
