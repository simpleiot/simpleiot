package modbus

import (
	"encoding/binary"
	"errors"
)

// RtuCrc calculates CRC for a Modbus RTU packet
func RtuCrc(buf []byte) uint16 {
	crc := uint16(0xFFFF)

	for _, b := range buf {
		crc ^= uint16(b) // XOR byte into least sig. byte of crc

		for i := 8; i != 0; i-- { // Loop over each bit
			if (crc & 0x0001) != 0 { // If the LSB is set
				crc >>= 1 // Shift right and XOR 0xA001
				crc ^= 0xA001
			} else { // Else LSB is not set
				crc >>= 1 // Just shift right
			}
		}
	}
	// Note, this number has low and high bytes swapped, so use it accordingly (or swap bytes)
	return (crc >> 8) | (crc << 8)
}

// ErrCrc is returned if a crc check fails
var ErrCrc = errors.New("CRC error")

// ErrNotEnoughData is returned if not enough data
var ErrNotEnoughData = errors.New("Not enough data to calculate CRC")

// CheckRtuCrc returns error if CRC fails
func CheckRtuCrc(packet []byte) error {
	if len(packet) < 4 {
		return ErrNotEnoughData
	}

	crcCalc := RtuCrc(packet[:len(packet)-2])

	crcPacket := binary.BigEndian.Uint16(packet[len(packet)-2 : len(packet)])
	if crcCalc != crcPacket {
		return ErrCrc
	}

	return nil
}
