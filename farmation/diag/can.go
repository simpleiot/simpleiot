package diag

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/simpleiot/simpleiot/farmation/isio"
)

type can struct{}

func (d can) String() string {
	return "can"
}

func (d can) Run() (ret error) {
	isio.GpioOut(isio.GpioCanStby, false)
	iface, err := net.InterfaceByName("can0")
	if err != nil {
		return errors.Wrap(err, "Internal CAN bus not found")
	}

	if iface.Flags&net.FlagUp == 0 {
		// bring up can0 interface
		err = exec.Command("ip", "link", "set", "can0", "type",
			"can", "bitrate", "250000").Run()
		if err != nil {
			return errors.Wrap(err, "Error configuring internal can iface")
		}

		err = exec.Command("ifconfig", "can0", "up").Run()
		if err != nil {
			return errors.Wrap(err, "Error bringing up internal can iface")
		}
	}

	iface, err = net.InterfaceByName("can1")
	if err != nil {
		// install slcan
		err = exec.Command("slcand", "-o", "-c", "-s5", "/dev/ttyACM0", "can1").Run()
		if err != nil {
			return errors.Wrap(err, "Error starting slcand on USB can")
		}
	}

	iface, err = net.InterfaceByName("can1")
	if err != nil {
		return errors.Wrap(err, "USB can interface not present")
	}

	if iface.Flags&net.FlagUp == 0 {
		// bring up can0 interface
		err = exec.Command("ifconfig", "can1", "up").Run()
		if err != nil {
			return errors.Wrap(err, "Error bringing up USB can interface")
		}

		err = exec.Command("ifconfig", "can0", "txqueuelen", "1000").Run()
		if err != nil {
			return errors.Wrap(err, "Error setting queue len on USB can interface")
		}
	}

	cmdCanDump := exec.Command("candump", "can0")
	stdout, err := cmdCanDump.StdoutPipe()
	go cmdCanDump.Start()

	time.Sleep(50 * time.Millisecond)

	testString := "123#0123456789abcdef"

	err = exec.Command("cansend", "can1", testString).Run()

	if err != nil {
		return errors.Wrap(err, "Error sending data to USB can interface")
	}
	time.Sleep(50 * time.Millisecond)

	buf := make([]byte, 50)
	n, err := stdout.Read(buf)

	buf = buf[:n]

	if err != nil {
		return errors.Wrap(err, "Error reading candump output")
	}

	if !strings.Contains(string(buf), "01 23 45 67 89 AB CD EF") {
		return fmt.Errorf("Did not receive CAN loopback test packet\nexpected: %v\nreceived: %v", testString, string(buf))
	}

	return nil
}

func init() {
	Register(can{})
}
