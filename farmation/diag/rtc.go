package diag

import (
	"github.com/simpleiot/simpleiot/system"
)

type rtc struct{}

func (d rtc) String() string {
	return "rtc"
}

func (d rtc) Run() error {
	return system.UpdateTimeFromNetwork()
}

func init() {
	Register(rtc{})
}
