package test

import "fmt"

// HexDump provides a string of bytes in hex format
func HexDump(data []byte) string {
	ret := ""

	for i, b := range data {
		if i != 0 {
			ret += " "
		}
		ret += fmt.Sprintf("%02x", b)
	}

	return ret
}
