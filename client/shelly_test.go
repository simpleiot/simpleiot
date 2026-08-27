package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

// Recorded Shelly.GetStatus responses. The point of these tests is that the
// component list and the points come out of the response, so a device this
// code has never been told about is still read correctly.

const statusPlus2PM = `{
	"ble": {},
	"cloud": {"connected": true},
	"input:0": {"id": 0, "state": false},
	"input:1": {"id": 1, "state": true},
	"mqtt": {"connected": false},
	"switch:0": {"id": 0, "source": "WS_in", "output": false, "apower": 0.0,
		"voltage": 123.3, "current": 0.0,
		"aenergy": {"total": 1.234, "by_minute": [0, 0, 0], "minute_ts": 1680536525},
		"temperature": {"tC": 44.4, "tF": 112.0}},
	"switch:1": {"id": 1, "source": "init", "output": true, "apower": 7.5,
		"voltage": 123.1, "current": 0.061,
		"aenergy": {"total": 9.9, "by_minute": [0, 0, 0], "minute_ts": 1680536525},
		"temperature": {"tC": 44.9, "tF": 112.8}},
	"sys": {"mac": "A8032AB0FFFF"},
	"wifi": {"sta_ip": "192.168.1.42"}
}`

// A Plus i4 has four inputs and nothing else. The old per-model table gave all
// four the same id, so only input 0 was ever read.
const statusPlusI4 = `{
	"ble": {},
	"input:0": {"id": 0, "state": false},
	"input:1": {"id": 1, "state": true},
	"input:2": {"id": 2, "state": false},
	"input:3": {"id": 3, "state": true},
	"sys": {"mac": "A8032AB0FFFF"}
}`

const statusPlugUS = `{
	"cloud": {"connected": true},
	"switch:0": {"id": 0, "output": true, "apower": 12.5, "voltage": 120.1,
		"current": 0.104, "aenergy": {"total": 42.5},
		"temperature": {"tC": 38.2, "tF": 100.8}},
	"sys": {"mac": "C049EF8889A0"}
}`

// A Plus 1 with an add-on module. The add-on contributes components with ids
// well outside the range of the device's own, which is exactly what a model
// table cannot express.
const statusPlus1Addon = `{
	"input:0": {"id": 0, "state": false},
	"switch:0": {"id": 0, "output": false},
	"temperature:100": {"id": 100, "tC": 21.4, "tF": 70.5},
	"humidity:100": {"id": 100, "rh": 47.5},
	"input:100": {"id": 100, "state": true},
	"sys": {"mac": "A8032AB0FFFF"}
}`

const statusPro3EM = `{
	"em:0": {"id": 0,
		"a_current": 1.1, "a_voltage": 231.0, "a_act_power": 253.0, "a_aprt_power": 254.1, "a_pf": 0.99,
		"b_current": 2.2, "b_voltage": 232.0, "b_act_power": 510.0, "b_aprt_power": 511.2, "b_pf": 0.98,
		"c_current": 3.3, "c_voltage": 233.0, "c_act_power": 768.0, "c_aprt_power": 769.3, "c_pf": 0.97,
		"total_current": 6.6, "total_act_power": 1531.0, "total_aprt_power": 1534.6},
	"temperature:0": {"id": 0, "tC": 41.2, "tF": 106.2},
	"sys": {"mac": "A8032AB0FFFF"}
}`

// A cover device reports its position and state rather than an output.
const statusPlus2PMCover = `{
	"cover:0": {"id": 0, "source": "init", "state": "stopped",
		"current_pos": 35, "target_pos": null, "apower": 0.0, "voltage": 231.4,
		"current": 0.0, "aenergy": {"total": 3.5},
		"temperature": {"tC": 39.1, "tF": 102.4}},
	"input:0": {"id": 0, "state": false},
	"input:1": {"id": 1, "state": false},
	"sys": {"mac": "A8032AB0FFFF"}
}`

func mustStatus(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	var status map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &status); err != nil {
		t.Fatalf("error decoding test status: %v", err)
	}
	return status
}

// compStrings renders a component list as "<name>:<id>" so a test can state
// what it expects in the same form the device does.
func compStrings(comps []shellyComp) []string {
	ret := make([]string, len(comps))
	for i, c := range comps {
		ret[i] = c.name + ":" + c.id
	}
	return ret
}

func TestShellyComponentsFromStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		exp    []string
	}{
		{"Plus2PM", statusPlus2PM,
			[]string{"input:0", "input:1", "switch:0", "switch:1"}},
		{"PlusI4", statusPlusI4,
			[]string{"input:0", "input:1", "input:2", "input:3"}},
		{"PlugUS", statusPlugUS,
			[]string{"switch:0"}},
		{"Plus1Addon", statusPlus1Addon,
			[]string{"humidity:100", "input:0", "input:100", "switch:0", "temperature:100"}},
		{"Pro3EM", statusPro3EM,
			[]string{"em:0", "temperature:0"}},
		{"Plus2PMCover", statusPlus2PMCover,
			[]string{"cover:0", "input:0", "input:1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			comps := compStrings(shellyCompsFromStatus(mustStatus(t, test.status)))
			if fmt.Sprint(comps) != fmt.Sprint(test.exp) {
				t.Errorf("exp %v, got %v", test.exp, comps)
			}
		})
	}
}

// pointStrings renders points as "<type>:<key>=<value>", sorted, so a test can
// compare a whole conversion at once.
func pointStrings(pts data.Points) []string {
	ret := make([]string, 0, len(pts))
	for _, p := range pts {
		v := fmt.Sprintf("%g", p.Val())
		if p.DataType == data.PointDataTypeString {
			s, err := p.ValueString()
			if err == nil {
				v = s
			}
		}
		ret = append(ret, fmt.Sprintf("%v:%v=%v", p.Type, p.Key, v))
	}
	sort.Strings(ret)
	return ret
}

func TestShellyPointsFromStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		exp    []string
	}{
		{"Plus2PM", statusPlus2PM, []string{
			"current:0=0", "current:1=0.061",
			"energy:0=1.234", "energy:1=9.9",
			"input:0=0", "input:1=1",
			"power:0=0", "power:1=7.5",
			"switch:0=0", "switch:1=1",
			"temp:0=44.4", "temp:1=44.9",
			"voltage:0=123.3", "voltage:1=123.1",
		}},
		{"PlusI4", statusPlusI4, []string{
			"input:0=0", "input:1=1", "input:2=0", "input:3=1",
		}},
		// A Plus 1 has no power monitoring, so it reports no measurements and
		// no table has to record that it does not.
		{"Plus1Addon", statusPlus1Addon, []string{
			"humidity:100=47.5",
			"input:0=0", "input:100=1",
			"switch:0=0",
			"temp:100=21.4",
		}},
		{"Plus2PMCover", statusPlus2PMCover, []string{
			"coverState:0=stopped",
			"current:0=0",
			"energy:0=3.5",
			"input:0=0", "input:1=0",
			"position:0=35",
			"power:0=0",
			"temp:0=39.1",
			"voltage:0=231.4",
		}},
		{"Pro3EM", statusPro3EM, []string{
			"apparentPower:0.a=254.1", "apparentPower:0.b=511.2",
			"apparentPower:0.c=769.3", "apparentPower:0=1534.6",
			"current:0.a=1.1", "current:0.b=2.2", "current:0.c=3.3", "current:0=6.6",
			"power:0.a=253", "power:0.b=510", "power:0.c=768", "power:0=1531",
			"powerFactor:0.a=0.99", "powerFactor:0.b=0.98", "powerFactor:0.c=0.97",
			"temp:0=41.2",
			"voltage:0.a=231", "voltage:0.b=232", "voltage:0.c=233",
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := pointStrings(shellyPointsFromStatus(mustStatus(t, test.status)))
			sort.Strings(test.exp)
			if fmt.Sprint(got) != fmt.Sprint(test.exp) {
				t.Errorf("\nexp %v\ngot %v", test.exp, got)
			}
		})
	}
}

