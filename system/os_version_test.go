package system

import (
	"fmt"
	"testing"

	"github.com/blang/semver/v4"
)

func TestReadOSRelease(t *testing.T) {
	rel, err := ReadOSVersion()
	// If there is no error, we have a problem
	if err != nil {
		t.Error("Error reading version: ", err)
	}

	t.Log("/etc/os-release contains a valid version?", rel)

	v, err := parseVersion([]byte("VERSION_ID=\"1.2\"\nTesting with quotes"))
	exp := semver.Version{
		Major: 1,
		Minor: 2,
		Patch: 0,
	}

	if v.NE(exp) {
		fmt.Printf("got %+v\n", v)
		t.Error("Did not get expected version")
	}

	v, err = parseVersion([]byte("VERSION_ID=1.2.352\nTesting without quotes"))
	exp = semver.Version{
		Major: 1,
		Minor: 2,
		Patch: 352,
	}

	if v.NE(exp) {
		fmt.Printf("got %+v\n", v)
		t.Error("Did not get expected version")
	}
}
