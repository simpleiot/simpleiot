package islcd

import (
	"log"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for lcd code
func Run(in, out chan interface{}) {
	lcdOK := false
	lcd, err := NewLcd()
	if err != nil {
		log.Println("Error opening LCD", err)
	} else {
		err := lcd.Init()
		if err != nil {
			log.Println("Error initializing LCD: ", err)
		} else {
			lcdOK = true
		}
	}
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.LcdBlt:
				if lcdOK {
					lcd.Write(m.Data)
				}

			default:
				log.Printf("islcd: unhandled message of type %T: %+v\r\n", m, m)
			}
		}
	}
}
