package modbus

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPduReadCoils(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128) // add register 8 for coil 128
	_ = regs.WriteCoil(128, true)

	pdu := ReadCoils(128, 1)

	_, resp, err := pdu.ProcessRequest(&regs)

	if err != nil {
		t.Errorf("Error processing request: %v", err)
	}

	bits, err := resp.RespReadBits(1)

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
	_ = regs.WriteCoil(128, true)

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
	_ = regs.WriteCoil(128, true)

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
	_ = regs.WriteReg(8, 0x1234)

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
	_ = regs.WriteCoil(128, true)
	_ = regs.WriteCoil(130, true)
	_ = regs.WriteCoil(132, true)
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
			t.Logf("register state: %+v", &regs)
		})
	}
}

// TestProcessRequestBounds checks that requests outside the protocol limits
// get an illegal-data-value exception instead of a panic.
func TestProcessRequestBounds(t *testing.T) {
	regs := Regs{}
	regs.AddCoil(128)
	regs.AddReg(8, 1)

	for _, test := range []struct {
		name string
		in   []byte
	}{
		{"ReadHoldingRegisters/0x8000", []byte{3, 0, 8, 0x80, 0x00}},
		{"ReadHoldingRegisters/126", []byte{3, 0, 8, 0, 126}},
		{"ReadHoldingRegisters/zero", []byte{3, 0, 8, 0, 0}},
		{"ReadInputRegisters/0x8000", []byte{4, 0, 8, 0x80, 0x00}},
		{"ReadInputRegisters/zero", []byte{4, 0, 8, 0, 0}},
		{"ReadCoils/2048", []byte{1, 0, 128, 0x08, 0x00}},
		{"ReadCoils/2001", []byte{1, 0, 128, 0x07, 0xd1}},
		{"ReadCoils/zero", []byte{1, 0, 128, 0, 0}},
		{"ReadDiscreteInputs/2048", []byte{2, 0, 128, 0x08, 0x00}},
		{"ReadDiscreteInputs/zero", []byte{2, 0, 128, 0, 0}},
		{"WriteMultipleCoils/zero", []byte{15, 0, 128, 0, 0, 0, 0}},
		{"WriteMultipleCoils/0x8000", []byte{15, 0, 128, 0x80, 0, 0, 0}},
		{"WriteMultipleRegisters/zero", []byte{0x10, 0, 8, 0, 0, 0, 0, 0}},
		{"WriteMultipleRegisters/0x8000", []byte{0x10, 0, 8, 0x80, 0, 0, 0, 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			pdu := &PDU{
				FunctionCode: FunctionCode(test.in[0]),
				Data:         test.in[1:],
			}
			changed, resp, err := pdu.ProcessRequest(&regs)
			if err != nil {
				t.Fatalf("Error processing request: %v", err)
			}
			if changed {
				t.Error("registers must not change")
			}
			if resp.FunctionCode != pdu.FunctionCode|0x80 {
				t.Errorf("got function code %x, want exception", resp.FunctionCode)
			}
			if len(resp.Data) != 1 || resp.Data[0] != byte(ExcIllegalValue) {
				t.Errorf("got %v, want illegal data value exception", resp.Data)
			}
		})
	}
}

// TestProcessRequestLimits checks the largest legal counts still work.
func TestProcessRequestLimits(t *testing.T) {
	regs := Regs{}
	for i := 0; i < MaxReadBits; i++ {
		regs.AddCoil(i)
	}
	regs.AddReg(0, MaxReadRegs)

	req := ReadCoils(0, MaxReadBits)
	_, resp, err := req.ProcessRequest(&regs)
	if err != nil {
		t.Fatal(err)
	}
	if bits, err := resp.RespReadBits(MaxReadBits); err != nil || len(bits) != MaxReadBits {
		t.Errorf("read %v coils: %v, %v", MaxReadBits, len(bits), err)
	}

	req = ReadHoldingRegs(0, MaxReadRegs)
	_, resp, err = req.ProcessRequest(&regs)
	if err != nil {
		t.Fatal(err)
	}
	if vals, err := resp.RespReadRegs(MaxReadRegs); err != nil || len(vals) != MaxReadRegs {
		t.Errorf("read %v regs: %v, %v", MaxReadRegs, len(vals), err)
	}
}

// TestRespReadBadLength checks that a response whose byte count disagrees
// with the request returns an error instead of indexing past the buffer.
func TestRespReadBadLength(t *testing.T) {
	for _, test := range []struct {
		name  string
		bits  bool
		pdu   PDU
		count uint16
	}{
		{"bits/count larger than data", true, PDU{FuncCodeReadCoils, []byte{200, 1}}, 8},
		{"bits/count short", true, PDU{FuncCodeReadCoils, []byte{1, 1}}, 16},
		{"bits/count claims more than present", true, PDU{FuncCodeReadCoils, []byte{2, 1}}, 16},
		{"bits/empty", true, PDU{FuncCodeReadCoils, []byte{}}, 1},
		{"bits/zero request", true, PDU{FuncCodeReadCoils, []byte{0}}, 0},
		{"bits/exception", true, PDU{FuncCodeReadCoils | 0x80, []byte{3}}, 1},
		{"bits/wrong function", true, PDU{FuncCodeReadHoldingRegisters, []byte{2, 0, 1}}, 1},
		{"regs/count larger than data", false, PDU{FuncCodeReadHoldingRegisters, []byte{250, 0, 1}}, 1},
		{"regs/count short", false, PDU{FuncCodeReadHoldingRegisters, []byte{2, 0, 1}}, 2},
		{"regs/count claims more than present", false, PDU{FuncCodeReadHoldingRegisters, []byte{4, 0, 1}}, 2},
		{"regs/odd", false, PDU{FuncCodeReadHoldingRegisters, []byte{3, 0, 1, 2}}, 1},
		{"regs/empty", false, PDU{FuncCodeReadHoldingRegisters, []byte{}}, 1},
		{"regs/exception", false, PDU{FuncCodeReadHoldingRegisters | 0x80, []byte{2}}, 1},
		{"regs/wrong function", false, PDU{FuncCodeReadCoils, []byte{2, 0, 1}}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.bits {
				_, err = test.pdu.RespReadBits(test.count)
			} else {
				_, err = test.pdu.RespReadRegs(test.count)
			}
			if err == nil {
				t.Error("expected error")
			}
		})
	}

	// an exception response surfaces as the exception
	_, err := (&PDU{FuncCodeReadHoldingRegisters | 0x80, []byte{2}}).RespReadRegs(1)
	if err != ExcIllegalAddress {
		t.Errorf("got %v, want %v", err, ExcIllegalAddress)
	}
}

func TestRespReadBits(t *testing.T) {
	pdu := PDU{FuncCodeReadCoils, []byte{2, 0b10000001, 0b00000010}}
	bits, err := pdu.RespReadBits(10)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, false, false, false, false, false, true, false, true}
	if diff := cmp.Diff(bits, want); diff != "" {
		t.Error(diff)
	}
}
