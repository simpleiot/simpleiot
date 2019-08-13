package isdb

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

func TestConfig(t *testing.T) {
	config := isdata.Config{
		ID:             "123",
		HighWindowPerc: 23.23,
		LowWindowPerc:  99.23,
	}

	os.Mkdir("./temp", os.ModePerm)

	db, err := NewDb("./temp")

	if err != nil {
		t.Error("failed to open db: ", err)
	}

	err = db.WriteConfig(&config)

	if err != nil {
		t.Error("failed writing config: ", err)
	}

	configR := isdata.Config{}
	err = db.ReadConfig(&configR)

	if err != nil {
		t.Error("failed reading config: ", err)
	}

	if !reflect.DeepEqual(config, configR) {
		t.Errorf("read config does not match: %+v\n", configR)
	}

	err = db.store.Close()

	if err != nil {
		t.Error("failed closing database: ", err)
	}
}

func TestSamples(t *testing.T) {
	sample := data.Sample{
		Type:  isdata.SampleTypeFlowWindowAvg,
		Value: 8888.88,
		Min:   4444.44,
		Attributes: map[string]float64{
			"inputInjector":  100.00,
			"inputWaterOn":   200.00,
			"inputIrrigator": 300.00,
		},
	}

	err := os.RemoveAll("./temp")

	if err != nil {
		t.Error("failed to remove file: ", err)
	}

	os.Mkdir("./temp", os.ModePerm)

	db, err := NewDb("./temp")

	if err != nil {
		t.Error("failed to open db: ", err)
	}

	for i := 0; i <= 10; i++ {
		sample.Time = time.Now()
		err = db.WriteSample(sample)

		if err != nil {
			t.Error("failed writing samples: ", err)
		}
	}

	samplesR, err := db.ReadSamples()

	if err != nil {
		t.Error("failed reading samples: ", err)
	}

	for _, sampleR := range samplesR {

		// All of the times are different
		/*if sample.Time.Sub(sampleR.Time) >= time.Duration(time.Nanosecond) {
			t.Errorf("read sample time does not match:\n // %+v\n || %+v\n", sample.Time, sampleR.Time)
		}*/

		if !reflect.DeepEqual(sample.Value, sampleR.Value) {
			t.Errorf("read sample value does not match: %+v\n", sampleR.Value)
		}

		if !reflect.DeepEqual(sample.Min, sampleR.Min) {
			t.Errorf("read sample min does not match: %+v\n", sampleR.Min)
		}

		if !reflect.DeepEqual(sample.Attributes, sampleR.Attributes) {
			t.Errorf("read sample attributes does not match: %+v\n", sampleR.Attributes)
		}
	}

	err = db.store.Close()

	if err != nil {
		t.Error("failed closing database: ", err)
	}
}

func TestFaultHist(t *testing.T) {
	fault := isdata.Fault{
		Fault: isdata.FaultTypeIrrOff,
		Time:  time.Now(),
	}

	err := os.RemoveAll("./temp")

	if err != nil {
		t.Error("failed to remove file: ", err)
	}

	os.Mkdir("./temp", os.ModePerm)

	db, err := NewDb("./temp")

	if err != nil {
		t.Error("failed to open db: ", err)
	}

	for i := 0; i <= 10; i++ {
		err = db.WriteFaultHist(fault)

		if err != nil {
			t.Error("failed writing fault: ", err)
		}
	}

	faultsR, err := db.ReadFaultHist()

	if err != nil {
		t.Error("failed reading faults: ", err)
	}
	for _, faultR := range faultsR {

		if fault.Time.Sub(faultR.Time) >= time.Duration(time.Nanosecond) {
			t.Errorf("read time does not match:\n // %+v\n || %+v\n", fault, faultR)
		}
	}

	err = db.store.Close()

	if err != nil {
		t.Error("failed closing database: ", err)
	}
}
