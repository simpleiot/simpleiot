package modbus

import (
	"testing"
)

func TestPduReadCoils(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128) // add register 8 for coil 128
	regs.WriteCoil(128, true)

	pdu := ReadCoils(128, 1)

	_, resp, err := pdu.ProcessRequest(&regs)

	if err != nil {
		t.Errorf("Error processing request: %v", err)
	}

	bits, err := resp.RespReadBits()

	if err != nil {
		t.Errorf("Error getting bits: %v", err)
	}

	if len(bits) != 1 {
		t.Errorf("expected 1 bit, got %v", len(bits))
	}

	if !bits[0] {
		t.Error("Expected high bit")
	}
}

func TestPduReadHoldingRegs(t *testing.T) {
	regs := Regs{}
	regs.AddReg(8)
	regs.WriteReg(8, 0x1234)

	pdu := ReadHoldingRegs(8, 1)

	_, resp, err := pdu.ProcessRequest(&regs)

	if err != nil {
		t.Fatal("Error processing request: ", err)
	}

	if resp.Data[0] != 2 {
		t.Fatal("expected byte count to be 2")
	}

	values := Uint16Array(resp.Data[1:])
	if len(values) != 1 {
		t.Fatal("Expected 1 values in response")
	}

	if values[0] != 0x1234 {
		t.Fatal("wrong value")
	}
}
