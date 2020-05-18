package modbus

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// the below data is Modbus ASCII
var testData1 = []byte{
	// request, adr 3, reading holding reg: 3, count: 6
	0x3A,
	0x30,
	0x33,
	0x30,
	0x33,
	0x30,
	0x30,
	0x30,
	0x30,
	0x30,
	0x30,
	0x30,
	0x36,
	0x46,
	0x34,
	0x0D,
	0x0A,
}

var testData2 = []byte{
	// response, adr 3, reading holding reg: 3, 12 bytes of data
	0x3A,
	0x30,
	0x33,
	0x30,
	0x33,
	0x30,
	0x43,
	0x30,
	0x31,
	0x30,
	0x36,
	0x30,
	0x30,
	0x30,
	0x30,
	0x30,
	0x31,
	0x35,
	0x44,
	0x30,
	0x30,
	0x36,
	0x33,
	0x30,
	0x33,
	0x31,
	0x41,
	0x32,
	0x30,
	0x30,
	0x30,
	0x45,
	0x39,
	0x0D,
	0x0A,
}

var testData3 = []byte{
	// broadcast (adr 0), Write Multiple registers (0x10), address 32768, count 6, byte count 12, data
	0x3A,
	0x30,
	0x30,
	0x31,
	0x30,
	0x38,
	0x30,
	0x30,
	0x30,
	0x30,
	0x30,
	0x30,
	0x36,
	0x30,
	0x43,
	0x30,
	0x30,
	0x32,
	0x32,
	0x30,
	0x30,
	0x36,
	0x33,
	0x30,
	0x32,
	0x30,
	0x30,
	0x32,
	0x31,
	0x33,
	0x34,
	0x30,
	0x30,
	0x30,
	0x35,
	0x30,
	0x30,
	0x30,
	0x30,
	0x37,
	0x44,
	0x0D,
	0x0A,
}

var testData = append(testData1, append(testData2, testData3...)...)

func TestScanner(t *testing.T) {
	buf := bytes.NewBuffer(testData)
	m := NewModbus(buf)

	count := 0

	for {
		data, err := m.Read()
		if err != nil {
			fmt.Println(err)
			break
		}

		fmt.Println(string(data))
		count++
	}

	if count != 3 {
		t.Error("Expected 3 frames")
	}
}

func TestDecodeASCIIByte(t *testing.T) {
	data := []byte("f423")

	dec, data, err := DecodeASCIIByte(data)
	if dec != 0xf4 {
		t.Errorf("dec error, got 0x%x\n", dec)
	}

	if string(data) != "23" {
		t.Errorf("returned data is %v\n", string(data))
	}

	if err != nil {
		t.Error(err)
	}
}

func TestASCIIDecode1(t *testing.T) {
	pdu, err := DecodeASCIIPDU(testData1)
	if err != nil {
		t.Error("error decoding: ", err)
	}

	if pdu.Address != 3 {
		t.Error("Error, wrong address, expected 3 got: ", pdu.Address)
	}

	if pdu.FunctionCode != FuncCodeReadHoldingRegisters {
		t.Error("Error reading function code, expected 3, got: ", pdu.FunctionCode)
	}
}

func TestASCIIDecode3(t *testing.T) {
	pdu, err := DecodeASCIIPDU(testData3)
	if err != nil {
		t.Error("error decoding: ", err)
	}

	if pdu.Address != 0 {
		t.Error("Error, wrong address, expected 0 got: ", pdu.Address)
	}

	if pdu.FunctionCode != FuncCodeWriteMultipleRegisters {
		t.Error("Error reading function code, got: ", pdu.FunctionCode)
	}

	fd, err := pdu.DecodeFunctionData()

	if err != nil {
		t.Error(err)
	}

	wmr := fd.(FuncWriteMultipleRegisterRequest)

	if wmr.StartingAddress != 32768 {
		t.Error("Wrong starting address: ", wmr.StartingAddress)
	}

	if wmr.RegCount != 6 {
		t.Error("Wrong reg count: ", wmr.RegCount)
	}

	if wmr.ByteCount != 12 {
		t.Error("Wrong byte count: ", wmr.ByteCount)
	}

	expectedRegData := []uint16{
		0x22,
		0x63,
		0x200,
		0x2134,
		0x5,
		0x0,
	}

	if !reflect.DeepEqual(expectedRegData, wmr.RegValues) {
		for i, r := range wmr.RegValues {
			fmt.Printf("exp: 0x%x, got: 0x%x\n", expectedRegData[i], r)
		}
		t.Error("reg values are not correct")
	}

}
