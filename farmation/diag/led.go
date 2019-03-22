package diag

import (
	"errors"
	"fmt"
	"time"
)

type heartbeat struct{}

func (d heartbeat) Run() error {
	fmt.Println("Is there a heartbeat pattern on hall and CPU leds (y/n)?")
	if !GetInput() {
		return errors.New("heartbeat led test failed")
	}
	return nil
}

func (d heartbeat) String() string {
	return "led-heartbeat"
}

type misc struct{}

func blinkMisc(stop chan bool) {
	state := false
	tick := time.NewTicker(time.Millisecond * 50)
	for {
		select {
		case <-tick.C:
			state = !state
			//gio.SetLedMisc(state)
		case <-stop:
			tick.Stop()
			//gio.SetLedMisc(false)
			return
		}
	}
}

func (d misc) Run() error {
	c := make(chan bool, 1)
	go blinkMisc(c)
	fmt.Println("Is misc LED blinking (y/n)?")
	r := GetInput()
	c <- false
	//gio.SetLedMisc(false)
	if !r {
		return errors.New("Misc led failure")
	}
	return nil
}

func (d misc) String() string {
	return "led-misc"
}

type port struct{}

func blinkPort(stop chan bool) {
	last := 1
	cur := 1
	up := true
	tick := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-tick.C:
			last = cur
			if up {
				cur++
				if cur > 6 {
					cur = 5
					up = false
				}
			} else {
				cur--
				if cur < 1 {
					cur = 2
					up = true
				}
			}

			_ = last

			//gio.SetLedPort(last, false)
			//gio.SetLedPort(cur, true)

		case <-stop:
			tick.Stop()
			return
		}
	}
}

func (d port) Run() error {
	c := make(chan bool, 1)
	go blinkPort(c)
	fmt.Println("Is port LED pattern complete (y/n)?")
	r := GetInput()
	c <- false
	for p := 1; p < 7; p++ {
		//gio.SetLedPort(p, false)
	}
	if !r {
		return errors.New("Port led failure")
	}
	return nil
}

func (d port) String() string {
	return "led-port"
}

func init() {
	var heartbeatDiag heartbeat
	Register(heartbeatDiag)
	var miscDiag misc
	Register(miscDiag)
	var portDiag port
	Register(portDiag)
}
