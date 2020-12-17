package modbus

import (
	"errors"
	"sync"
)

// Reg defines a Modbus register
type Reg struct {
	Address uint16
	Value   uint16
}

// Regs represents all registers in a modbus device and provides functions
// to read/write 16-bit and bit values. This register module assumes all
// register types map into one address space
// as described in the modbus spec
// (http://www.modbus.org/docs/Modbus_Application_Protocol_V1_1b3.pdf)
// on page 6 and 7.
// All operations on Regs are threadsafe and protected by a mutex.
type Regs struct {
	regs []Reg
	lock sync.RWMutex
}

// AddReg is used to add a modbus register to the server.
// the callback function is called when the reg is updated
// The register can be updated by word or bit operations.
func (r *Regs) AddReg(address uint16) {
	r.lock.Lock()
	defer r.lock.Unlock()
	// first check if reg already exists
	for _, reg := range r.regs {
		if reg.Address == address {
			return
		}
	}
	r.regs = append(r.regs, Reg{address, 0})
}

func (r *Regs) readReg(address uint16) (uint16, error) {
	for _, reg := range r.regs {
		if reg.Address == address {
			return reg.Value, nil
		}
	}

	return 0, errors.New("register not found")
}

// ReadReg is used to read a modbus register
func (r *Regs) ReadReg(address uint16) (uint16, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.readReg(address)
}

func (r *Regs) writeReg(address uint16, value uint16) error {
	for i, reg := range r.regs {
		if reg.Address == address {
			(r.regs)[i].Value = value
			return nil
		}
	}

	return errors.New("register not found")
}

// WriteReg is used to write a modbus register
func (r *Regs) WriteReg(address uint16, value uint16) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.writeReg(address, value)
}

// AddCoil is used to add a discrete io to the register map.
// Note coils are aliased on top of other registers, so coil 20
// would be register 1 bit 4 (16 + 4 = 20).
func (r *Regs) AddCoil(num int) {
	regAddress := uint16(num / 16)
	r.AddReg(regAddress)
}

// ReadCoil gets a coil value (can also be used for discrete inputs)
func (r *Regs) ReadCoil(num int) (bool, error) {
	regAddress := uint16(num / 16)
	regValue, err := r.ReadReg(regAddress)
	if err != nil {
		return false, err
	}

	bitPos := uint16(num % 16)
	ret := (regValue & (1 << bitPos)) != 0
	return ret, nil
}

// WriteCoil writes a coil value
func (r *Regs) WriteCoil(num int, value bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	regAddress := uint16(num / 16)
	regValue, err := r.readReg(regAddress)
	if err != nil {
		return err
	}

	bitPos := uint16(num % 16)

	if value {
		regValue |= 1 << bitPos
	} else {
		regValue &= ^(1 << bitPos)
	}

	return r.writeReg(regAddress, regValue)
}
