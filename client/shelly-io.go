package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

type shellyIOConfig struct {
	Name string `json:"name"`
}

type shellyGen2SysConfig struct {
	Device struct {
		Name string `json:"name"`
	} `json:"device"`
}

func (sg2c shellyGen2SysConfig) toSettings() shellyIOConfig {
	return shellyIOConfig{
		Name: sg2c.Device.Name,
	}
}

// ShellyIo describes the config/state for a shelly io
type ShellyIo struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	DeviceID    string `point:"deviceID"`
	Type        string `point:"type"`
	IP          string `point:"ip"`
}

func (sio *ShellyIo) Desc() string {
	ret := sio.Type
	if len(sio.Description) > 0 {
		ret += ":" + sio.Description
	}
	return ret
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ShellyGen describes the generation of device (Gen1/Gen2)
type ShellyGen int

const (
	ShellyGenUnknown ShellyGen = iota
	ShellyGen1
	ShellyGen2
)

var shellyGenMap = map[string]ShellyGen{
	"BulbDuo": ShellyGen1,
	"rgbw2":   ShellyGen1,
	"1pm":     ShellyGen1,
	"PlugUS":  ShellyGen2,
}

// Gen returns generation of Shelly device
func (sio *ShellyIo) Gen() ShellyGen {
	gen, ok := shellyGenMap[sio.Type]
	if !ok {
		return ShellyGenUnknown
	}

	return gen
}

func (sio *ShellyIo) GetConfig() (shellyIOConfig, error) {
	switch sio.Gen() {
	case ShellyGen1:
		var ret shellyIOConfig
		res, err := httpClient.Get("http://" + sio.IP + "/settings")
		if err != nil {
			return ret, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return ret, fmt.Errorf("Shelly GetConfig returned an error code: %v", res.StatusCode)
		}

		err = json.NewDecoder(res.Body).Decode(&ret)

		return ret, err
	case ShellyGen2:
		var config shellyGen2SysConfig
		res, err := httpClient.Get("http://" + sio.IP + "/rpc/Sys.GetConfig")
		if err != nil {
			return config.toSettings(), err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return config.toSettings(), fmt.Errorf("Shelly GetConfig returned an error code: %v", res.StatusCode)
		}

		err = json.NewDecoder(res.Body).Decode(&config)
		return config.toSettings(), nil

	default:
		return shellyIOConfig{}, fmt.Errorf("Unsupported device: %v", sio.Type)
	}
}

type shellyGen2Response struct {
	RestartRequired bool   `json:"restartRequired"`
	Code            int    `json:"code"`
	Message         string `json:"message"`
}

func (sio *ShellyIo) SetName(name string) error {
	switch sio.Gen() {
	case ShellyGen1:
		uri := fmt.Sprintf("http://%v/settings?name=%v", sio.IP, name)
		res, err := httpClient.Get(uri)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("Shelly SetName returned an error code: %v", res.StatusCode)
		}
		// TODO: not sure how to test if it worked ...
	case ShellyGen2:
		uri := fmt.Sprintf("http://%v/rpc/Sys.Setconfig?config={\"device\":{\"name\":\"%v\"}}", sio.IP, name)
		res, err := httpClient.Get(uri)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("Shelly SetName returned an error code: %v", res.StatusCode)
		}
		var ret shellyGen2Response
		err = json.NewDecoder(res.Body).Decode(&ret)
		if ret.Code != 0 || ret.Message != "" {
			return fmt.Errorf("Error setting Shelly device %v name: %v", sio.Type, ret.Message)
		}
	default:
		return fmt.Errorf("Unsupported device: %v", sio.Type)
	}
	return nil
}

// ShellyIOClient is a SIOT particle client
type ShellyIOClient struct {
	nc              *nats.Conn
	config          ShellyIo
	stop            chan struct{}
	newPoints       chan NewPoints
	newEdgePoints   chan NewPoints
	newShellyPoints chan NewPoints
}

// NewShellyIOClient ...
func NewShellyIOClient(nc *nats.Conn, config ShellyIo) Client {
	return &ShellyIOClient{
		nc:              nc,
		config:          config,
		stop:            make(chan struct{}),
		newPoints:       make(chan NewPoints),
		newEdgePoints:   make(chan NewPoints),
		newShellyPoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (sioc *ShellyIOClient) Run() error {
	log.Println("Starting shelly IO client: ", sioc.config.Description)

	syncConfig := func() {
		config, err := sioc.config.GetConfig()
		if err != nil {
			log.Println("Error getting shelly IO settings: ", sioc.config.Desc(), err)
		}

		if sioc.config.Description == "" && config.Name != "" {
			sioc.config.Description = config.Name
			err := SendNodePoint(sioc.nc, sioc.config.ID, data.Point{
				Type: data.PointTypeDescription, Text: config.Name}, false)
			if err != nil {
				log.Println("Error sending shelly io description: ", err)
			}
		} else if sioc.config.Description != config.Name {
			err := sioc.config.SetName(sioc.config.Description)
			if err != nil {
				log.Println("Error setting name on Shelly device: ", err)
			}
		}
	}

	syncConfig()

	syncConfigTicker := time.NewTicker(time.Minute * 5)

done:
	for {
		select {
		case <-sioc.stop:
			log.Println("Stopping shelly IO client: ", sioc.config.Description)
			break done
		case pts := <-sioc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDescription:
					syncConfig()
				case data.PointTypeDisable:
					if p.Value == 1 {
					} else {
					}
				}
			}

		case pts := <-sioc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case <-syncConfigTicker.C:
			syncConfig()

		}
	}

	// clean up
	return nil
}

// Stop sends a signal to the Run function to exit
func (sioc *ShellyIOClient) Stop(err error) {
	close(sioc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sioc *ShellyIOClient) Points(nodeID string, points []data.Point) {
	sioc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sioc *ShellyIOClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	sioc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
