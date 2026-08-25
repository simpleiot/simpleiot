//go:build !linux

package client

import (
	"fmt"
	"runtime"
)

// gpioCdevRequest reports that the GPIO character device is not available.
// Only Linux offers /dev/gpiochipN; the simulated chip builds everywhere, so
// the client and its tests still run on every platform.
func gpioCdevRequest(_ gpioLineConfig) (gpioLine, gpioLineInfo, error) {
	return nil, gpioLineInfo{}, fmt.Errorf(
		"GPIO character device is not supported on %v", runtime.GOOS)
}
