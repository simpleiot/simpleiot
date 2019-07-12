package islcd

import (
	"periph.io/x/periph/conn/gpio"
)

// Lcd defines a struct used to control a LCD display
type Lcd struct {
	fd      int
	gpioPwm gpio.PinIO
}

// NewLcd creates a new LCD object and opens the SPI port
func NewLcd() (ret Lcd, err error) {
	return ret, nil
}

func (l *Lcd) writeLcd(data []byte) error {
	return nil
}

// Init resets and sends init sequence to LCD
func (l *Lcd) Init() error {
	return nil
}

func (l *Lcd) Write(data []bool) error {
	return nil
}
