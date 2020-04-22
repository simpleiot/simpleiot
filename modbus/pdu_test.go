package modbus

import (
	"testing"
)

func TestPdu(t *testing.T) {
	s := Regs{}
	s.AddReg(8) // add register 8 for coil 128
	s.WriteReg(8, 1)

	pdu := ReadCoils(128, 1)

	_, resp, err := pdu.ProcessRequest(&s)

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
