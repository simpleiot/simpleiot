package isdb

import (
	"os"
	"reflect"
	"testing"

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
}
