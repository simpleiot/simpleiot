package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// ShellyGen is the generation of a Shelly device. Generation 2 introduced the
// RPC API, and every generation after it speaks the same API, so anything at
// or above ShellyGen2 is handled the same way.
type ShellyGen int

// Shelly device generations
const (
	ShellyGen1 ShellyGen = 1
	ShellyGen2 ShellyGen = 2
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// shellyDeviceInfo is the response to GET /shelly, the one request both
// generations answer without authentication. A Gen1 device returns `type` and
// no `gen`; a Gen2 or later device returns `model` and its generation.
type shellyDeviceInfo struct {
	ID    string `json:"id"`
	MAC   string `json:"mac"`
	Name  string `json:"name"`
	Model string `json:"model"`
	Gen   int    `json:"gen"`
	Ver   string `json:"ver"`
	Type  string `json:"type"`
	Fw    string `json:"fw"`
}

// gen returns the device generation. Gen1 devices do not report one.
func (di shellyDeviceInfo) gen() ShellyGen {
	if di.Gen < int(ShellyGen2) {
		return ShellyGen1
	}
	return ShellyGen(di.Gen)
}

// model returns the model the device reports for itself, such as
// "SNPL-00116US" for a Plus Plug US or "SHPLG-S" for a Gen1 Plug S.
func (di shellyDeviceInfo) model() string {
	if di.Model != "" {
		return di.Model
	}
	return di.Type
}

// shellyGetDeviceInfo asks a device what it is. This replaces guessing the
// model from the mDNS hostname, and it is how a device that was released after
// this code was written still identifies itself correctly.
func shellyGetDeviceInfo(ip string) (shellyDeviceInfo, error) {
	var di shellyDeviceInfo
	res, err := httpClient.Get("http://" + ip + "/shelly")
	if err != nil {
		return di, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return di, fmt.Errorf("shelly /shelly returned status %v", res.StatusCode)
	}
	err = json.NewDecoder(res.Body).Decode(&di)
	return di, err
}

// shellyComp is one component instance a device reports, such as switch 0 or
// the temperature sensor an add-on module contributes as id 100. The id stays a
// string because it is also the Key of every point the component produces, and
// because add-on ids are not a dense range.
type shellyComp struct {
	name string
	id   string
}

// shellyCompPoints maps a Gen2 component name to the function that turns its
// status into points. A component absent from this map produces no points, so
// this map is also the list of components Simple IoT understands.
//
// Note what this map is not keyed by: the device model. A Gen2 or later device
// reports its own components through Shelly.GetStatus, so support follows from
// which components a device has rather than from whether this code has heard of
// it.
var shellyCompPoints = map[string]func(id string, raw json.RawMessage) (data.Points, error){
	"switch":      shellySwitchPoints,
	"light":       shellyLightPoints,
	"rgb":         shellyLightPoints,
	"rgbw":        shellyLightPoints,
	"cct":         shellyLightPoints,
	"input":       shellyInputPoints,
	"cover":       shellyCoverPoints,
	"temperature": shellyTemperaturePoints,
	"humidity":    shellyHumidityPoints,
	"devicepower": shellyDevicePowerPoints,
	"voltmeter":   shellyVoltmeterPoints,
	"em":          shellyEMPoints,
	"em1":         shellyEM1Points,
	"smoke":       shellySmokePoints,
}

// shellyCompSetPoint maps a component to the point type that commands it. A
// component absent from this map cannot be driven.
var shellyCompSetPoint = map[string]string{
	"switch": data.PointTypeSwitchSet,
	"light":  data.PointTypeLightSet,
	"rgb":    data.PointTypeLightSet,
	"rgbw":   data.PointTypeLightSet,
	"cct":    data.PointTypeLightSet,
	"cover":  data.PointTypePositionSet,
}

// shellyCompStatePoint maps a component to the point type that reports the
// state its set point commands, so the client can tell when they differ.
var shellyCompStatePoint = map[string]string{
	"switch": data.PointTypeSwitch,
	"light":  data.PointTypeLight,
	"rgb":    data.PointTypeLight,
	"rgbw":   data.PointTypeLight,
	"cct":    data.PointTypeLight,
	"cover":  data.PointTypePosition,
	"input":  data.PointTypeInput,
}

// shellyCompsFromStatus reads the component list out of a Shelly.GetStatus or
// Shelly.GetConfig response. Component keys are "<name>:<id>"; keys without an
// id are device services such as "sys", "wifi", and "cloud".
func shellyCompsFromStatus(status map[string]json.RawMessage) []shellyComp {
	comps := []shellyComp{}
	for key := range status {
		name, id, found := strings.Cut(key, ":")
		if !found {
			continue
		}
		if _, ok := shellyCompPoints[name]; !ok {
			continue
		}
		comps = append(comps, shellyComp{name: name, id: id})
	}
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].name != comps[j].name {
			return comps[i].name < comps[j].name
		}
		return shellyCompIDLess(comps[i].id, comps[j].id)
	})
	return comps
}

