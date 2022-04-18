package system

import (
	"fmt"
	"regexp"

	"github.com/blang/semver/v4"
)

func parseVersion(releaseFile []byte, field string) (ver semver.Version, err error) {
	// This regex will parse VERSION_ID=1.2 or VERSION_ID="1.2.3" just as easily
	re := field + `=['"]?([^'"\s]*)`
	reCompiled, err := regexp.Compile(re)

	if err != nil {
		return semver.Version{}, fmt.Errorf("regex compile failed: %v", err)
	}

	// Now parse the file to get the version info
	versionInfo := reCompiled.FindSubmatch(releaseFile)
	if versionInfo == nil {
		err = fmt.Errorf("field %v not found", field)
		return
	}
	return semver.ParseTolerant(string(versionInfo[1]))
}