// A NotifyStatus frame carries only what changed, so merging one must leave the
// components it does not mention as they were.
func TestShellyMergeStatus(t *testing.T) {
	status := mustStatus(t, statusPlus2PM)
	notify := mustStatus(t, `{"switch:1": {"id": 1, "output": false, "apower": 0.0}}`)

	shellyMergeStatus(status, notify)

	comps := compStrings(shellyCompsFromStatus(status))
	exp := []string{"input:0", "input:1", "switch:0", "switch:1"}
	if fmt.Sprint(comps) != fmt.Sprint(exp) {
		t.Errorf("merge changed the component list: exp %v, got %v", exp, comps)
	}

	pts := pointStrings(shellyPointsFromStatus(status))
	for _, want := range []string{
		"switch:1=0", // the frame turned switch 1 off
		"power:1=0",  // and reported its power with it
		"switch:0=0", // switch 0 was not mentioned and is unchanged
		"voltage:0=123.3",
		"input:1=1",
	} {
		found := false
		for _, p := range pts {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %v after merge, got %v", want, pts)
		}
	}
}

// The points a partial frame produces cover only the components it carries.
func TestShellyPointsFromPartialStatus(t *testing.T) {
	notify := mustStatus(t, `{"switch:1": {"id": 1, "output": true, "apower": 3.25}}`)
	got := pointStrings(shellyPointsFromStatus(notify))
	exp := []string{"power:1=3.25", "switch:1=1"}
	if fmt.Sprint(got) != fmt.Sprint(exp) {
		t.Errorf("exp %v, got %v", exp, got)
	}
}

func TestShellyDeviceInfo(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expGen   ShellyGen
		expModel string
		expID    string
	}{
		{"gen2 plug", `{"name":null,"id":"shellyplusplugus-b0b21c12ad58",
			"mac":"B0B21C12AD58","model":"SNPL-00116US","gen":2,
			"fw_id":"20230913-112003/1.14.0-gcb84623","ver":"1.14.0","app":"PlusPlugUS"}`,
			ShellyGen2, "SNPL-00116US", "shellyplusplugus-b0b21c12ad58"},
		{"gen3 dimmer", `{"id":"shelly0110dimg3-84fce63bfdf4","mac":"84FCE63BFDF4",
			"model":"S3DM-0010WW","gen":3,"ver":"1.4.4","app":"S3DM-0010WW"}`,
			3, "S3DM-0010WW", "shelly0110dimg3-84fce63bfdf4"},
		{"gen1 plug", `{"type":"SHPLG-S","mac":"C049EF8889A0","auth":false,
			"fw":"20230913-112003/v1.14.0-gcb84623","discoverable":true}`,
			ShellyGen1, "SHPLG-S", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var di shellyDeviceInfo
			if err := json.Unmarshal([]byte(test.body), &di); err != nil {
				t.Fatal(err)
			}
			if di.gen() != test.expGen {
				t.Errorf("gen: exp %v, got %v", test.expGen, di.gen())
			}
			if di.model() != test.expModel {
				t.Errorf("model: exp %v, got %v", test.expModel, di.model())
			}
			if di.ID != test.expID {
				t.Errorf("id: exp %v, got %v", test.expID, di.ID)
			}
		})
	}
}

func TestShellyHost(t *testing.T) {
	tests := []struct {
		host  string
		match bool
		id    string
	}{
		{"ShellyPlugUS-C049EF8889A0.local.", true, "C049EF8889A0"},
		{"ShellyBulbDuo-6646EB.local.", true, "6646EB"},
		{"shellyrgbw2-D93C00.local.", true, "D93C00"},
		{"shelly1pm-B91754.local.", true, "B91754"},
		{"printer.local.", false, ""},
		{"my-shelf-A1.local.", false, ""},
	}

	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			if got := shellyHost(test.host); got != test.match {
				t.Fatalf("shellyHost: exp %v, got %v", test.match, got)
			}
			if !test.match {
				return
			}
			// With no id from the device, the id comes from the hostname.
			if got := shellyDeviceID(test.host, shellyDeviceInfo{}); got != test.id {
				t.Errorf("id: exp %v, got %v", test.id, got)
			}
			// A device that reports its own id is identified by that.
			di := shellyDeviceInfo{ID: "shellyplusplugus-b0b21c12ad58"}
			if got := shellyDeviceID(test.host, di); got != di.ID {
				t.Errorf("id: exp %v, got %v", di.ID, got)
			}
		})
	}
}
