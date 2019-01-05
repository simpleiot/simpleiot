/* This is just some test code to test out the feasibility of using go's waitForEdge()
   functionality in the gpio package.  Tests were made initially on an iMX6 quad core
   processor and found that operation was reliable down to 500uS per edge which is
   much faster than any edges from the flow meters.  However if the kernel gets
   busy some pulses apparently get dropped as even though the ports are edge driven
   the edges do not queue up so if waitForEdge() goes away for too long it's
   possible to miss edges.  For accuracy's sake it's probably not a good idea
   to use waitForEdge() for flow metering but it's eminently suitable for
   keyboard operation.  But at the slower pulse rates from some meters it might
   be acceptable to lose a pulse once in a while.
*/

package main

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
	"periph.io/x/periph/host/sysfs"
)

// gpio 43 on Variscite module

func flow(interval time.Duration, gpionum string, c chan int) {
	count := 0
	// Lookup a pin by its number:
	/*
		p := gpioreg.ByName(gpionum)
		if p == nil {
			log.Fatal("gpioreg.ByName returned nil")
		}
	*/

	p := &sysfs.Pin{
		number: 0,
		name:   "PD25",
		root:   "/sys/class/gpio/PD25",
	}
	if err := gpioreg.Register(p); err != nil {
		log.Fatal("error registering gpio: ", err)
	}

	fmt.Printf("%s: %s\n", p, p.Function())

	// Set it as input, with an internal pull down resistor:
	if err := p.In(gpio.Float, gpio.FallingEdge); err != nil {
		log.Fatal("p.In error: ", err)
	}
	go func() {
		for {
			p.WaitForEdge(-1) //one second?
			count++
		}
	}()
	for {
		time.Sleep(interval * time.Millisecond)
		c <- count
		count = 0
	}
}

func main() {
	// Load all the drivers:
	if _, err := host.Init(); err != nil {
		log.Fatal("error initializing host: ", err)
	}

	ch := make(chan int, 10)

	go flow(1000, "PD25", ch)

	for {
		fmt.Printf("Samples: %d\n", <-ch)
	}
}
