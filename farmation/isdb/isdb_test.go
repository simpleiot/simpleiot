package isdb

import (
	"os"
	"reflect"
	"testing"
	"time"

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
		if !reflect.DeepEqual(fault, faultR) {
			t.Errorf("read config does not match:\n // %+v\n || %+v\n", fault, faultR)
		}
	}

	err = db.store.Close()

	if err != nil {
		t.Error("failed closing database: ", err)
	}
}
