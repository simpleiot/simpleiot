package isdata

import (
	"errors"
	"fmt"

	"github.com/simpleiot/simpleiot/modbus"
)

// LindsayState represents the state of a Lindsay System
type LindsayState uint16

// define valid Lindsay States
const (
	LindsayStateStopped             LindsayState = 0x0
	LindsayStateFault                            = 0x1
	LindsayStateSoftBarrierStopped               = 0x2
	LindsayStateServiceStop                      = 0x4
	LindsayStateLowPressureShutdown              = 0x5
	LindsayStateLowVoltageFault                  = 0x6
	LindsayStateRestartDelay                     = 0x10
	LindsayStatePressureWaiting                  = 0x11
	LindsayStateRunningForward                   = 0x14
	LindsayStateRunningReverse                   = 0x15
	LindsayStatePositionError                    = 0x21
	LindsayStateRunningNoPos                     = 0x29
)

func (ls LindsayState) String() (ret string) {
	switch ls {
	case LindsayStateStopped:
		ret = "Stopped"
	case LindsayStateFault:
		ret = "Safety Fault"
	case LindsayStateServiceStop:
		ret = "Service Stop"
	case LindsayStateSoftBarrierStopped:
		ret = "Soft Barrier Stopped"
	case LindsayStateLowPressureShutdown:
		ret = "Low Pressure Shutdown"
	case LindsayStateLowVoltageFault:
		ret = "Low Voltage Shutdown"
	case LindsayStateRestartDelay:
		ret = "Restart Delay"
	case LindsayStatePressureWaiting:
		ret = "Pressure Waiting"
	case LindsayStateRunningForward:
		ret = "Running Forward"
	case LindsayStateRunningReverse:
		ret = "Running Reverse"
	case LindsayStatePositionError:
		ret = "Position Error"
	case LindsayStateRunningNoPos:
		ret = "Running No Position"
	default:
		ret = "Unknown"
	}

	return
}

// LindsayStatusRegs represents modbus regs for the status messages
type LindsayStatusRegs struct {
	PosWithOffset    uint16       `json:"posWithOffset"`
	PosWithoutOffset uint16       `json:"posWithoutOffset"`
	Status           uint16       `json:"status"`
	Rate             uint16       `json:"rate"`
	Pressure         uint16       `json:"pressure"`
	State            LindsayState `json:"state"`
}

func (lsr LindsayStatusRegs) String() string {
	ret := "Lindsay status:\n"
	ret += fmt.Sprintf("Pos with offset: %v\n", lsr.PosWithOffset)
	ret += fmt.Sprintf("Pos without offset: %v\n", lsr.PosWithoutOffset)
	ret += fmt.Sprintf("Status: 0x%x\n", lsr.Status)
	ret += fmt.Sprintf("Rate: %v\n", lsr.Rate)
	ret += fmt.Sprintf("Pressure: %v\n", lsr.Pressure)
	ret += fmt.Sprintf("State: %v\n", lsr.State.String())
	ret += fmt.Sprintf("State (hex): 0x%x\n", uint16(lsr.State))
	ret += "===========================\n"
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

// Accessory2On indicator, not sure if this is on or not
func (lsr *LindsayStatusRegs) Accessory2On() bool {
	return (lsr.Status & (1 << 6)) != 0
}

// AutoReverse indicator
func (lsr *LindsayStatusRegs) AutoReverse() bool {
	return (lsr.Status & (1 << 8)) != 0
}

// AutoRestart indicator
func (lsr *LindsayStatusRegs) AutoRestart() bool {
	return (lsr.Status & (1 << 9)) != 0
}

// IrrigatorRunning indicates of irrigator is running
func (lsr *LindsayStatusRegs) IrrigatorRunning() bool {
	return (lsr.State == LindsayStateRunningForward ||
		lsr.State == LindsayStateRunningReverse) &&
		(lsr.Forward() || lsr.Reverse())
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
