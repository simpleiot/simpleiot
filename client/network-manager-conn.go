//go:build linux
// +build linux

package client

import (
	"encoding/binary"
	"math/bits"
	"net"
	"reflect"
	"strings"

	nm "github.com/Wifx/gonetworkmanager"
	"github.com/godbus/dbus/v5"
)

// NetworkManagerConn defines a NetworkManager connection
type NetworkManagerConn struct {
	ID     string `node:"id"`
	Parent string `node:"parent"`
	UUID   string `point:"uuid"`
	// Type is one of the NetworkManager connection types (i.e. 802-3-ethernet)
	// See https://developer-old.gnome.org/NetworkManager/stable/
	Type string `point:"type"`
	// Disabled flag removes the connection from NetworkManager entirely
	Disabled            bool   `point:"disabled"`
	AutoConnect         bool   `point:"autoConnect"`
	AutoConnectPriority int32  `point:"autoConnectPriority"`
	InterfaceName       string `point:"interfaceName"`
	// LastActivated is the timestamp the connection was last activated in
	// seconds since the UNIX epoch. Called "timestamp" in NetworkManager.
	LastActivated uint64     `point:"lastActivated"`
	IPv4Config    IPv4Config `point:"ipv4Config"`
	IPv6Config    IPv6Config `point:"ipv6Config"`
	WiFiConfig    WiFiConfig `point:"wiFiConfig"`
}

// WiFiConfig defines 802.11 wireless configuration
type WiFiConfig struct {
	SSID string `point:"ssid"`
	// From NetworkManager: Key management used for the connection. One of
	// "none" (WEP), "ieee8021x" (Dynamic WEP), "wpa-none" (Ad-Hoc WPA-PSK),
	// "wpa-psk" (infrastructure WPA-PSK), "sae" (SAE) or "wpa-eap"
	// (WPA-Enterprise). This property must be set for any Wi-Fi connection that
	// uses security.
	KeyManagement string `point:"keyManagement"`
	PSK           string `point:"psk"`
}

// ResolveNetworkManagerConn returns a NetworkManagerConn from D-Bus settings
func ResolveNetworkManagerConn(settings nm.ConnectionSettings) NetworkManagerConn {
	sc := settings["connection"]
	conn := NetworkManagerConn{
		ID:          sc["id"].(string),
		UUID:        sc["uuid"].(string),
		Type:        sc["type"].(string),
		AutoConnect: true,
	}
	if val, ok := sc["autoconnect"].(bool); ok {
		conn.AutoConnect = val
	}
	if val, ok := sc["autoconnect-priority"].(int32); ok {
		conn.AutoConnectPriority = val
	}
	if val, ok := sc["interface-name"].(string); ok {
		conn.InterfaceName = val
	}
	if val, ok := sc["timestamp"].(uint64); ok {
		conn.LastActivated = val
	}

	// Parse IPv4 / IPv6 settings
	if val, ok := settings["ipv4"]; ok {
		conn.IPv4Config = ResolveIPv4Config(val)
	}
	if val, ok := settings["ipv6"]; ok {
		conn.IPv6Config = ResolveIPv6Config(val)
	}

	return conn
}

// Managed returns true if the connection is managed by the SimpleIoT client
// Returns true if and only if the connection ID is prefixed with `SimpleIoT:`
func (c NetworkManagerConn) Managed() bool {
	return strings.HasPrefix(c.ID, "SimpleIoT:")
}

// DBus returns an object that can be passed over D-Bus
// Returns nil if the connection ID does not include the prefix `SimpleIoT:`
// See https://developer-old.gnome.org/NetworkManager/stable/ch01.html
func (c NetworkManagerConn) DBus() nm.ConnectionSettings {
	if !c.Managed() {
		return nil
	}
	sc := map[string]any{
		"id":                   c.ID,
		"type":                 c.Type,
		"autoconnect":          c.AutoConnect,
		"autoconnect-priority": c.AutoConnectPriority,
	}
	if c.UUID != "" {
		sc["uuid"] = c.UUID
	}
	if c.InterfaceName != "" {
		sc["interface-name"] = c.InterfaceName
	}
	settings := nm.ConnectionSettings{
		"connection": sc,
		"ipv4":       c.IPv4Config.DBus(),
		"ipv6":       c.IPv6Config.DBus(),
	}
	if c.Type == "802-11-wireless" && c.WiFiConfig.SSID != "" {
		settings["802-11-wireless"] = map[string]any{
			"ssid": []byte(c.WiFiConfig.SSID),
		}
	}
	return settings
}

