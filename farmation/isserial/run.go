package isserial

import (
	"log"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for IO code
func Run(in, out chan interface{}, configInit isdata.Config) {
	config := configInit
	_ = config
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
			default:
				log.Printf("isflow mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}