// shellyCompIDLess orders component ids numerically where it can, so that
// id 9 sorts before id 100 rather than after it.
func shellyCompIDLess(a, b string) bool {
	ai, aErr := strconv.Atoi(a)
	bi, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil {
		return ai < bi
	}
	return a < b
}

// shellyPointsFromStatus converts a whole Shelly.GetStatus response, or the
// partial status a NotifyStatus frame carries, into points.
func shellyPointsFromStatus(status map[string]json.RawMessage) data.Points {
	pts := data.Points{}
	for _, comp := range shellyCompsFromStatus(status) {
		toPoints := shellyCompPoints[comp.name]
		cPts, err := toPoints(comp.id, status[comp.name+":"+comp.id])
		if err != nil {
			// A component we cannot decode should not discard the ones we can.
			continue
		}
		pts = append(pts, cPts...)
	}
	return pts
}

// shellyMergeStatus merges the partial status a NotifyStatus frame carries into
// the cached status, leaving components the frame does not mention alone.
func shellyMergeStatus(dst, src map[string]json.RawMessage) {
	for k, v := range src {
		dst[k] = v
	}
}

// Gen2 component status shapes. Every field is a pointer because a component
// reports only the fields it has: a switch without power monitoring has no
// apower, and an add-on temperature sensor has no output.

type shellyGen2Temperature struct {
	TC *float64 `json:"tC"`
}

type shellyGen2Energy struct {
	Total *float64 `json:"total"`
}

type shellyGen2Switch struct {
	Output      *bool                  `json:"output"`
	Apower      *float64               `json:"apower"`
	Voltage     *float64               `json:"voltage"`
	Current     *float64               `json:"current"`
	PF          *float64               `json:"pf"`
	Freq        *float64               `json:"freq"`
	Aenergy     *shellyGen2Energy      `json:"aenergy"`
	Temperature *shellyGen2Temperature `json:"temperature"`
}

func shellySwitchPoints(id string, raw json.RawMessage) (data.Points, error) {
	var s shellyGen2Switch
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	pts := data.Points{}
	if s.Output != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeSwitch, id, data.BoolToFloat(*s.Output)))
	}
	return append(pts, shellyMeterPoints(id, s.Apower, s.Voltage, s.Current, s.PF, s.Freq, s.Aenergy, s.Temperature)...), nil
}

// shellyMeterPoints emits the measurement points a switch or cover reports when
// it has power monitoring. A device without it reports none of these, so the
// nil checks take the place of a table of which models measure power.
func shellyMeterPoints(id string, apower, voltage, current, pf, freq *float64,
	energy *shellyGen2Energy, temp *shellyGen2Temperature) data.Points {
	pts := data.Points{}
	if apower != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypePower, id, *apower))
	}
	if voltage != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeVoltage, id, *voltage))
	}
	if current != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeCurrent, id, *current))
	}
	if pf != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypePowerFactor, id, *pf))
	}
	if freq != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeFrequency, id, *freq))
	}
	if energy != nil && energy.Total != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeEnergy, id, *energy.Total))
	}
	if temp != nil && temp.TC != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeTemperature, id, *temp.TC))
	}
	return pts
}

type shellyGen2Light struct {
	Output      *bool                  `json:"output"`
	Brightness  *float64               `json:"brightness"`
	White       *float64               `json:"white"`
	CT          *float64               `json:"ct"`
	Apower      *float64               `json:"apower"`
	Voltage     *float64               `json:"voltage"`
	Current     *float64               `json:"current"`
	Aenergy     *shellyGen2Energy      `json:"aenergy"`
	Temperature *shellyGen2Temperature `json:"temperature"`
}

