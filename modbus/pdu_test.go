package modbus

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestPduWriteSingleCoil(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128) // add register 8 for coil 128
	regs.WriteCoil(128, true)

	pdu := WriteSingleCoil(128, false)

	_, resp, err := pdu.ProcessRequest(&regs)

	if err != nil {
		t.Errorf("Error processing request: %v", err)
	}

	if got, want := resp.FunctionCode, FuncCodeWriteSingleCoil; got != want {
		t.Errorf("got function code %x, want %x", got, want)
	}

	data := Uint16Array(resp.Data)
	if got, want := data[0], uint16(128); got != want {
		t.Errorf("got address %d, want %d", got, want)
	}
	if got, want := data[1], uint16(0); got != want {
		t.Errorf("got value %d, want %d", got, want)
	}
}

func TestPduWriteSingleCoilError(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128) // add register 8 for coil 128
	regs.WriteCoil(128, true)

	pdu := WriteSingleCoil(64, false)

	_, resp, err := pdu.ProcessRequest(&regs)

	if err != nil {
		t.Errorf("Error processing request: %v", err)
	}

	if got, want := resp.FunctionCode, 0x80|FuncCodeWriteSingleCoil; got != want {
		t.Errorf("got function code %x, want %x", got, want)
	}
	if len(resp.Data) != 1 || resp.Data[0] != byte(ExcIllegalAddress) {
		t.Errorf("got exception code %x, want %x", resp.Data[0], byte(ExcIllegalAddress))
	}
}

func TestPduReadHoldingRegs(t *testing.T) {
	regs := Regs{}
	regs.AddReg(8, 1)
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

func TestProcessRequest(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128) // add register 8 for coil 128
	regs.WriteCoil(128, true)
	regs.WriteCoil(130, true)
	regs.WriteCoil(132, true)
	regs.AddCoil(144) // add register 9 for coil 144

	for _, test := range []struct {
		name string
		in   []byte
		out  []byte
	}{
		{"ReadCoils/one", []byte{1, 0, 128, 0, 1}, []byte{1, 1, 1}},
		{"ReadCoils/multiple", []byte{1, 0, 128, 0, 3}, []byte{1, 1, 5}},
		{"ReadCoils/missing", []byte{1, 0, 16, 0, 1}, []byte{0x81, 2}},
		{"ReadDiscreteInputs/one", []byte{2, 0, 128, 0, 1}, []byte{2, 1, 1}},
		{"ReadDiscreteInputs/multiple", []byte{2, 0, 128, 0, 1}, []byte{2, 1, 1}},
		{"ReadDiscreteInputs/one", []byte{2, 0, 128, 0, 3}, []byte{2, 1, 5}},
		{"ReadDiscreteInputs/missing", []byte{2, 0, 16, 0, 1}, []byte{0x82, 2}},
		{"ReadHoldingRegisters/one", []byte{3, 0, 8, 0, 1}, []byte{3, 2, 0, 0x15}},
		{"ReadHoldingRegisters/multiple", []byte{3, 0, 8, 0, 2}, []byte{3, 4, 0, 0x15, 0, 0}},
		{"ReadHoldingRegisters/missing", []byte{3, 0, 7, 0, 2}, []byte{0x83, 2}},
		{"ReadInputRegisters/one", []byte{4, 0, 8, 0, 1}, []byte{4, 2, 0, 0x15}},
		{"ReadInputRegisters/multiple", []byte{4, 0, 8, 0, 2}, []byte{4, 4, 0, 0x15, 0, 0}},
		{"ReadInputRegisters/missing", []byte{4, 0, 7, 0, 2}, []byte{0x84, 2}},
		{"WriteSingleCoil/present", []byte{5, 0, 129, 0xFF, 0}, []byte{5, 0, 129, 0xFF, 0}},
		{"WriteSingleCoil/illegal", []byte{5, 0, 129, 10, 0}, []byte{0x85, 3}},
		{"WriteSingleCoil/missing", []byte{5, 0, 127, 0xFF, 0}, []byte{0x85, 2}},
		{"WriteSingleCoil/readback", []byte{1, 0, 128, 0, 3}, []byte{1, 1, 7}},
		{"WriteMultipleCoils/present", []byte{15, 0, 130, 0, 3, 1, 0x07}, []byte{15, 0, 130, 0, 3}},
		{"WriteMultipleCoils/missing", []byte{15, 0, 120, 0, 3, 1, 0x07}, []byte{0x8F, 2}},
		{"WriteMultipleCoils/readback", []byte{1, 0, 128, 0, 8}, []byte{1, 1, 0b00011111}},
		{"WriteSingleRegister/present", []byte{6, 0, 8, 0, 4}, []byte{6, 0, 8, 0, 4}},
		{"WriteSingleRegister/missing", []byte{6, 0, 6, 0, 4}, []byte{0x86, 2}},
		{"WriteSingleRegister/readback", []byte{3, 0, 8, 0, 1}, []byte{3, 2, 0, 4}},
		{"WriteMultipleRegisters/one", []byte{0x10, 0, 8, 0, 1, 2, 0, 8}, []byte{0x10, 0, 8, 0, 1}},
		{"WriteMultipleRegisters/multiple", []byte{0x10, 0, 8, 0, 2, 4, 0, 8, 10, 15}, []byte{0x10, 0, 8, 0, 2}},
		{"WriteMultipleRegisters/missing", []byte{0x10, 0, 7, 0, 2, 4, 9, 10, 11, 12}, []byte{0x90, 2}},
		{"WriteMultipleRegisters/wronglen", []byte{0x10, 0, 7, 0, 2, 4, 9, 10}, []byte{0x90, 3}},
		{"WriteMultipleRegisters/readback", []byte{3, 0, 8, 0, 2}, []byte{3, 4, 0, 8, 10, 15}},
	} {
		t.Run(test.name, func(t *testing.T) {
			pdu := &PDU{
				FunctionCode: FunctionCode(test.in[0]),
				Data:         test.in[1:],
			}
			_, resp, err := pdu.ProcessRequest(&regs)

			if err != nil {
				t.Errorf("Error processing request: %v", err)
			}

			want := PDU{
				FunctionCode: FunctionCode(test.out[0]),
				Data:         test.out[1:],
			}

			if diff := cmp.Diff(resp, want); diff != "" {
				t.Errorf("unexpected reply: got(-), want(+):\n%s", diff)
			}
			t.Logf("register state: %+v", regs)
		})
	}
}
