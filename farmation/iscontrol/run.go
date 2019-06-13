package iscontrol

import (
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Run goroutine for ui code
func Run(in, out chan interface{}, configInit isdata.Config, stateInit isdata.State) {
	//config := configInit
	//state := stateInit

	controlTicker := time.NewTicker(1000 * time.Millisecond)
	for {
		select {
		case <-controlTicker.C:
		}
	}
}
