package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// ShellyIOConfig describes the configuration of a Shelly device
type ShellyIOConfig struct {
	Name string `json:"name"`
}

type shellyGen2SysConfig struct {
	Device struct {
		Name string `json:"name"`
	} `json:"device"`
}

// Example response
// {"id":0, "source":"WS_in", "output":false, "apower":0.0, "voltage":123.3, "current":0.000, "aenergy":{"total":0.000,"by_minute":[0.000,0.000,0.000],"minute_ts":1680536525},"temperature":{"tC":44.4, "tF":112.0}}
type shellyGen2SwitchStatus struct {
	ID      int     `json:"id"`
	Source  string  `json:"source"`
	Output  bool    `json:"output"`
	Apower  float32 `json:"apower"`
	Voltage float32 `json:"voltage"`
	Current float32 `json:"current"`
	Aenergy struct {
		Total    float32   `json:"total"`
		ByMinute []float32 `json:"by_minute"`
		MinuteTS int64     `json:"minute_ts"`
	} `json:"aenergy"`
	Temperature struct {
		TC float32 `json:"tC"`
		TF float32 `json:"tF"`
	} `json:"temperature"`
}

type shellyGen2SwitchSetResp struct {
	WasOn bool `json:"wasOn"`
}

func (swi *shellyGen2SwitchStatus) toPoints() data.Points {
	now := time.Now()
	return data.Points{
		{Time: now, Type: data.PointTypeValue, Value: data.BoolToFloat(swi.Output)},
		{Time: now, Type: data.PointTypePower, Value: float64(swi.Apower)},
		{Time: now, Type: data.PointTypeVoltage, Value: float64(swi.Voltage)},
		{Time: now, Type: data.PointTypeCurrent, Value: float64(swi.Current)},
		{Time: now, Type: data.PointTypeTemperature, Value: float64(swi.Temperature.TC)},
	}
}

// Example response
// {"id":2,"state":true}
type shellyGen2InputStatus struct {
	ID    int  `json:"id"`
	State bool `json:"state"`
}

func (in *shellyGen2InputStatus) toPoints() data.Points {
	now := time.Now()
	return data.Points{
		{Time: now, Type: data.PointTypeValue,
			Key:   strconv.Itoa(in.ID),
			Value: data.BoolToFloat(in.State)},
	}
}

type shellyGen1LightStatus struct {
	Ison       bool `json:"ison"`
	Brightness int  `json:"brightness"`
	White      int  `json:"white"`
	Temp       int  `json:"temp"`
	Transition int  `json:"transition"`
}

func (sls *shellyGen1LightStatus) toPoints() data.Points {
	now := time.Now()
	return data.Points{
		{Time: now, Type: data.PointTypeValue, Value: data.BoolToFloat(sls.Ison)},
		{Time: now, Type: data.PointTypeBrightness, Value: float64(sls.Brightness)},
		{Time: now, Type: data.PointTypeWhite, Value: float64(sls.White)},
		{Time: now, Type: data.PointTypeLightTemp, Value: float64(sls.Temp)},
		{Time: now, Type: data.PointTypeTransition, Value: float64(sls.Transition)},
	}
}

func (sg2c shellyGen2SysConfig) toSettings() ShellyIOConfig {
	return ShellyIOConfig{
		Name: sg2c.Device.Name,
	}
}

// ShellyIo describes the config/state for a shelly io
type ShellyIo struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	DeviceID    string  `point:"deviceID"`
	Type        string  `point:"type"`
	IP          string  `point:"ip"`
	Value       float64 `point:"value"`
	ValueSet    float64 `point:"valueSet"`
	Offline     bool    `point:"offline"`
}

// Desc gets the description of a Shelly IO
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

// Shelly Generations
const (
	ShellyGenUnknown ShellyGen = iota
	ShellyGen1
	ShellyGen2
)

var shellyGenMap = map[string]ShellyGen{
	data.PointValueShellyTypeBulbDuo: ShellyGen1,
	data.PointValueShellyTypeRGBW2:   ShellyGen1,
	data.PointValueShellyType1PM:     ShellyGen1,
	data.PointValueShellyTypePlugUS:  ShellyGen2,
	data.PointValueShellyTypeI4:      ShellyGen2,
	data.PointValueShellyTypePlus1:   ShellyGen2,
}

var shellySettableOnOff = map[string]bool{
	data.PointValueShellyTypeBulbDuo: true,
	data.PointValueShellyTypeRGBW2:   true,
	data.PointValueShellyType1PM:     true,
	data.PointValueShellyTypePlugUS:  true,
	data.PointValueShellyTypePlus1:   true,
}