func shellyLightPoints(id string, raw json.RawMessage) (data.Points, error) {
	var l shellyGen2Light
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, err
	}
	pts := data.Points{}
	if l.Output != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeLight, id, data.BoolToFloat(*l.Output)))
	}
	if l.Brightness != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeBrightness, id, *l.Brightness))
	}
	if l.White != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeWhite, id, *l.White))
	}
	if l.CT != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeLightTemp, id, *l.CT))
	}
	return append(pts, shellyMeterPoints(id, l.Apower, l.Voltage, l.Current, nil, nil, l.Aenergy, l.Temperature)...), nil
}

type shellyGen2Input struct {
	State   *bool    `json:"state"`
	Percent *float64 `json:"percent"`
}

func shellyInputPoints(id string, raw json.RawMessage) (data.Points, error) {
	var in shellyGen2Input
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	pts := data.Points{}
	if in.State != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeInput, id, data.BoolToFloat(*in.State)))
	}
	if in.Percent != nil {
		// An analog input reports a percentage rather than a state.
		pts = append(pts, data.NewPointFloat(data.PointTypeValue, id, *in.Percent))
	}
	return pts, nil
}

type shellyGen2Cover struct {
	State       *string                `json:"state"`
	CurrentPos  *float64               `json:"current_pos"`
	Apower      *float64               `json:"apower"`
	Voltage     *float64               `json:"voltage"`
	Current     *float64               `json:"current"`
	PF          *float64               `json:"pf"`
	Freq        *float64               `json:"freq"`
	Aenergy     *shellyGen2Energy      `json:"aenergy"`
	Temperature *shellyGen2Temperature `json:"temperature"`
}

func shellyCoverPoints(id string, raw json.RawMessage) (data.Points, error) {
	var c shellyGen2Cover
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	pts := data.Points{}
	if c.State != nil {
		pts = append(pts, data.NewPointString(data.PointTypeCoverState, id, *c.State))
	}
	if c.CurrentPos != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypePosition, id, *c.CurrentPos))
	}
	return append(pts, shellyMeterPoints(id, c.Apower, c.Voltage, c.Current, c.PF, c.Freq, c.Aenergy, c.Temperature)...), nil
}

func shellyTemperaturePoints(id string, raw json.RawMessage) (data.Points, error) {
	var t shellyGen2Temperature
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	if t.TC == nil {
		return data.Points{}, nil
	}
	return data.Points{data.NewPointFloat(data.PointTypeTemperature, id, *t.TC)}, nil
}

func shellyHumidityPoints(id string, raw json.RawMessage) (data.Points, error) {
	var h struct {
		RH *float64 `json:"rh"`
	}
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, err
	}
	if h.RH == nil {
		return data.Points{}, nil
	}
	return data.Points{data.NewPointFloat(data.PointTypeHumidity, id, *h.RH)}, nil
}

func shellyDevicePowerPoints(id string, raw json.RawMessage) (data.Points, error) {
	var dp struct {
		Battery *struct {
			V       *float64 `json:"V"`
			Percent *float64 `json:"percent"`
		} `json:"battery"`
		External *struct {
			Present *bool `json:"present"`
		} `json:"external"`
	}
	if err := json.Unmarshal(raw, &dp); err != nil {
		return nil, err
	}
	pts := data.Points{}
	if dp.Battery != nil {
		if dp.Battery.V != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeBattery, id, *dp.Battery.V))
		}
		if dp.Battery.Percent != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeBatteryLevel, id, *dp.Battery.Percent))
		}
	}
	if dp.External != nil && dp.External.Present != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeExternalPower, id,
			data.BoolToFloat(*dp.External.Present)))
	}
	return pts, nil
}

