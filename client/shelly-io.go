package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// shellyIOConfig describes the configuration of a Shelly device
type shellyIOConfig struct {
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

func (swi *shellyGen2SwitchStatus) toPoints(index int) data.Points {
	now := time.Now()
	key := strconv.Itoa(index)
	return data.Points{
		{Time: now, Type: data.PointTypeSwitch, Key: key, Value: data.BoolToFloat(swi.Output)},
		{Time: now, Type: data.PointTypePower, Key: key, Value: float64(swi.Apower)},
		{Time: now, Type: data.PointTypeVoltage, Key: key, Value: float64(swi.Voltage)},
		{Time: now, Type: data.PointTypeCurrent, Key: key, Value: float64(swi.Current)},
		{Time: now, Type: data.PointTypeTemperature, Key: key, Value: float64(swi.Temperature.TC)},
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
		{Time: now, Type: data.PointTypeInput,
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
		{Time: now, Type: data.PointTypeLight, Key: "0", Value: data.BoolToFloat(sls.Ison)},
		{Time: now, Type: data.PointTypeBrightness, Key: "0", Value: float64(sls.Brightness)},
		{Time: now, Type: data.PointTypeWhite, Key: "0", Value: float64(sls.White)},
		{Time: now, Type: data.PointTypeLightTemp, Key: "0", Value: float64(sls.Temp)},
		{Time: now, Type: data.PointTypeTransition, Key: "0", Value: float64(sls.Transition)},
	}
}

func (sg2c shellyGen2SysConfig) toSettings() shellyIOConfig {
	return shellyIOConfig{
		Name: sg2c.Device.Name,
	}
}

// ShellyIo describes the config/state for a shelly io
type ShellyIo struct {
	ID          string    `node:"id"`
	Parent      string    `node:"parent"`
	Description string    `point:"description"`
	DeviceID    string    `point:"deviceID"`
	Type        string    `point:"type"`
	IP          string    `point:"ip"`
	Value       []float64 `point:"value"`
	ValueSet    []float64 `point:"valueSet"`
	Switch      []bool    `point:"switch"`
	SwitchSet   []bool    `point:"switchSet"`
	Light       []bool    `point:"light"`
	LightSet    []bool    `point:"lightSet"`
	Input       []bool    `point:"input"`
	Offline     bool      `point:"offline"`
	Control     bool      `point:"control"`
	Disabled    bool      `point:"disabled"`
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
	data.PointValueShellyTypePlugUK:  ShellyGen2,
	data.PointValueShellyTypePlugIT:  ShellyGen2,
	data.PointValueShellyTypePlugS:   ShellyGen2,
	data.PointValueShellyTypeI4:      ShellyGen2,
	data.PointValueShellyTypePlus1:   ShellyGen2,
	data.PointValueShellyTypePlus2PM: ShellyGen2,
}

// Gen 2 metadata

// shellComp is used to describe shelly "components" a device may support
type shellyComp struct {
	name  string
	count int
}

var shellyCompMap = map[string][]shellyComp{
	data.PointValueShellyTypeBulbDuo: {{"light", 1}},
	data.PointValueShellyType1PM:     {{"switch", 1}},
	data.PointValueShellyTypeI4:      {{"input", 4}},
	data.PointValueShellyTypePlugUS:  {{"switch", 1}},
	data.PointValueShellyTypePlugUK:  {{"switch", 1}},
	data.PointValueShellyTypePlugIT:  {{"switch", 1}},
	data.PointValueShellyTypePlugS:   {{"switch", 1}},
	data.PointValueShellyTypePlus1:   {{"switch", 1}, {"input", 1}},
	data.PointValueShellyTypePlus2PM: {{"switch", 2}, {"input", 2}},
}

var shellySettableOnOff = map[string]bool{
	data.PointValueShellyTypeBulbDuo: true,
	data.PointValueShellyTypeRGBW2:   true,
	data.PointValueShellyType1PM:     true,
	data.PointValueShellyTypePlugUS:  true,
	data.PointValueShellyTypePlugUK:  true,
	data.PointValueShellyTypePlugIT:  true,
	data.PointValueShellyTypePlugS:   true,
	data.PointValueShellyTypePlus1:   true,
	data.PointValueShellyTypePlus2PM: true,
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
func (sio *ShellyIo) getConfig() (shellyIOConfig, error) {
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
		return config.toSettings(), err

	default:
		return shellyIOConfig{}, fmt.Errorf("Unsupported device: %v", sio.Type)
	}
}

// SetOnOff sets on/off state of device
// BulbDuo: http://10.0.0.130/light/0?turn=on
// PlugUS: http://192.168.33.1/rpc/Switch.Set?id=0&on=true
func (sio *ShellyIo) SetOnOff(comp string, index int, on bool) (data.Points, error) {
	if len(comp) < 2 {
		return nil, fmt.Errorf("Component must be specified")
	}
	_ = index
	gen := sio.Gen()
	switch gen {
	case ShellyGen1:
		onoff := "off"
		if on {
			onoff = "on"
		}
		url := fmt.Sprintf("http://%v/%v/%v?turn=%v", sio.IP, comp, index, onoff)
		res, err := httpClient.Get(url)
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
	case ShellyGen2:
		onValue := "false"
		if on {
			onValue = "true"
		}

		compCap := strings.ToUpper(string(comp[0])) + comp[1:]

		url := fmt.Sprintf("http://%v/rpc/%v.Set?id=%v&on=%v", sio.IP, compCap, index, onValue)
		res, err := httpClient.Get(url)
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

func (sio *ShellyIo) gen1GetLight(count int) (data.Points, error) {
	ret := data.Points{}

	for i := 0; i < count; i++ {
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

		ret = append(ret, status.toPoints()...)
	}

	return ret, nil
}

func (sio *ShellyIo) gen2GetSwitch(count int) (data.Points, error) {
	ret := data.Points{}

	for i := 0; i < count; i++ {

		url := fmt.Sprintf("http://%v/rpc/Switch.GetStatus?id=%v", sio.IP, i)

		res, err := httpClient.Get(url)
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
		pts := status.toPoints(i)
		ret = append(ret, pts...)
	}

	return ret, nil
}

func (sio *ShellyIo) gen2GetInput(count int) (data.Points, error) {
	var points data.Points
	for channel := 0; channel < count; channel++ {
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
}

// GetStatus gets the current status of the device
func (sio *ShellyIo) GetStatus() (data.Points, error) {
	ret := data.Points{}

	// TODO: this needs clean up some to be data driven instead of all the if statements
	gen := sio.Gen()

	if cnt := sio.getCompCount("switch"); cnt > 0 {
		if gen == ShellyGen1 {
			_ = gen
			// TODO: need to add gen 1 support for switch status
		}
		if gen == ShellyGen2 {
			pts, err := sio.gen2GetSwitch(cnt)
			if err != nil {
				return nil, err
			}
			ret = append(ret, pts...)
		}
	}

	if cnt := sio.getCompCount("input"); cnt > 0 {
		if gen == ShellyGen1 {
			_ = gen
			// TODO: need to add gen 1 support for input status
		}
		if gen == ShellyGen2 {
			pts, err := sio.gen2GetInput(cnt)
			if err != nil {
				return nil, err
			}
			ret = append(ret, pts...)
		}
	}

	if cnt := sio.getCompCount("light"); cnt > 0 {
		if gen == ShellyGen1 {
			pts, err := sio.gen1GetLight(cnt)
			if err != nil {
				return nil, err
			}
			ret = append(ret, pts...)
		}
	}

	return ret, nil
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

// GetCompCount returns the number of components found in the device
func (sio *ShellyIo) getCompCount(comp string) int {
	comps, ok := shellyCompMap[sio.Type]
	if !ok {
		return 0
	}

	for _, c := range comps {
		if c.name == comp {
			return c.count
		}
	}

	return 0
}