// Gen returns generation of Shelly device
func (sio *ShellyIo) Gen() ShellyGen {
	gen, ok := shellyGenMap[sio.Type]
	if !ok {
		return ShellyGenUnknown
	}

	return gen
}

// IsSettableOnOff returns true if the device can be turned on/off
func (sio *ShellyIo) IsSettableOnOff() bool {
	settable := shellySettableOnOff[sio.Type]
	return settable
}

// GetConfig returns the configuration of Shelly Device
func (sio *ShellyIo) GetConfig() (ShellyIOConfig, error) {
	switch sio.Gen() {
	case ShellyGen1:
		var ret ShellyIOConfig
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
		return config.toSettings(), err

	default:
		return ShellyIOConfig{}, fmt.Errorf("Unsupported device: %v", sio.Type)
	}
}

// SetOnOff sets on/off state of device
// BulbDuo: http://10.0.0.130/light/0?turn=on
// PlugUS: http://192.168.33.1/rpc/Switch.Set?id=0&on=true
func (sio *ShellyIo) SetOnOff(on bool) (data.Points, error) {
	switch sio.Type {
	case data.PointValueShellyTypeBulbDuo:
		onoff := "off"
		if on {
			onoff = "on"
		}
		res, err := httpClient.Get("http://" + sio.IP + "/light/0?turn=" + onoff)
		if err != nil {
			return data.Points{}, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return data.Points{}, fmt.Errorf("Shelly GetConfig returned an error code: %v", res.StatusCode)
		}

		var status shellyGen1LightStatus

		err = json.NewDecoder(res.Body).Decode(&status)
		if err != nil {
			return data.Points{}, err
		}
		return status.toPoints(), nil
	case data.PointValueShellyTypePlugUS, data.PointValueShellyTypePlus1:
		onValue := "false"
		if on {
			onValue = "true"
		}

		res, err := httpClient.Get("http://" + sio.IP + "/rpc/Switch.Set?id=0&on=" + onValue)
		if err != nil {
			return data.Points{}, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return data.Points{}, fmt.Errorf("Shelly Switch.Set returned an error code: %v", res.StatusCode)
		}

		var status shellyGen2SwitchSetResp

		err = json.NewDecoder(res.Body).Decode(&status)
		if err != nil {
			return data.Points{}, err
		}
		return data.Points{}, nil

	default:
		return data.Points{}, nil
	}
}

// GetStatus gets the current status of the device
func (sio *ShellyIo) GetStatus() (data.Points, error) {
	switch sio.Type {
	case data.PointValueShellyTypePlugUS, data.PointValueShellyTypePlus1:
		res, err := httpClient.Get("http://" + sio.IP + "/rpc/Switch.GetStatus?id=0")
		if err != nil {
			return data.Points{}, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return data.Points{}, fmt.Errorf("Shelly GetConfig returned an error code: %v", res.StatusCode)
		}

		var status shellyGen2SwitchStatus

		err = json.NewDecoder(res.Body).Decode(&status)
		if err != nil {
			return data.Points{}, err
		}
		return status.toPoints(), nil

	case data.PointValueShellyTypeI4:
		var points data.Points
		for channel := 0; channel < 4; channel++ {
			res, err := httpClient.Get("http://" + sio.IP + "/rpc/Input.GetStatus?id=" + strconv.Itoa(channel))
			if err != nil {
				return data.Points{}, err
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				return data.Points{}, fmt.Errorf("Shelly GetConfig returned an error code: %v", res.StatusCode)
			}

			var status shellyGen2InputStatus

			err = json.NewDecoder(res.Body).Decode(&status)
			if err != nil {
				return data.Points{}, err
			}

			points = append(points, status.toPoints()...)
		}

		return points, nil

	case data.PointValueShellyTypeBulbDuo:
		res, err := httpClient.Get("http://" + sio.IP + "/light/0")
		if err != nil {
			return data.Points{}, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return data.Points{}, fmt.Errorf("Shelly GetConfig returned an error code: %v", res.StatusCode)
		}

		var status shellyGen1LightStatus

		err = json.NewDecoder(res.Body).Decode(&status)
		if err != nil {
			return data.Points{}, err
		}
		return status.toPoints(), nil
	default:
		return data.Points{}, nil
	}
}

type shellyGen2Response struct {
	RestartRequired bool   `json:"restartRequired"`
	Code            int    `json:"code"`
	Message         string `json:"message"`
}

