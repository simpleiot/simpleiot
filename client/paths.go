package client

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Several clients build a file path from a point: an IIO device or channel,
// a OneWire sensor ID, a rule's audio file, and the name of a file sent to
// a device. These checks keep such a value from naming a file outside the
// directory the client expects.

// safeName checks that a value is one path element: not empty, not "." or
// "..", and free of separators and NUL.
func safeName(name string) error {
	switch {
	case name == "", name == ".", name == "..":
		return fmt.Errorf("invalid name %q", name)
	case strings.ContainsAny(name, `/\`+"\x00"):
		return fmt.Errorf("name %q must not contain a path separator", name)
	}
	return nil
}

// safePath checks a value that may be either one path element or an
// absolute path. An absolute path must already be clean, which rules out
// ".." and repeated separators; a relative path with a separator in it is
// refused, since it would be resolved against whatever the working
// directory happens to be.
func safePath(p string) error {
	if !strings.ContainsRune(p, filepath.Separator) {
		return safeName(p)
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("path %q must be absolute", p)
	}
	if strings.ContainsRune(p, 0) || filepath.Clean(p) != p {
		return fmt.Errorf("path %q must be a clean absolute path", p)
	}
	return nil
}
