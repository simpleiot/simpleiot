package diag

import (
	"errors"
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/gps"
)

type gpsDiag struct{}

func (d gpsDiag) Run() error {
	// first test with GPS in reset
	isio.GpioOut(isio.GpioGpsReset, true)
	c := make(chan data.GpsPos)
	g := gps.NewGps(isio.SerialGps, 9600, c)
	timeout := time.NewTimer(3 * time.Second)
	g.Start()
	select {
	case m := <-c:
		fmt.Printf("GPS: %+v\n", m)
		g.Stop()
		return errors.New("Got GPS data while in reset")
	case <-timeout.C:
	}

	// now test GPS not in reset
	isio.GpioOut(isio.GpioGpsReset, false)
	timeout = time.NewTimer(5 * time.Second)
	select {
	case m := <-c:
		fmt.Printf("GPS: %+v\n", m)
	case <-timeout.C:
		g.Stop()
		return errors.New("Timeout waiting for GPS data")

	}

	g.Stop()

	return nil
}

func (d gpsDiag) String() string {
	return "gps"
}

func init() {
	//Register(gpsDiag{})
}
