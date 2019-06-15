package isserial

import (
	"errors"
	"fmt"
	"io"

	"github.com/simpleiot/simpleiot/modbus"
)

// LindsayState represents the state of a Lindsay System
type LindsayState uint16

// define valid Lindsay States
const (
	LindsayStateFault               LindsayState = 1
	LindsayStateSoftBarrierStopped               = 2
	LindsayStateServiceStop                      = 4
	LindsayStateLowPressureShutdown              = 5
	LindsayStatePressureWaiting                  = 11
	LindsayStateRunningForward                   = 14
	LindsayStateRunningReverse                   = 15
	LindsayStateStoppedNoPos                     = 21
	LindsayStateRunningNoPos                     = 29
)

func (ls LindsayState) String() (ret string) {
	switch ls {
	case LindsayStateFault:
		ret = "Safety Fault"
	case LindsayStateSoftBarrierStopped:
		ret = "Soft Barrier Stopped"
	case LindsayStateServiceStop:
		ret = "Service Stop"
	case LindsayStateLowPressureShutdown:
		ret = "Low Pressure Shutdown"
	case LindsayStatePressureWaiting:
		ret = "Pressure Waiting"
	case LindsayStateRunningForward:
		ret = "Running Forward"
	case LindsayStateRunningReverse:
		ret = "Running Reverse"
	case LindsayStateStoppedNoPos:
		ret = "Stopped No Position"
	case LindsayStateRunningNoPos:
		ret = "Running No Position"
	default:
		ret = "Unknown"
	}

	return
}

// LindsayStatusRegs represents modbus regs for the status messages
type LindsayStatusRegs struct {
	PosWithOffset    uint16
	PosWithoutOffset uint16
	Status           uint16
	Rate             uint16
	Pressure         uint16
	State            LindsayState
}

func (lsr LindsayStatusRegs) String() string {
	ret := fmt.Sprintf("Pos with offset: %v\n", lsr.PosWithOffset)
	ret += fmt.Sprintf("Pos without offset: %v\n", lsr.PosWithoutOffset)
	ret += fmt.Sprintf("Status: 0x%x\n", lsr.Status)
	ret += fmt.Sprintf("Rate: %v\n", lsr.Rate)
	ret += fmt.Sprintf("Pressure: %v\n", lsr.Pressure)
	ret += lsr.State.String()
	return ret
}

// Forward indicator
func (lsr *LindsayStatusRegs) Forward() bool {
	return (lsr.Status & (1 << 0)) != 0
}

// Reverse indicator
func (lsr *LindsayStatusRegs) Reverse() bool {
	return (lsr.Status & (1 << 1)) != 0
}

// WaterOn indicator
func (lsr *LindsayStatusRegs) WaterOn() bool {
	return (lsr.Status & (1 << 2)) != 0
}

// EndGun1On indicator
func (lsr *LindsayStatusRegs) EndGun1On() bool {
	return (lsr.Status & (1 << 3)) != 0
}

// EndGun2On indicator
func (lsr *LindsayStatusRegs) EndGun2On() bool {
	return (lsr.Status & (1 << 4)) != 0
}

// Accessory1On indicator
func (lsr *LindsayStatusRegs) Accessory1On() bool {
	return (lsr.Status & (1 << 5)) != 0
}

// AutoReverse indicator
func (lsr *LindsayStatusRegs) AutoReverse() bool {
	return (lsr.Status & (1 << 8)) != 0
}

// AutoRestart indicator
func (lsr *LindsayStatusRegs) AutoRestart() bool {
	return (lsr.Status & (1 << 9)) != 0
}

// NewLindsayStatusRegs create Lindsay status from modbus PDU packet
func NewLindsayStatusRegs(req modbus.FuncWriteMultipleRegisterRequest) (ret LindsayStatusRegs, err error) {
	if req.RegCount != 6 {
		err = errors.New("not enough regs for Lindsay Status")
		return
	}

	ret.PosWithOffset = req.RegValues[0]
	ret.PosWithoutOffset = req.RegValues[1]
	ret.Status = req.RegValues[2]
	ret.Rate = req.RegValues[3]
	ret.Pressure = req.RegValues[4]
	ret.State = LindsayState(req.RegValues[5])

	return
}

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
func (l *Lindsay) Read() (regs LindsayStatusRegs, err error) {
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

		regs, err = NewLindsayStatusRegs(req)
		return
	}

	err = errorNotLindsayStatus

	return
}
