// +build !linux

package system

import "github.com/blang/semver/v4"

// ReadOSVersion returns version
func ReadOSVersion() (imgRelease semver.Version, err error) {
	return semver.Version{}, nil
}
