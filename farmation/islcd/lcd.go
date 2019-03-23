package islcd

import (
	"errors"
	"os"
	"syscall"
	"time"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
)

// Gpios
const pinRegSel string = "PC5"
const pinReset string = "PC8"
const pinPwm string = "PC3"

// Lcd defines a struct used to control a LCD display
type Lcd struct {
	fd         int
	gpioRegSel gpio.PinIO
	gpioReset  gpio.PinIO
	gpioPwm    gpio.PinIO
}

// NewLcd creates a new LCD object and opens the SPI port
func NewLcd() (ret Lcd, err error) {
	ret.fd, err = syscall.Open("/dev/spidev1.0", os.O_WRONLY, 0666)
	if err != nil {
		return
	}

	//Try to grab the gpio for the register address sel and nRST:
	ret.gpioRegSel = gpioreg.ByName(pinRegSel)
	if ret.gpioRegSel == nil {
		err = errors.New("Could not get reg sel gpio")
		return
	}

	ret.gpioReset = gpioreg.ByName(pinReset)
	if ret.gpioReset == nil {
		err = errors.New("Could not get reset gpio")
		return
	}

	ret.gpioPwm = gpioreg.ByName(pinPwm)
	if ret.gpioPwm == nil {
		err = errors.New("Could not get pwm gpio")
		return
	}

	return ret, nil
}

func (l *Lcd) writeLcd(data []byte) error {
	n, err := syscall.Write(l.fd, data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.New("Did not write all bytes to LCD")
	}

	return nil
}

// Init resets and sends init sequence to LCD
func (l *Lcd) Init() error {
	l.gpioReset.Out(gpio.High)
	time.Sleep(10 * time.Millisecond)
	l.gpioReset.Out(gpio.Low)
	time.Sleep(10 * time.Millisecond)
	l.gpioReset.Out(gpio.High)

	l.gpioRegSel.Out(gpio.Low)
	err := l.writeLcd([]byte{0xAE, 0xA5, 0xA2, 0xA1, 0xC0, 0x26, 0x81, 0x1F,
		0xF8, 0x00, 0xAF, 0xA4, 0x2F})
	l.gpioRegSel.Out(gpio.Low)

	if err != nil {
		return err
	}

	return nil
}

func (l *Lcd) Write(data []bool) error {
	if len(data) != 128*64 {
		return errors.New("Must supply 128x64 pixels of data")
	}

	pageSizeBits := 8 * 128

	for page := 0; page < 8; page++ {
		pageData := make([]byte, 128)
		for i := range pageData {
			for b := uint(0); b < 8; b++ {
				var bit byte
				if data[pageSizeBits*page+i+int(b)*128] {
					bit = 1
				}
				pageData[i] |= bit << b
			}
		}
		l.gpioRegSel.Out(gpio.Low)
		err := l.writeLcd([]byte{0xB0 + byte(page), 0x10, 0x0})
		if err != nil {
			return err
		}
		l.gpioRegSel.Out(gpio.High)
		err = l.writeLcd(pageData)
		if err != nil {
			return err
		}
	}
	return nil
}