// Equal returns true if and only if the two connections are equal
func (c NetworkManagerConn) Equal(c2 NetworkManagerConn) bool {
	return reflect.DeepEqual(c, c2)
}

// IPv4Address is a string representation of an IPv4 address
type IPv4Address string

// IPv4AddressUint32 converts an IPv4 address in uint32 format to a string
func IPv4AddressUint32(n uint32, order binary.ByteOrder) IPv4Address {
	buf := []byte{0, 0, 0, 0}
	order.PutUint32(buf, n)
	return IPv4Address(net.IP(buf).String())
}

// Uint32 convert an IPv4 address in string format to a uint32
func (addr IPv4Address) Uint32(order binary.ByteOrder) uint32 {
	ip := net.ParseIP(addr.String()).To4()
	if len(ip) != 4 {
		return 0
	}
	return order.Uint32(ip)
}

// Valid returns true if string is valid IPv4 address
func (addr IPv4Address) Valid() bool {
	str := addr.String()
	return strings.Contains(str, ".") && net.ParseIP(str).To4() != nil
}

// String returns the underlying string
func (addr IPv4Address) String() string {
	return string(addr)
}

// IPv4Netmask is a string representation of an IPv4 netmask
type IPv4Netmask IPv4Address

// IPv4NetmaskPrefix converts an integer IPv4 prefix to netmask string
func IPv4NetmaskPrefix(prefix uint8) IPv4Netmask {
	var mask uint32 = 0xFFFFFFFF << (32 - prefix)
	return IPv4Netmask(IPv4AddressUint32(mask, binary.BigEndian))
}

// Prefix converts a subnet mask string to IPv4 prefix
func (str IPv4Netmask) Prefix() uint32 {
	return uint32(bits.OnesCount32(IPv4Address(str).Uint32(binary.BigEndian)))
}

// Valid returns true if subnet mask in dot notation is valid
func (str IPv4Netmask) Valid() bool {
	if !IPv4Address(str).Valid() {
		return false
	}

	mask := IPv4Address(str).Uint32(binary.BigEndian)
	return (mask & (^mask >> 1)) <= 0
}

// IPv4Config defines data for IPv4 config
type IPv4Config struct {
	StaticIP   bool        `json:"staticIP" point:"staticIP"`
	Address    IPv4Address `json:"address" point:"address"`
	Netmask    IPv4Netmask `json:"netmask" point:"netmask"`
	Gateway    IPv4Address `json:"gateway" point:"gateway"`
	DNSServer1 IPv4Address `json:"dnsServer1" point:"dnsServer1"`
	DNSServer2 IPv4Address `json:"dnsServer2" point:"dnsServer2"`
}

// ResolveIPv4Config creates a new IPv4Config from a map of D-Bus settings
func ResolveIPv4Config(settings map[string]any) IPv4Config {
	c := IPv4Config{
		// 'method' setting can be 'auto', 'manual', or 'link-local'
		// Go is so cool; you can compare interface{} with a string no problem
		StaticIP: settings["method"] == "manual",
	}

	// Note: 'address-data' is []map[string]dbus.Variant where each map has
	// "address" (string) and "prefix" (uint32) keys
	addressData, ok := settings["address-data"].([]map[string]dbus.Variant)
	if ok && len(addressData) > 0 {
		// Note: dbus.Variant is a struct, and its zero value is valid
		str, _ := addressData[0]["address"].Value().(string)
		c.Address = IPv4Address(str)
		// Convert integer prefix to string subnet mask format
		if prefix, ok := addressData[0]["prefix"].Value().(uint32); ok &&
			prefix <= 32 {
			c.Netmask = IPv4NetmaskPrefix(uint8(prefix))
		}
	}

	str, _ := settings["gateway"].(string)
	c.Gateway = IPv4Address(str)

	// 'dns' setting is slice of IP addresses in uint32 format
	dns, _ := settings["dns"].([]uint32)
	if len(dns) > 0 {
		c.DNSServer1 = IPv4AddressUint32(dns[0], binary.LittleEndian)
	}
	if len(dns) > 1 {
		c.DNSServer2 = IPv4AddressUint32(dns[1], binary.LittleEndian)
	}

	return c
}

