package file

import "os"

// Exists returns true if file exists, else false
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}
