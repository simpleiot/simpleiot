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

	sendSamples := api.NewSendSamples(portal, false)
	tempSim := NewSim(72, 0.2, 70, 75)
	voltSim := NewSim(2, 0.1, 1, 5)
	voltSim2 := NewSim(5, 0.5, 1, 10)

	for {
		samples := make([]data.Sample, 3)
		samples[0] = data.Sample{
			Type:  "temp",
			Value: tempSim.Sim(),
		}

		samples[1] = data.Sample{
			ID:    "V0",
			Type:  "volt",
			Value: voltSim.Sim(),
		}

		samples[2] = data.Sample{
			ID:    "V1",
			Type:  "volt",
			Value: voltSim2.Sim(),
		}

		err := sendSamples(deviceID, samples)
		if err != nil {
			log.Println("Error sending samples: ", err)
		}
		packetDelay()
	}
}
