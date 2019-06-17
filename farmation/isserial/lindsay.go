package isserial

import (
	"errors"
	"io"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/modbus"
)

// Lindsay reads data from a panel
type Lindsay struct {
	modbus *modbus.Modbus
}

// NewLindsay creates a new reader
func NewLindsay(port io.ReadWriter) *Lindsay {
	return &Lindsay{
		modbus: modbus.NewModbus(port),
	}
}

var errorNotLindsayStatus = errors.New("Not lindsay status")

// Read waits for a Lindsay status regs and then returns it
func (l *Lindsay) Read() (regs isdata.LindsayStatusRegs, err error) {
	var data []byte
	data, err = l.modbus.Read()
	if err != nil {
		return
	}

	var pdu modbus.PDU
	pdu, err = modbus.DecodeASCIIPDU(data)

	if pdu.FunctionCode == modbus.FuncCodeWriteMultipleRegisters {
		var reqI interface{}
		reqI, err = pdu.DecodeFunctionData()
		if err != nil {
			return
		}

		var req modbus.FuncWriteMultipleRegisterRequest
		var ok bool
		req, ok = reqI.(modbus.FuncWriteMultipleRegisterRequest)
		if !ok {
			err = errors.New("Error converting modbus req type")
			return
		}

		regs, err = isdata.NewLindsayStatusRegs(req)
		return
	}

	err = errorNotLindsayStatus

	return
}
