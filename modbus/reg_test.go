package modbus

import (
	"errors"
	"testing"
)

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

func TestRegValidator(t *testing.T) {
	regs := Regs{}

	// add a register with a validator
	regs.AddReg(13, 1)
	err := regs.AddRegValueValidator(13, func(u uint16) bool { return u < 10 })
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// add another register without a validator
	regs.AddReg(14, 1)

	// add a validator on a register that does not exist (should return an error)
	err = regs.AddRegValueValidator(15, func(_ uint16) bool { return true })
	if !errors.Is(err, ErrUnknownRegister) {
		t.Errorf("Expected error to be ErrUnknownRegister")
	}

	// check if write fails correctly the value is invalid
	err = regs.WriteReg(13, 10)
	if !errors.Is(err, ExcIllegalValue) {
		t.Error("Expected error to be ExcIllegalValue")
	}

	// check if write succeeds if the value is valid
	err = regs.WriteReg(13, 9)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// no validator here, just make sure everything works without validators
	err = regs.WriteReg(14, 10)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
