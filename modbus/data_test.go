package modbus

import (
	"testing"
)

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

func TestFloat32(t *testing.T) {
	v := float32(2124.23e18)

	regs := Float32ToRegs([]float32{v})

	v2 := RegsToFloat32(regs)

	if v != v2[0] {
		t.Error("Failed: ", v, v2[0])
	}
}

func TestRegsToFloat32SwapWords(t *testing.T) {
	exp := float32(0.01)

	f := RegsToFloat32SwapWords([]uint16{0xd70a, 0x3c23})

	if exp != f[0] {
		t.Error("Failed: ", exp, f)
	}
}
