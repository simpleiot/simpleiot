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
	slave := NewServer(id, portA)
	slave.Regs.AddCoil(128)
	err := slave.Regs.WriteCoil(128, true)
	if err != nil {
		t.Fatal(err)
	}

	// start slave so it can respond to requests
	go slave.Listen(func(err error) {
		log.Println("modbus server listen error: ", err)
	}, func(changes []RegChange) {
		log.Printf("modbus changes: %+v\n", changes)
	})

	// set up client (master)
	portB := respreader.NewReadWriter(wire.GetB(), time.Second*2,
		5*time.Millisecond)
	master := NewClient(portB)

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

	slave.Regs.WriteCoil(128, false)
	coils, _ = master.ReadCoils(id, 128, 1)

	if coils[0] != false {
		t.Fatal("wrong coil value")
	}

}
