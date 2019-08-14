package file

import (
	"os"
	"os/exec"
)

// Exists returns true if file exists, else false
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

// SyncDisks runs the linux "sync" command
func SyncDisks() error {
	return exec.Command("sync").Run()
}
