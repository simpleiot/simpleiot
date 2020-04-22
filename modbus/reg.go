package modbus

import "errors"

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
type Regs []Reg

// AddReg is used to add a modbus register to the server.
// the callback function is called when the reg is updated
// The register can be updated by word or bit operations.
func (r *Regs) AddReg(address uint16) {
	*r = append(*r, Reg{address, 0})
}

// ReadReg is used to read a modbus register
func (r *Regs) ReadReg(address uint16) (uint16, error) {
	for _, reg := range *r {
		if reg.Address == address {
			return reg.Value, nil
		}
	}

	return 0, errors.New("register not found")
}

// WriteReg is used to write a modbus register
func (r *Regs) WriteReg(address uint16, value uint16) error {
	for i, reg := range *r {
		if reg.Address == address {
			(*r)[i].Value = value
			return nil
		}
	}

	return errors.New("register not found")
}
