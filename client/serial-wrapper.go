package client

import "github.com/simpleiot/simpleiot/data"

// SerialEncode can be used in a client to encode points sent over a serial link.
func SerialEncode(seq byte, subject string, points data.Points) []byte {
	return []byte{}
}

// SerialDecode can be used to decode serial data in a client.
func SerialDecode(data []byte) (byte, string, data.Points, error) {
	return 0, "", nil, nil
}
