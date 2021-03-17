package modbus

import (
	"reflect"
	"testing"
	"time"
)

func TestTCPEncodeDecode(t *testing.T) {
	pdu := PDU{
		FunctionCode: FuncCodeWriteMultipleCoils,
		Data:         []byte{1, 2, 3},
	}

	tport := NewTCP(nil, 500*time.Millisecond, TransportClient)
	data, err := tport.Encode(1, pdu)

	if err != nil {
		t.Fail()
	}

	_, pdu2, err := tport.Decode(data)

	if err != nil {
		t.Fail()
	}

	if pdu2.FunctionCode != pdu.FunctionCode {
		t.Error("Function code not the same")
	}

	if !reflect.DeepEqual(pdu2.Data, pdu.Data) {
		t.Error("Data compare failed")
	}
}