func shellyVoltmeterPoints(id string, raw json.RawMessage) (data.Points, error) {
	var v struct {
		Voltage *float64 `json:"voltage"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	if v.Voltage == nil {
		return data.Points{}, nil
	}
	return data.Points{data.NewPointFloat(data.PointTypeVoltage, id, *v.Voltage)}, nil
}

func shellySmokePoints(id string, raw json.RawMessage) (data.Points, error) {
	var s struct {
		Alarm *bool `json:"alarm"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if s.Alarm == nil {
		return data.Points{}, nil
	}
	return data.Points{data.NewPointFloat(data.PointTypeAlarm, id, data.BoolToFloat(*s.Alarm))}, nil
}

// shellyEM1Points converts a single-phase energy meter.
func shellyEM1Points(id string, raw json.RawMessage) (data.Points, error) {
	var em struct {
		ActPower  *float64 `json:"act_power"`
		AprtPower *float64 `json:"aprt_power"`
		Voltage   *float64 `json:"voltage"`
		Current   *float64 `json:"current"`
		PF        *float64 `json:"pf"`
		Freq      *float64 `json:"freq"`
	}
	if err := json.Unmarshal(raw, &em); err != nil {
		return nil, err
	}
	return shellyPhasePoints(id, em.ActPower, em.AprtPower, em.Voltage, em.Current, em.PF, em.Freq), nil
}

// shellyEMPoints converts a three-phase energy meter. Each phase becomes its
// own point key, "<id>.a" through "<id>.c", with the totals under "<id>".
func shellyEMPoints(id string, raw json.RawMessage) (data.Points, error) {
	var em struct {
		AActPower  *float64 `json:"a_act_power"`
		AAprtPower *float64 `json:"a_aprt_power"`
		AVoltage   *float64 `json:"a_voltage"`
		ACurrent   *float64 `json:"a_current"`
		APF        *float64 `json:"a_pf"`
		AFreq      *float64 `json:"a_freq"`

		BActPower  *float64 `json:"b_act_power"`
		BAprtPower *float64 `json:"b_aprt_power"`
		BVoltage   *float64 `json:"b_voltage"`
		BCurrent   *float64 `json:"b_current"`
		BPF        *float64 `json:"b_pf"`
		BFreq      *float64 `json:"b_freq"`

		CActPower  *float64 `json:"c_act_power"`
		CAprtPower *float64 `json:"c_aprt_power"`
		CVoltage   *float64 `json:"c_voltage"`
		CCurrent   *float64 `json:"c_current"`
		CPF        *float64 `json:"c_pf"`
		CFreq      *float64 `json:"c_freq"`

		TotalActPower  *float64 `json:"total_act_power"`
		TotalAprtPower *float64 `json:"total_aprt_power"`
		TotalCurrent   *float64 `json:"total_current"`
	}
	if err := json.Unmarshal(raw, &em); err != nil {
		return nil, err
	}
	pts := shellyPhasePoints(id+".a", em.AActPower, em.AAprtPower, em.AVoltage, em.ACurrent, em.APF, em.AFreq)
	pts = append(pts, shellyPhasePoints(id+".b", em.BActPower, em.BAprtPower, em.BVoltage, em.BCurrent, em.BPF, em.BFreq)...)
	pts = append(pts, shellyPhasePoints(id+".c", em.CActPower, em.CAprtPower, em.CVoltage, em.CCurrent, em.CPF, em.CFreq)...)
	return append(pts, shellyPhasePoints(id, em.TotalActPower, em.TotalAprtPower, nil, em.TotalCurrent, nil, nil)...), nil
}

func shellyPhasePoints(key string, actPower, aprtPower, voltage, current, pf, freq *float64) data.Points {
	pts := data.Points{}
	if actPower != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypePower, key, *actPower))
	}
	if aprtPower != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeApparentPower, key, *aprtPower))
	}
	if voltage != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeVoltage, key, *voltage))
	}
	if current != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeCurrent, key, *current))
	}
	if pf != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypePowerFactor, key, *pf))
	}
	if freq != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeFrequency, key, *freq))
	}
	return pts
}

