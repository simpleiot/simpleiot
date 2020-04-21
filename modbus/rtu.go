package modbus

// RtuPDU defines a PDU for RTU packets
type RtuPDU struct {
	Address      byte
	FunctionCode FunctionCode
	Data         []byte
	CRC          byte
}
