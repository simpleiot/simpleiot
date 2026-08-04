package client

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/modbus"
)

// These tests cover the conversion between an IO value and the register
// contents on their own, with no server and no wire. If they pass while the
// end to end tests fail, the problem is in the plumbing rather than the
// arithmetic.

func regTestClient() *ModbusClient {
	return &ModbusClient{
		regs:     &modbus.Regs{},
		lastSent: make(map[string]time.Time),
	}
}

func TestModbusRegRoundTrip(t *testing.T) {
	tests := []struct {
		desc    string
		format  string
		address int
		scale   float64
		offset  float64
		value   float64
	}{
		{"uint16 top of range", data.PointValueUINT16, 0, 1, 0, 65535},
		{"uint16 zero", data.PointValueUINT16, 1, 1, 0, 0},
		{"int16 negative", data.PointValueINT16, 2, 1, 0, -1234},
		{"int16 positive", data.PointValueINT16, 3, 1, 0, 1234},
		{"uint32 above int32", data.PointValueUINT32, 4, 1, 0, 4000000000},
		{"int32 negative", data.PointValueINT32, 6, 1, 0, -2000000},
		{"float32", data.PointValueFLOAT32, 8, 1, 0, 3.25},
		{"scaled and offset", data.PointValueUINT16, 10, 0.1, -40, 25},
		{"scale unset is one", data.PointValueUINT16, 11, 0, 0, 300},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			c := regTestClient()

			io := ModbusIo{
				ID:           "io",
				Address:      tc.address,
				ModbusIOType: data.PointValueModbusHoldingRegister,
				DataFormat:   tc.format,
				Scale:        tc.scale,
				Offset:       tc.offset,
				Value:        tc.value,
			}

			c.regs.AddReg(io.Address, modbusRegCount(io.DataFormat))

			if err := c.WriteReg(&io); err != nil {
				t.Fatal("WriteReg error: ", err)
			}

			v, err := c.ReadReg(&io)
			if err != nil {
				t.Fatal("ReadReg error: ", err)
			}

			if v != tc.value {
				t.Fatalf("round trip changed value: wrote %v, read %v", tc.value, v)
			}
		})
	}
}

// TestModbusRegScaling checks the raw register contents, since a scale and
// offset that cancel out in a round trip would still be wrong on the wire.
func TestModbusRegScaling(t *testing.T) {
	c := regTestClient()

	io := ModbusIo{
		ID:           "io",
		Address:      0,
		ModbusIOType: data.PointValueModbusHoldingRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        0.1,
		Offset:       -40,
		Value:        25,
	}

	c.regs.AddReg(io.Address, 1)

	if err := c.WriteReg(&io); err != nil {
		t.Fatal("WriteReg error: ", err)
	}

	raw, err := c.regs.ReadReg(io.Address)
	if err != nil {
		t.Fatal("ReadReg error: ", err)
	}

	// (25 - -40) / 0.1
	if raw != 650 {
		t.Fatalf("expected raw register 650, got %v", raw)
	}
}

func TestModbusRegCount(t *testing.T) {
	tests := []struct {
		format string
		count  int
	}{
		{data.PointValueUINT16, 1},
		{data.PointValueINT16, 1},
		{data.PointValueUINT32, 2},
		{data.PointValueINT32, 2},
		{data.PointValueFLOAT32, 2},
	}

	for _, tc := range tests {
		if c := modbusRegCount(tc.format); c != tc.count {
			t.Errorf("%v: expected %v registers, got %v", tc.format, tc.count, c)
		}
	}
}

// TestModbusValidate checks that an incomplete configuration is reported
// rather than silently opening a port with defaults.
func TestModbusValidate(t *testing.T) {
	tests := []struct {
		desc  string
		cfg   Modbus
		valid bool
	}{
		{"no client/server", Modbus{Protocol: data.PointValueTCP}, false},
		{"no protocol", Modbus{ClientServer: data.PointValueServer}, false},
		{"RTU with no port", Modbus{
			ClientServer: data.PointValueServer,
			Protocol:     data.PointValueRTU,
		}, false},
		{"RTU with no baud", Modbus{
			ClientServer: data.PointValueServer,
			Protocol:     data.PointValueRTU,
			Port:         "/dev/ttyUSB0",
		}, false},
		{"TCP client with no URI", Modbus{
			ClientServer: data.PointValueClient,
			Protocol:     data.PointValueTCP,
			PollPeriod:   100,
		}, false},
		{"TCP client with no poll period", Modbus{
			ClientServer: data.PointValueClient,
			Protocol:     data.PointValueTCP,
			URI:          "127.0.0.1:502",
		}, false},
		{"TCP server with no port", Modbus{
			ClientServer: data.PointValueServer,
			Protocol:     data.PointValueTCP,
		}, false},
		{"complete RTU server", Modbus{
			ClientServer: data.PointValueServer,
			Protocol:     data.PointValueRTU,
			Port:         "/dev/ttyUSB0",
			Baud:         "9600",
		}, true},
		{"complete TCP client", Modbus{
			ClientServer: data.PointValueClient,
			Protocol:     data.PointValueTCP,
			URI:          "127.0.0.1:502",
			PollPeriod:   100,
		}, true},
		{"complete TCP server", Modbus{
			ClientServer: data.PointValueServer,
			Protocol:     data.PointValueTCP,
			Port:         "502",
		}, true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			c := &ModbusClient{config: tc.cfg}
			problem := c.validate()
			if tc.valid && problem != "" {
				t.Fatalf("expected a valid config, got %q", problem)
			}
			if !tc.valid && problem == "" {
				t.Fatal("expected the config to be reported as incomplete")
			}
		})
	}
}
