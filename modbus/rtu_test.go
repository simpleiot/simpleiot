package modbus

import (
	"reflect"
	"testing"
)

// examples from MPE SC2000
// Examples: Reading the Wet Well Level in Modbus register 40011, with slave address
// of 1, Function code 03
// Actual Modbus message:              [001] [003] [000] [010] [000] [001] [164] [008]
//
// Reading the High Level Alarm in Modbus coil 129, with slave address of 1,
// Function code 01
// Actual Modbus message:              [001] [001] [000] [128] [000] [001] [252] [034]

var rtuSc2000Test1 = []byte{1, 3, 0, 10, 0, 1, 164, 8}
var rtuSc2000Test2 = []byte{1, 1, 0, 128, 0, 1, 252, 34}

func TestRtuSc2000Test1(t *testing.T) {
	err := CheckRtuCrc(rtuSc2000Test1)
	if err != nil {
		t.Error("rtuSc2000Test1 CRC check failed")
	}
}

func TestRtuSc2000Test2(t *testing.T) {
	err := CheckRtuCrc(rtuSc2000Test2)
	if err != nil {
		t.Error("rtuSc2000Test2 CRC check failed")
	}

	packet, err := RtuEncode(1, ReadCoils(128, 1))

	if err != nil {
		t.Error("Error encoding packet: ", err)
	}

	if !reflect.DeepEqual(rtuSc2000Test2, packet) {
		t.Error("packet encoding error")
	}
}
