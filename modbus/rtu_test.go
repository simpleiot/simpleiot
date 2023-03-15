package modbus

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/simpleiot/simpleiot/test"
)

// examples from MPE SC2000
// Reading the Wet Well Level in Modbus register 40011, with slave address of 1, Function code 03.
//ModScan32 Request: [001] [003] [000] [010] [000] [001] [164] [008]
//SC2000 Response: [001] [003] [002] [000] [098] [057] [173]   (9.8ft)
//SC2000 Response: [001] [003] [002] [000] [090] [056] [127]   (9.0ft)
//
//Reading the High Level Alarm in Modbus coil 129, with slave address of 1, Function code 01,  .
// ModScan32 Request: [001] [001] [000] [128] [000] [001] [252] [034]
// SC2000 Response: [001] [001] [001] [000] [081] [136] (no high level alarm)
// SC2000 Response: [001] [001] [001] [001] [144] [072] (high level alarm)

var rtuSc2000LevelPrompt = []byte{1, 3, 0, 10, 0, 1, 164, 8}
var rtuSc2000LevelResp = []byte{1, 3, 2, 0, 98, 57, 173}

var rtuSc2000CoilPrompt = []byte{1, 1, 0, 128, 0, 1, 252, 34}
var rtuSc2000CoilResp = []byte{1, 1, 1, 1, 144, 72}

func TestRtuSc2000Level(t *testing.T) {
	err := CheckRtuCrc(rtuSc2000LevelPrompt)
	if err != nil {
		t.Fatal("rtuSc2000LevelPrompt CRC check failed")
	}

	rtu := NewRTU(nil)

	prompt := ReadHoldingRegs(10, 1)
	promptRtu, err := rtu.Encode(1, prompt)

	if err != nil {
		t.Fatal("error encoding")
	}

	if !reflect.DeepEqual(rtuSc2000LevelPrompt, promptRtu) {
		t.Fatal("encoded packet is not as expected")
	}

	regs := Regs{}
	regs.AddReg(10, 1)
	_ = regs.WriteReg(10, 98)

	_, resp, err := prompt.ProcessRequest(&regs)
	if err != nil {
		t.Fatal("error processing: ", err)
	}

	respRtu, err := rtu.Encode(1, resp)
	if err != nil {
		t.Fatal("resp encode error: ", err)
	}

	if !reflect.DeepEqual(rtuSc2000LevelResp, respRtu) {
		fmt.Println("Expected: ", test.HexDump(rtuSc2000LevelResp))
		fmt.Println("Got:      ", test.HexDump(respRtu))
		fmt.Println("resp packet is not right")
	}

}

func TestRtuSc2000Coil(t *testing.T) {
	err := CheckRtuCrc(rtuSc2000CoilPrompt)
	if err != nil {
		t.Error("rtuSc2000CoilPrompt CRC check failed")
	}

	rtu := NewRTU(nil)

	prompt := ReadCoils(128, 1)
	promptRtu, err := rtu.Encode(1, prompt)

	if err != nil {
		t.Error("Error encoding packet: ", err)
	}

	if !reflect.DeepEqual(rtuSc2000CoilPrompt, promptRtu) {
		t.Error("packet encoding error")
	}

	regs := Regs{}
	regs.AddCoil(128)
	_ = regs.WriteCoil(128, true)

	_, resp, err := prompt.ProcessRequest(&regs)
	if err != nil {
		t.Fatal("error processing: ", err)
	}

	respRtu, err := rtu.Encode(1, resp)
	if err != nil {
		t.Fatal("resp encode error: ", err)
	}

	if !reflect.DeepEqual(rtuSc2000CoilResp, respRtu) {
		fmt.Println("Expected: ", test.HexDump(rtuSc2000CoilResp))
		fmt.Println("Got:      ", test.HexDump(respRtu))
		fmt.Println("resp packet is not right")
	}
}
