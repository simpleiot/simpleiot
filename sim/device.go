package sim

import (
	"log"
	"time"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/data"
)

func packetDelay() {
	time.Sleep(5 * time.Second)
}

// DeviceSim simulates a simple device
func DeviceSim(portal, deviceID string) {
	log.Printf("starting simulator: ID: %v, portal: %v\n", deviceID, portal)

	sendSamples := api.NewSendSamples(portal)
	tempSim := NewSim(72, 0.2, 70, 75)
	voltSim := NewSim(2, 0.1, 1, 5)

	for {
		samples := make([]data.Sample, 2)
		samples[0] = data.Sample{
			ID:    "T0",
			Type:  "temp",
			Value: tempSim.Sim(),
		}

		samples[1] = data.Sample{
			ID:    "V0",
			Type:  "volt",
			Value: voltSim.Sim(),
		}

		err := sendSamples(deviceID, samples)
		if err != nil {
			log.Println("Error sending samples: ", err)
		}
		packetDelay()
	}
}
