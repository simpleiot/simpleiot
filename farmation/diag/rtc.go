package diag

import (
	"github.com/simpleiot/simpleiot/farmation/isnetwork"
)

type rtc struct{}

func (d rtc) String() string {
	return "rtc"
}

func (d rtc) Run() error {
	return isnetwork.UpdateTimeFromNetwork()
}

func init() {
	Register(rtc{})
}
