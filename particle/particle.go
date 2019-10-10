package particle

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// Event from particle
type Event struct {
	Name string
	Data struct {
		Data      string    `json:"data"`
		TTL       uint32    `json:"ttl"`
		Timestamp time.Time `json:"published_at"`
		CoreID    string    `json:"coreid"`
	}
}

const url string = "https://api.particle.io/v1/devices/events/"

// SampleReader does a streaming http read and returns when the connection closes
func SampleReader(eventPrefix, token string, callback func([]byte)) error {
	var client = &http.Client{
		Timeout: 0,
	}

	resp, err := client.Get(url + eventPrefix + "?access_token=" + token)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errstring := "Server error: " + resp.Status
		body, _ := ioutil.ReadAll(resp.Body)
		errstring += " " + string(body)
		return errors.New(errstring)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		var line []byte
		line, _, err = reader.ReadLine()
		if err != nil {
			break

		}
		callback(line)
	}

	return nil
}

// Subscribe to a particle.io event stream and returns a channel to receive them
func Subscribe(eventPrefix string, token string) <-chan Event {
	out := make(chan Event)

	var client = &http.Client{
		Timeout: 0,
	}

	req, err := http.NewRequest("GET", url+eventPrefix+"?access_token="+token, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(resp.Body)

	// check for :ok as first event on stream
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	} else if line != ":ok\n" {
		log.Fatal(line)
	}

	go func() {
		for {
			var event Event

			line, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			} else if strings.HasPrefix(line, "event:") {
				event.Name = strings.TrimPrefix(strings.TrimSuffix(line, "\n"), "event: ")
				line, err := reader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				} else if strings.HasPrefix(line, "data:") {
					jsonBlob := strings.TrimPrefix(strings.TrimSuffix(line, "\n"), "data: ")
					err := json.Unmarshal([]byte(jsonBlob), &event.Data)
					if err != nil {
						log.Fatal(err)
					}
					out <- event

				} else {
					log.Fatal("Expected event data, got: " + line)
				}
			} else if line == "\n" {
				// next
			} else {
				log.Fatal("Expected event name, got: " + line)
			}
		}
	}()

	return out
}
