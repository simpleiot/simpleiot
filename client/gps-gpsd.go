package client

// runGpsd reads position data from the gpsd daemon. Implemented in a later
// phase.
func (gc *GPSClient) runGpsd(config GPS, src *gpsSource) {
	gc.log.Printf("%v: gpsd source not implemented yet", config.Description)
	<-src.stop
}
