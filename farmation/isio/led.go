package isio

// StatusLightRed controls red status LED
func StatusLightRed(on bool) {
	GpioOut(GpioStatusRed, on)
}

// StatusLightGreen controls green status LED
func StatusLightGreen(on bool) {
	GpioOut(GpioStatusGreen, on)
}
