package client

// ModbusIo describes a modbus IO, which is always a child of a Modbus node.
// An IO cannot act on its own -- it shares the port, the register map, and the
// poll cycle with its bus -- so it is managed by the ModbusClient rather than
// being a client of its own.
type ModbusIo struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	// ServerID is the modbus server (unit) address, which is unrelated to the
	// SIOT node ID above.
	ServerID           int     `point:"id"`
	Address            int     `point:"address"`
	ModbusIOType       string  `point:"modbusIoType"`
	DataFormat         string  `point:"dataFormat"`
	ReadOnly           bool    `point:"readOnly"`
	Scale              float64 `point:"scale"`
	Offset             float64 `point:"offset"`
	Value              float64 `point:"value"`
	ValueSet           float64 `point:"valueSet"`
	Disabled           bool    `point:"disabled"`
	ErrorCount         int     `point:"errorCount"`
	ErrorCountCRC      int     `point:"errorCountCRC"`
	ErrorCountEOF      int     `point:"errorCountEOF"`
	ErrorCountReset    bool    `point:"errorCountReset"`
	ErrorCountCRCReset bool    `point:"errorCountCRCReset"`
	ErrorCountEOFReset bool    `point:"errorCountEOFReset"`
}

// scaleFactor returns the scale to apply to register values. A scale of zero
// would read every register as zero, which is never what is wanted, so an
// unset scale is treated as one.
func (io ModbusIo) scaleFactor() float64 {
	if io.Scale == 0 {
		return 1
	}
	return io.Scale
}
