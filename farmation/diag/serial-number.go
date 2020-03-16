package diag

import (
	"errors"

	"github.com/simpleiot/simpleiot/farmation/isdb"
)

type diagSn struct{}

func (d diagSn) String() string {
	return "serial-number"
}

func (d diagSn) Run() (ret error) {
	sn, err := isdb.ReadSerialNumber()
	if err != nil || sn == "" {
		sn := GetEnter("Please enter unit serial number")
		err := isdb.WriteSerialNumber(sn)
		if err != nil {
			return err
		}
		sn, err = isdb.ReadSerialNumber()
		if err != nil {
			return err
		}
		if sn == "" {
			return errors.New("SN is blank")
		}
	}
	return nil
}

func init() {
	Register(diagSn{})
}
