package modbus

import "testing"

func TestCoil(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128) // add register 8 for coil 128
	err := regs.WriteCoil(128, true)
	if err != nil {
		t.Error("Error writing coil")
	}

	reg, err := regs.ReadReg(8)
	if err != nil {
		t.Error(err)
	}

	if reg&0x1 != 0x1 {
		t.Error("expected bit 1 to be high")
	}

	err = regs.WriteCoil(128, false)
	if err != nil {
		t.Error("Error writing coil")
	}

	reg, err = regs.ReadReg(8)
	if err != nil {
		t.Error(err)
	}

	if reg != 0 {
		t.Errorf("Expected reg to be 0, got 0x%x\n", reg)
	}
}
