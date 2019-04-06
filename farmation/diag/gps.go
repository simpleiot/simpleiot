package diag

import (
	"errors"
	"fmt"
	"time"

	nmea "github.com/adrianmo/go-nmea"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/gps"
)

type gpsDiag struct{}

func (d gpsDiag) Run() error {
	c := make(chan nmea.Sentence)
	g := gps.NewGps(isio.SerialGps, 9600, c)
	timeout := time.NewTimer(5 * time.Second)
	g.Start()
	for {
		select {
		case m := <-c:
			fmt.Printf("GPS: %+v\n", m)
		case <-timeout.C:
			g.Stop()
			return errors.New("No data from GPS")
		}
		g.Stop()
		break
	}

	return nil
}

func (d gpsDiag) String() string {
	return "gps"
}

func init() {
	Register(gpsDiag{})
}
