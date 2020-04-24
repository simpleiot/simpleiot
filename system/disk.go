package system

import "syscall"

// FsPercentUsed returns % of storage used for mountPoint
func FsPercentUsed(mountPoint string) (float64, error) {
	buf := new(syscall.Statfs_t)
	err := syscall.Statfs(mountPoint, buf)
	if err != nil {
		return 0, err
	}
	return (float64(buf.Blocks-buf.Bfree) / float64(buf.Blocks)) * 100, nil
}
