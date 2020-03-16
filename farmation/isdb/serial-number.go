package isdb

import "io/ioutil"

var snFile = "/boot/serial-number"

// ReadSerialNumber returns the unit serial number
func ReadSerialNumber() (string, error) {
	sn, err := ioutil.ReadFile(snFile)
	return string(sn), err
}

// WriteSerialNumber to unit storage
func WriteSerialNumber(sn string) error {
	return ioutil.WriteFile(snFile, []byte(sn), 600)
}
