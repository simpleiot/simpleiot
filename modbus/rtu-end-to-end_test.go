package modbus

import (
	"log"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/respreader"
)

func TestRtuEndToEnd(t *testing.T) {

	id := byte(1)

	// create virtual serial wire to simulate connection between
	// server and client
	wire := NewIoSim(false)

	// first set up the server (slave) to process data
	portA := respreader.NewReadWriter(wire.GetA(), time.Second*2,
		5*time.Millisecond)
	transportA := NewRTU(portA)
	regs := &Regs{}
	slave := NewServer(id, transportA, regs)
	regs.AddCoil(128)
	err := regs.WriteCoil(128, true)
	if err != nil {
		t.Fatal(err)
	}

	regs.AddReg(2, 1)
	err = regs.WriteReg(2, 0x1234)
	if err != nil {
		t.Fatal(err)
	}

	// start slave so it can respond to requests
	go slave.Listen(9, func(err error) {
		log.Println("modbus server listen error: ", err)
	}, func() {
		log.Printf("modbus reg changes")
	})

	// set up client (master)
	portB := respreader.NewReadWriter(wire.GetB(), time.Second*2,
		5*time.Millisecond)
	transportB := NewRTU(portB)
	master := NewClient(transportB, 9)

	coils, err := master.ReadCoils(id, 128, 1)
	if err != nil {
		t.Fatal("read coils returned err: ", err)
	}
	if len(coils) != 1 {
		t.Fatal("invalid coil length")
		return
	}

	if coils[0] != true {
		t.Fatal("wrong coil value")
	}

	regs.WriteCoil(128, false)
	coils, _ = master.ReadCoils(id, 128, 1)

	if coils[0] != false {
		t.Fatal("wrong coil value")
	}

	hr, err := master.ReadHoldingRegs(id, 2, 1)
	if err != nil {
		t.Fatal("read holding regs returned err: ", err)
	}

	if len(hr) != 1 {
		t.Fatal("invalid regs length")
	}

	if hr[0] != 0x1234 {
		t.Fatalf("read holding reg returned wrong value: 0x%x", hr[0])
	}
}
