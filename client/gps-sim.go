package client

// runSim generates a simulated GPS track. Implemented in a later phase.
func (gc *GPSClient) runSim(config GPS, stop chan struct{}) {
	gc.log.Printf("%v: simulation source not implemented yet", config.Description)
	<-stop
}
