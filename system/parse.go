package system

import (
	"errors"
	"regexp"

	"github.com/blang/semver/v4"
)

// This regex will parse VERSION_ID=1.2 or VERSION_ID="1.2.3" just as easily
var reExtractVersionID = regexp.MustCompile(`VERSION_ID=['"]?([^'"\s]*)`)

func parseVersion(releaseFile []byte) (ver semver.Version, err error) {
	// Now parse the file to get the VERSION_ID version info
	versionInfo := reExtractVersionID.FindSubmatch(releaseFile)
	if versionInfo == nil {
		err = errors.New("VERSION_ID not found in version file")
		return
	}
	return semver.ParseTolerant(string(versionInfo[1]))
}