// ShellyIo describes the config/state for a shelly io
type ShellyIo struct {
	ID          string             `node:"id"`
	Parent      string             `node:"parent"`
	Description string             `point:"description"`
	DeviceID    string             `point:"deviceID"`
	Type        string             `point:"type"`
	Generation  int                `point:"gen"`
	IP          string             `point:"ip"`
	Switch      map[string]bool    `point:"switch"`
	SwitchSet   map[string]bool    `point:"switchSet"`
	Light       map[string]bool    `point:"light"`
	LightSet    map[string]bool    `point:"lightSet"`
	Input       map[string]bool    `point:"input"`
	Position    map[string]float64 `point:"position"`
	PositionSet map[string]float64 `point:"positionSet"`
	Offline     bool               `point:"offline"`
	Controlled  bool               `point:"controlled"`
	Disabled    bool               `point:"disabled"`
}

// Desc gets the description of a Shelly IO
func (sio *ShellyIo) Desc() string {
	ret := sio.Type
	if len(sio.Description) > 0 {
		ret += ":" + sio.Description
	}
	return ret
}

// Gen returns the generation of the Shelly device
func (sio *ShellyIo) Gen() ShellyGen {
	if sio.Generation < int(ShellyGen2) {
		return ShellyGen1
	}
	return ShellyGen(sio.Generation)
}

// shellyIOConfig describes the configuration of a Shelly device
type shellyIOConfig struct {
	Name string
}

// GetConfig returns the configuration of Shelly Device
func (sio *ShellyIo) getConfig() (shellyIOConfig, error) {
	var ret shellyIOConfig
	if sio.Gen() >= ShellyGen2 {
		var config struct {
			Device struct {
				Name string `json:"name"`
			} `json:"device"`
		}
		err := sio.rpc("Sys.GetConfig", nil, &config)
		ret.Name = config.Device.Name
		return ret, err
	}

	var settings struct {
		Name string `json:"name"`
	}
	err := sio.gen1Get("settings", nil, &settings)
	ret.Name = settings.Name
	return ret, err
}

// SetName is use to set the name in a device
func (sio *ShellyIo) SetName(name string) error {
	if sio.Gen() >= ShellyGen2 {
		params := map[string]interface{}{
			"config": map[string]interface{}{
				"device": map[string]interface{}{"name": name},
			},
		}
		return sio.rpc("Sys.SetConfig", params, nil)
	}

	return sio.gen1Get("settings", map[string]string{"name": name}, nil)
}

// GetStatus reads the status of every component the device reports.
//
// A Gen2 or later device answers the whole device in one request, so this does
// not depend on knowing which components the model has.
func (sio *ShellyIo) GetStatus() (data.Points, error) {
	if sio.Gen() >= ShellyGen2 {
		var status map[string]json.RawMessage
		if err := sio.rpc("Shelly.GetStatus", nil, &status); err != nil {
			return nil, err
		}
		return shellyPointsFromStatus(status), nil
	}

	var status shellyGen1Status
	if err := sio.gen1Get("status", nil, &status); err != nil {
		return nil, err
	}
	return status.toPoints(), nil
}

// components returns the components the device reports.
func (sio *ShellyIo) components() ([]shellyComp, error) {
	if sio.Gen() >= ShellyGen2 {
		var status map[string]json.RawMessage
		if err := sio.rpc("Shelly.GetStatus", nil, &status); err != nil {
			return nil, err
		}
		return shellyCompsFromStatus(status), nil
	}

	var status shellyGen1Status
	if err := sio.gen1Get("status", nil, &status); err != nil {
		return nil, err
	}
	return status.comps(), nil
}

// SetOnOff turns a component on or off.
func (sio *ShellyIo) SetOnOff(comp, id string, on bool) error {
	if sio.Gen() >= ShellyGen2 {
		method := strings.ToUpper(comp[:1]) + comp[1:] + ".Set"
		idNum, err := strconv.Atoi(id)
		if err != nil {
			return fmt.Errorf("shelly component id %v is not a number", id)
		}
		return sio.rpc(method, map[string]interface{}{"id": idNum, "on": on}, nil)
	}

	onoff := "off"
	if on {
		onoff = "on"
	}
	// Gen1 relays live at /relay/<id>; the "switch" name is Gen2's.
	path := comp
	if comp == "switch" {
		path = "relay"
	}
	return sio.gen1Get(path+"/"+id, map[string]string{"turn": onoff}, nil)
}

// SetPosition drives a cover to a position, 0 through 100.
func (sio *ShellyIo) SetPosition(id string, pos float64) error {
	idNum, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("shelly component id %v is not a number", id)
	}
	return sio.rpc("Cover.GoToPosition", map[string]interface{}{
		"id": idNum, "pos": int(pos),
	}, nil)
}

