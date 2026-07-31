package client

// runSerial reads NMEA sentences from a serial port. Implemented in a later
// phase.
func (gc *GPSClient) runSerial(config GPS, src *gpsSource) {
	gc.log.Printf("%v: serial source not implemented yet", config.Description)
	<-src.stop
}