// SetName is use to set the name in a device
func (sio *ShellyIo) SetName(name string) error {
	switch sio.Gen() {
	case ShellyGen1:
		uri := fmt.Sprintf("http://%v/settings?name=%v", sio.IP, name)
		uri = strings.Replace(uri, " ", "%20", -1)
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
		uri = strings.Replace(uri, " ", "%20", -1)
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
		if err != nil {
			return err
		}
		if ret.Code != 0 || ret.Message != "" {
			return fmt.Errorf("Error setting Shelly device %v name: %v", sio.Type, ret.Message)
		}
	default:
		return fmt.Errorf("Error setting name: Unsupported device: %v", sio.Type)
	}
	return nil
}

// ShellyIOClient is a SIOT particle client
type ShellyIOClient struct {
	nc              *nats.Conn
	config          ShellyIo
	points          data.Points
	stop            chan struct{}
	newPoints       chan NewPoints
	newEdgePoints   chan NewPoints
	newShellyPoints chan NewPoints
	errorCount      int
}

// NewShellyIOClient ...
func NewShellyIOClient(nc *nats.Conn, config ShellyIo) Client {
	ne, err := data.Encode(config)
	if err != nil {
		log.Println("Error encoding shelly config: ", err)
	}
	return &ShellyIOClient{
		nc:              nc,
		config:          config,
		points:          ne.Points,
		stop:            make(chan struct{}),
		newPoints:       make(chan NewPoints),
		newEdgePoints:   make(chan NewPoints),
		newShellyPoints: make(chan NewPoints),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (sioc *ShellyIOClient) Run() error {
	log.Println("Starting shelly IO client: ", sioc.config.Description)

	sampleRate := time.Second * 2
	sampleRateOffline := time.Minute * 10

	syncConfigTicker := time.NewTicker(sampleRateOffline)
	sampleTicker := time.NewTicker(sampleRate)

	if sioc.config.Offline {
		sampleTicker = time.NewTicker(sampleRateOffline)
	}

	shellyError := func() {
		sioc.errorCount++
		if !sioc.config.Offline && sioc.errorCount > 5 {
			log.Printf("Shelly device %v is offline", sioc.config.Description)
			sioc.config.Offline = true
			err := SendNodePoint(sioc.nc, sioc.config.ID, data.Point{
				Type: data.PointTypeOffline, Value: 1}, false)

			if err != nil {
				log.Println("ShellyIO: error sending node point: ", err)
			}
			sampleTicker = time.NewTicker(sampleRateOffline)
		}
	}

	shellyCommOK := func() {
		sioc.errorCount = 0
		if sioc.config.Offline {
			log.Printf("Shelly device %v is online", sioc.config.Description)
			sioc.config.Offline = false
			err := SendNodePoint(sioc.nc, sioc.config.ID, data.Point{
				Type: data.PointTypeOffline, Value: 0}, false)

			if err != nil {
				log.Println("ShellyIO: error sending node point: ", err)
			}
			sampleTicker = time.NewTicker(sampleRate)
		}
	}

	syncConfig := func() {
		config, err := sioc.config.GetConfig()
		if err != nil {
			shellyError()
			log.Println("Error getting shelly IO settings: ", sioc.config.Desc(), err)
			return
		}

		shellyCommOK()

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
				}
			}

		case pts := <-sioc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sioc.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		case <-syncConfigTicker.C:
			syncConfig()

		case <-sampleTicker.C:
			points, err := sioc.config.GetStatus()
			if err != nil {
				log.Printf("Error getting status for %v: %v\n", sioc.config.Description, err)
				shellyError()
				break
			}

			if sioc.config.IsSettableOnOff() {
				if sioc.config.Value != sioc.config.ValueSet {
					pts, err := sioc.config.SetOnOff(data.FloatToBool(sioc.config.ValueSet))
					if err != nil {
						log.Printf("Error setting %v: %v\n", sioc.config.Description, err)
					}

					if len(pts) > 0 {
						points = append(points, pts...)
					} else {
						// get current status as the set did not return status
						points, err = sioc.config.GetStatus()
						if err != nil {
							log.Printf("Error getting status for %v: %v\n", sioc.config.Description, err)
							shellyError()
							break
						}
					}
				}
			}

			shellyCommOK()

			newPoints := sioc.points.Merge(points, time.Minute*15)
			if len(newPoints) > 0 {
				err := data.MergePoints(sioc.config.ID, newPoints, &sioc.config)
				if err != nil {
					log.Println("shelly io: error merging newPoints: ", err)
				}
				err = SendNodePoints(sioc.nc, sioc.config.ID, newPoints, false)
				if err != nil {
					log.Println("shelly io: error sending newPoints: ", err)
				}
			}
		}
	}

	// clean up
	return nil
}

// Stop sends a signal to the Run function to exit
func (sioc *ShellyIOClient) Stop(_ error) {
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
