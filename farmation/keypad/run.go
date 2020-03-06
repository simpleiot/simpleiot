package keypad

import (
	"log"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
)

func keypad(out chan interface{}, name string, keyPress, keyHold, keyRelease isdata.Key) {
	p := gpioreg.ByName(name)
	//var lastSent time.Time
	if p == nil {
		log.Println("got nil when requesting GPIO ", name)
		return
	}

	// Set it as input, with an internal pull down resistor:
	if err := p.In(gpio.Float, gpio.BothEdges); err != nil {
		log.Println("Error setting up gpio: ", err)
		return
	}
	// make channel to send edges from button presses
	cEdge := make(chan bool)

	keyStateDown := false

	// timer for press-and-hold
	pressTimerDur := time.Millisecond * 500
	pressTimer := time.NewTimer(pressTimerDur)
	pressTimer.Stop()

	// timer for use in debouncing
	timer := time.NewTimer(time.Millisecond * 50)
	timer.Stop()

	// ticker for scrolling with arrow keys
	scrollTicker := time.NewTicker(time.Millisecond * 150)

	// tells us if the pressTimer has expired -- if so, start
	// scrolling while the arrow key is held down
	var scrolling bool

	var timerRunning bool
	go func() {
		for {
			p.WaitForEdge(-1) //one second?
			cEdge <- true
		}
	}()

	for {
		select {
		case <-cEdge:
			if timerRunning == false {
				// If the key was pressed and is released
				if keyStateDown && p.Read() == gpio.High {
					out <- keyRelease
					pressTimer.Stop()
					scrolling = false
					keyStateDown = false
					// If the key was unpressed and is pressed
				} else if !keyStateDown && p.Read() == gpio.Low {
					out <- keyPress
					pressTimer.Reset(pressTimerDur)
					keyStateDown = true
				}
				timer.Reset(time.Millisecond * 50)
				timerRunning = true
			}
			// if the time has expired, send out the keyHold
		case <-pressTimer.C:
			if keyStateDown && p.Read() == gpio.Low {
				out <- keyHold
				pressTimer.Stop()
				scrolling = true
			}
		case <-scrollTicker.C:
			if scrolling &&
				keyStateDown &&
				p.Read() == gpio.Low {
				out <- keyHold
			}
		case <-timer.C:
			timerRunning = false
		}
		/*
			curr := time.Now()
			if curr.Sub(lastSent) > time.Millisecond*50 {
				out <- key
				lastSent = curr
			}
		*/
	}
}

/*
// Get character as a byte slice
func getch() []byte {
	t, _ := term.Open("/dev/tty")
	term.RawMode(t)
	bytes := make([]byte, 3)
	numRead, err := t.Read(bytes)
	t.Restore()
	t.Close()
	if err != nil {
		return nil
	}
	return bytes[0:numRead]
}
*/

/* map of pins on keyswitch
 * 1: SK4 (PA17)
 * 2: SK3 (PA15)
 * 3: SK2 (PA19)
 * 4: SK1 (PA18)
 * 5: left (PA20)
 * 6: up (PA21)
 * 7: enter (PD10)
 * 8: down (PA12)
 * 9: right (PA11)
 * 10: arm  (PC25)
 *
 * switches are active low
 */

// Run goroutine for keypad code
func Run(in, out chan interface{}, hwVersion int) {
	if runtime.GOARCH == "arm" && hwVersion != 1 {
		go keypad(out, "PA17", isdata.KeySK4, isdata.KeySK4Hold, isdata.KeySK4Release)
		go keypad(out, "PA15", isdata.KeySK3, isdata.KeySK3Hold, isdata.KeySK3Release)
		go keypad(out, "PA19", isdata.KeySK2, isdata.KeySK2Hold, isdata.KeySK2Release)
		go keypad(out, "PA18", isdata.KeySK1, isdata.KeySK1Hold, isdata.KeySK1Release)
		go keypad(out, "PA20", isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyLeftRelease)
		go keypad(out, "PA21", isdata.KeyUp, isdata.KeyUpHold, isdata.KeyUpRelease)
		go keypad(out, "PD10", isdata.KeyEnter, isdata.KeyEnterHold, isdata.KeyEnterRelease)
		go keypad(out, "PA12", isdata.KeyDown, isdata.KeyDownHold, isdata.KeyDownRelease)
		go keypad(out, "PA11", isdata.KeyRight, isdata.KeyRightHold, isdata.KeyRightRelease)
		go keypad(out, "PA16", isdata.KeyArmKp, isdata.KeyArmKpHold, isdata.KeyArmKpRelease)
		go keypad(out, "PA14", isdata.KeyPump, isdata.KeyPumpHold, isdata.KeyPumpRelease)

		// wired to discrete switch
		go keypad(out, "PC25", isdata.KeyArm, isdata.KeyArmHold, isdata.KeyArmRelease)
	}

	if runtime.GOARCH == "arm" && hwVersion == 1 {
		go keypad(out, "PA19", isdata.KeySK4, isdata.KeySK4Hold, isdata.KeySK4Release)
		go keypad(out, "PA18", isdata.KeySK3, isdata.KeySK3Hold, isdata.KeySK3Release)
		go keypad(out, "PA20", isdata.KeySK2, isdata.KeySK2Hold, isdata.KeySK2Release)
		go keypad(out, "PA21", isdata.KeySK1, isdata.KeySK1Hold, isdata.KeySK1Release)
		go keypad(out, "PD10", isdata.KeyLeft, isdata.KeyLeftHold, isdata.KeyLeftRelease)
		go keypad(out, "PA12", isdata.KeyUp, isdata.KeyUpHold, isdata.KeyUpRelease)
		go keypad(out, "PA11", isdata.KeyEnter, isdata.KeyEnterHold, isdata.KeyEnterRelease)
		go keypad(out, "PA16", isdata.KeyDown, isdata.KeyDownHold, isdata.KeyDownRelease)
		go keypad(out, "PA14", isdata.KeyRight, isdata.KeyRightHold, isdata.KeyRightRelease)
		go keypad(out, "PA17", isdata.KeyArmKp, isdata.KeyArmKpHold, isdata.KeyArmKpRelease)
		go keypad(out, "PA15", isdata.KeyPump, isdata.KeyPumpHold, isdata.KeyPumpRelease)

		// wired to discrete switch
		go keypad(out, "PC25", isdata.KeyArm, isdata.KeyArmHold, isdata.KeyArmRelease)
	}

	/*
		go func() {
			for {
				c := getch()
				switch {
				case bytes.Equal(c, []byte{3}):
					return
				case bytes.Equal(c, []byte{27, 91, 65}): // up arrow
					out <- isdata.KeyUp
				case bytes.Equal(c, []byte{27, 91, 66}): // down
					out <- isdata.KeyDown
				case bytes.Equal(c, []byte{27, 91, 67}): // right
					out <- isdata.KeyRight
				case bytes.Equal(c, []byte{27, 91, 68}): // left
					out <- isdata.KeyLeft
				case bytes.Equal(c, []byte{13}): // enter
					out <- isdata.KeyEnter
				case bytes.Equal(c, []byte{49}): // 1
					out <- isdata.KeySK1
				case bytes.Equal(c, []byte{50}): // 2
					out <- isdata.KeySK2
				case bytes.Equal(c, []byte{51}): // 3
					out <- isdata.KeySK3
				case bytes.Equal(c, []byte{52}): // 4
					out <- isdata.KeySK4
				}
			}
		}()
	*/
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		}
	}
}
