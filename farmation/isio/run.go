package isio

import (
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func setRelay(config *isdata.Config) {
	// todo
}

// Run goroutine for IO code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	config := configInit
	//state := stateInit
	StatusLightRed(false)
	StatusLightGreen(false)
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
				setRelay(&config)
			case data.Sample:
				// ... todo
				_ = m
			}
		}
	}
}
