package client

// ZMini represents a Zonit mini client
type ZMini struct {
	SerialPort []SerialDev `child:"serialDev"`
}
