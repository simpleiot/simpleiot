package modbus

import "testing"

func TestUint32(t *testing.T) {
	v := uint32(412345623)

	regs := Uint32ToRegs([]uint32{v})

	v2 := RegsToUint32(regs)

	if v != v2[0] {
		t.Error("Failed: ", v, v2[0])
	}
}

func TestInt32(t *testing.T) {
	v := int32(-412345623)

	regs := Int32ToRegs([]int32{v})

	v2 := RegsToInt32(regs)

	if v != v2[0] {
		t.Error("Failed: ", v, v2[0])
	}
}
