package system

import (
	"fmt"
	"io/ioutil"

	"github.com/blang/semver/v4"
)

const releaseFilePath = "/etc/os-release"

// ReadOSVersion reads `releaseFilePath` and parses VERSION_ID into a `Version` struct
func ReadOSVersion(field string) (imgRelease semver.Version, err error) {
	// Read `releaseFilePath` into []byte
	data, err := ioutil.ReadFile(releaseFilePath)
	if err != nil {
		return
	}

	imgRelease, err = parseVersion(data, field)

	if err != nil {
		err = fmt.Errorf("searching %v, got: %v", releaseFilePath, err)
	}
	return
}
