package version

import (
	"io/ioutil"

	"github.com/blang/semver"
)

const releaseFilePath = "/etc/os-release"

// ReadOSVersion reads `releaseFilePath` and parses VERSION_ID into a `Version` struct
func ReadOSVersion() (imgRelease semver.Version, err error) {
	// Read `releaseFilePath` into []byte
	data, err := ioutil.ReadFile(releaseFilePath)
	if err != nil {
		return
	}

	imgRelease, err = parseVersion(data)
	return
}
