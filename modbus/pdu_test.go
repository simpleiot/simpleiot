package modbus

import (
	"testing"
)

func TestPdu(t *testing.T) {
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