// DBus returns the IPv4 settings in a generic map to be sent over D-Bus
// See https://developer-old.gnome.org/NetworkManager/stable/settings-ipv4.html
func (c IPv4Config) DBus() map[string]any {
	if !c.StaticIP ||
		!c.Address.Valid() ||
		!c.Netmask.Valid() ||
		!c.Gateway.Valid() {

		// Automatic (DHCP)
		return map[string]any{
			"method": "auto",
		}
	}

	// Manual (Static IP)
	settings := map[string]any{
		"method": "manual",
	}

	if c.Address != "" && c.Netmask != "" && c.Gateway != "" {
		settings["address-data"] = []map[string]any{{
			"address": c.Address.String(),
			"prefix":  c.Netmask.Prefix(),
		}}
		settings["gateway"] = c.Gateway.String()
	}

	dns := make([]uint32, 0, 2)
	if c.DNSServer1.Valid() {
		dns = append(dns, c.DNSServer1.Uint32(binary.LittleEndian))
	}
	if c.DNSServer2.Valid() {
		dns = append(dns, c.DNSServer2.Uint32(binary.LittleEndian))
	}
	settings["dns"] = dns

	return settings
}

// IPv6Address is a string representation of an IPv4 address
type IPv6Address string

// NewIPv6Address converts an IPv6 address in []byte format to a string
func NewIPv6Address(bs []byte) IPv6Address {
	return IPv6Address(net.IP(bs).To16().String())
}

// Valid return true if string is valid IPv6 address
func (addr IPv6Address) Valid() bool {
	str := addr.String()
	return strings.Contains(str, ":") && net.ParseIP(str).To16() != nil
}

// Bytes convert an IPv6 address in string format to []byte
func (addr IPv6Address) Bytes() []byte {
	return []byte(net.ParseIP(addr.String()).To16())
}

// String returns the underlying string
func (addr IPv6Address) String() string {
	return string(addr)
}

// IPv6Config defines data for IPv6 configs
type IPv6Config struct {
	StaticIP   bool        `json:"staticIP"`
	Address    IPv6Address `json:"address"`
	Prefix     uint8       `json:"prefix"`
	Gateway    IPv6Address `json:"gateway"`
	DNSServer1 IPv6Address `json:"dnsServer1"`
	DNSServer2 IPv6Address `json:"dnsServer2"`
}

// ResolveIPv6Config creates a new IPv6Config from a map of D-Bus settings
func ResolveIPv6Config(settings map[string]any) IPv6Config {
	c := IPv6Config{
		StaticIP: settings["method"] == "manual",
	}

	addressData, ok := settings["address-data"].([]map[string]dbus.Variant)
	if ok && len(addressData) > 0 {
		str, _ := addressData[0]["address"].Value().(string)
		c.Address = IPv6Address(str)
		prefix, ok := addressData[0]["prefix"].Value().(uint32)
		if ok && prefix <= 128 {
			c.Prefix = uint8(prefix)
		}
	}

	str, _ := settings["gateway"].(string)
	c.Gateway = IPv6Address(str)

	// 'dns' setting is slice of IP addresses as 16-byte slices
	dns, _ := settings["dns"].([][]byte)
	if len(dns) > 0 {
		c.DNSServer1 = NewIPv6Address(dns[0])
	}
	if len(dns) > 1 {
		c.DNSServer2 = NewIPv6Address(dns[1])
	}

	return c
}

// DBus returns the IPv6 settings in a generic map to be sent over D-Bus
// See https://developer-old.gnome.org/NetworkManager/stable/settings-ipv6.html
func (c IPv6Config) DBus() map[string]any {

	if !c.StaticIP ||
		!c.Address.Valid() ||
		!c.Gateway.Valid() {

		// Automatic (DHCP)
		return map[string]any{
			"method": "auto",
		}
	}

	settings := map[string]any{
		"method": "manual",
	}

	if c.Address.Valid() && c.Gateway.Valid() {
		settings["address-data"] = []map[string]any{{
			"address": c.Address.String(),
			"prefix":  c.Prefix,
		}}
		settings["gateway"] = c.Gateway.String()
	}

	dns := make([][]byte, 0, 2)
	if c.DNSServer1.Valid() {
		dns = append(dns, c.DNSServer1.Bytes())
	}
	if c.DNSServer2 != "" {
		dns = append(dns, c.DNSServer2.Bytes())
	}
	settings["dns"] = dns

	return settings
}
