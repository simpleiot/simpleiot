// +build !linux

package version

import "github.com/blang/semver"

// ReadOSVersion returns version
func ReadOSVersion() (imgRelease semver.Version, err error) {
	return semver.Version{}, nil
}
