//go:build !linux

package system

import "github.com/blang/semver/v4"

// ReadOSVersion returns version
func ReadOSVersion(field string) (imgRelease semver.Version, err error) {
	return semver.Version{}, nil
}
