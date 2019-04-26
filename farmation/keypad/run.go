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

func keypad(out chan interface{}, name string, key isdata.Key) {
	p := gpioreg.ByName(name)
	//var lastSent time.Time
	if p == nil {
		log.Println("got nil when requesting GPIO ", name)
		return
	}

	// Set it as input, with an internal pull down resistor:
	if err := p.In(gpio.Float, gpio.FallingEdge); err != nil {
		log.Println("Error setting up gpio: ", err)
		return
	}
	// make channel to send edges from button presses
	cEdge := make(chan bool)

	// timer for use in debouncing
	timer := time.NewTimer(time.Millisecond * 50)
	timer.Stop()

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
				//fmt.Println("got edge", p.Read())
				if p.Read() == false {
					out <- key
				}
				timer.Reset(time.Millisecond * 50)
				timerRunning = true
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
 *
 * switches are active low
 */

// Run goroutine for keypad code
func Run(in, out chan interface{}) {

	if runtime.GOARCH == "arm" {
		go keypad(out, "PA17", isdata.KeySK4)
		go keypad(out, "PA15", isdata.KeySK3)
		go keypad(out, "PA19", isdata.KeySK2)
		go keypad(out, "PA18", isdata.KeySK1)
		go keypad(out, "PA20", isdata.KeyLeft)
		go keypad(out, "PA21", isdata.KeyUp)
		go keypad(out, "PD10", isdata.KeyEnter)
		go keypad(out, "PA12", isdata.KeyDown)
		go keypad(out, "PA11", isdata.KeyRight)
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
