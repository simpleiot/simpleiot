package network

import (
	"bytes"
	"testing"
)

// looking for: +CSQ: 9,99
func TestGetSignal(t *testing.T) {
	buf := bytes.NewBufferString("+CSQ: 9,99")

	sig, biterror, err := CmdGetSignal(buf)
	if err != nil {
		t.Fatal("Error: ", err)
	}

	if sig != 29 {
		t.Fatal("Error, signal is: ", sig)
	}

	if biterror != -1 {
		t.Fatal("Error biterror is: ", biterror)
	}
}

func TestGetApn(t *testing.T) {

	resp :=
		`+CGDCONT: 1,"IPV4V6","","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0
+CGDCONT: 2,"IPV4V6","vzwadmin","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0
+CGDCONT: 3,"IPV4V6","vzwinternet","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0
+CGDCONT: 4,"IPV4V6","vzwapp","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0
+CGDCONT: 5,"IPV4V6","vzwclass6","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0
+CGDCONT: 6,"IPV4V6","vzwiotts","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0

OK
`

	buf := bytes.NewBuffer([]byte(resp))

	apn, err := CmdGetApn(buf)
	if err != nil {
		t.Fatal("Error: ", err)
	}

	if apn != "vzwinternet" {
		t.Fatal("Apn is: ", apn)
	}

}

func TestGetUsbCfg(t *testing.T) {
	buf := bytes.NewBufferString("#USBCFG: 3")

	cfg, err := CmdGetUsbCfg(buf)
	if err != nil {
		t.Fatal("Error: ", err)
	}

	if cfg != 3 {
		t.Fatal("Cfg is: ", cfg)
	}
}

func TestGetFwSwitch(t *testing.T) {
	buf := bytes.NewBufferString("#FWSWITCH: 1")

	fw, err := CmdGetFwSwitch(buf)
	if err != nil {
		t.Fatal("Error: ", err)
	}

	if fw != 1 {
		t.Fatal("fw is: ", fw)
	}
}

func TestGetGpio(t *testing.T) {
	buf := bytes.NewBufferString("#GPIO: 1,1,4")

	dir, level, err := CmdGetGpio(buf, 10)
	if err != nil {
		t.Fatal("Error: ", err)
	}

	if dir != GpioOut {
		t.Fatal("dir is: ", dir)
	}

	if level != GpioHigh {
		t.Fatal("level is ", level)
	}
}