// Gen1 status. Components are enumerated from the arrays the device returns,
// so a Gen1 device with two relays reports two switch components without this
// code holding a per-model count.
type shellyGen1Status struct {
	Relays []struct {
		IsOn bool `json:"ison"`
	} `json:"relays"`
	Lights []struct {
		IsOn       bool     `json:"ison"`
		Brightness *float64 `json:"brightness"`
		White      *float64 `json:"white"`
		Temp       *float64 `json:"temp"`
	} `json:"lights"`
	Meters []struct {
		Power *float64 `json:"power"`
		Total *float64 `json:"total"`
	} `json:"meters"`
	Inputs []struct {
		Input int `json:"input"`
	} `json:"inputs"`
	Temperature *float64 `json:"temperature"`
	Tmp         *struct {
		TC *float64 `json:"tC"`
	} `json:"tmp"`
	Hum *struct {
		Value *float64 `json:"value"`
	} `json:"hum"`
	Bat *struct {
		Value   *float64 `json:"value"`
		Voltage *float64 `json:"voltage"`
	} `json:"bat"`
}

func (s shellyGen1Status) comps() []shellyComp {
	comps := []shellyComp{}
	for i := range s.Relays {
		comps = append(comps, shellyComp{"switch", strconv.Itoa(i)})
	}
	for i := range s.Lights {
		comps = append(comps, shellyComp{"light", strconv.Itoa(i)})
	}
	for i := range s.Inputs {
		comps = append(comps, shellyComp{"input", strconv.Itoa(i)})
	}
	return comps
}

func (s shellyGen1Status) toPoints() data.Points {
	pts := data.Points{}
	for i, r := range s.Relays {
		key := strconv.Itoa(i)
		pts = append(pts, data.NewPointFloat(data.PointTypeSwitch, key, data.BoolToFloat(r.IsOn)))
		// A Gen1 device reports one meter per relay when it measures power.
		if i < len(s.Meters) {
			if p := s.Meters[i].Power; p != nil {
				pts = append(pts, data.NewPointFloat(data.PointTypePower, key, *p))
			}
			if t := s.Meters[i].Total; t != nil {
				pts = append(pts, data.NewPointFloat(data.PointTypeEnergy, key, *t))
			}
		}
	}
	for i, l := range s.Lights {
		key := strconv.Itoa(i)
		pts = append(pts, data.NewPointFloat(data.PointTypeLight, key, data.BoolToFloat(l.IsOn)))
		if l.Brightness != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeBrightness, key, *l.Brightness))
		}
		if l.White != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeWhite, key, *l.White))
		}
		if l.Temp != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeLightTemp, key, *l.Temp))
		}
	}
	for i, in := range s.Inputs {
		pts = append(pts, data.NewPointFloat(data.PointTypeInput, strconv.Itoa(i),
			data.BoolToFloat(in.Input != 0)))
	}
	if s.Temperature != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeTemperature, "0", *s.Temperature))
	} else if s.Tmp != nil && s.Tmp.TC != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeTemperature, "0", *s.Tmp.TC))
	}
	if s.Hum != nil && s.Hum.Value != nil {
		pts = append(pts, data.NewPointFloat(data.PointTypeHumidity, "0", *s.Hum.Value))
	}
	if s.Bat != nil {
		if s.Bat.Voltage != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeBattery, "0", *s.Bat.Voltage))
		}
		if s.Bat.Value != nil {
			pts = append(pts, data.NewPointFloat(data.PointTypeBatteryLevel, "0", *s.Bat.Value))
		}
	}
	return pts
}

// gen1Get makes a Gen1 HTTP request and decodes the response into result when
// one is supplied.
func (sio *ShellyIo) gen1Get(path string, params map[string]string, result interface{}) error {
	uri := "http://" + sio.IP + "/" + path
	if len(params) > 0 {
		q := make([]string, 0, len(params))
		for k, v := range params {
			q = append(q, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
		sort.Strings(q)
		uri += "?" + strings.Join(q, "&")
	}
	res, err := httpClient.Get(uri)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("shelly %v returned status %v", path, res.StatusCode)
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(result)
}
