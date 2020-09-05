package particle

import (
	"encoding/json"
	"log"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/simpleiot/simpleiot/data"
)

// Event from particle
type Event struct {
	Data      string    `json:"data"`
	TTL       uint32    `json:"ttl"`
	Timestamp time.Time `json:"published_at"`
	CoreID    string    `json:"coreid"`
}

const url string = "https://api.particle.io/v1/devices/events/"

// PointReader does a streaming http read and returns when the connection closes
func PointReader(eventPrefix, token string, callback func(string, data.Points)) error {
	urlAuth := url + eventPrefix + "?access_token=" + token

	stream, err := eventsource.Subscribe(urlAuth, "")

	if err != nil {
		return err
	}

	for {
		select {
		case event := <-stream.Events:
			var pEvent Event
			err := json.Unmarshal([]byte(event.Data()), &pEvent)
			if err != nil {
				log.Println("Got error decoding particle event: ", err)
				continue
			}

			var points []data.Point
			err = json.Unmarshal([]byte(pEvent.Data), &points)
			if err != nil {
				log.Println("Got error decoding samples: ", err)
				continue
			}

			callback(pEvent.CoreID, points)

		case err := <-stream.Errors:
			log.Println("Got error: ", err)
		}
	}
}
