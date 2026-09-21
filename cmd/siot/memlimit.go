package main

import (
	"bufio"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
)

// memLimitRatio is the share of the cgroup memory limit given to the Go
// runtime as its soft limit. The rest covers memory the runtime does not
// count, such as the page cache the cgroup also charges.
const memLimitRatio = 0.9

// setMemoryLimit gives the Go runtime a soft memory limit derived from the
// cgroup the process runs in, such as a systemd unit's MemoryMax, so the
// garbage collector works harder as memory nears the limit rather than the
// process being killed on reaching it. GOMEMLIMIT, when set, takes precedence.
func setMemoryLimit() {
	if os.Getenv("GOMEMLIMIT") != "" {
		return
	}

	limit := cgroupMemoryLimit("/proc/self/cgroup", "/sys/fs/cgroup")
	if limit <= 0 {
		return
	}

	soft := int64(float64(limit) * memLimitRatio)
	debug.SetMemoryLimit(soft)
	log.Printf("Memory: cgroup limit %d MiB, Go memory limit %d MiB\n",
		limit>>20, soft>>20)
}

// cgroupMemoryLimit returns the memory limit in bytes of the cgroup the
// process belongs to, or 0 when it has none. procCgroup is the process's
// cgroup membership file and root the cgroup filesystem mount.
func cgroupMemoryLimit(procCgroup, root string) int64 {
	f, err := os.Open(procCgroup)
	if err != nil {
		return 0
	}
	defer f.Close()

	var v2Path, v1Path string
	haveV1 := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		// hierarchy-ID:controller-list:cgroup-path
		fields := strings.SplitN(s.Text(), ":", 3)
		if len(fields) != 3 {
			continue
		}
		if fields[0] == "0" && fields[1] == "" {
			v2Path = fields[2]
		}
		for _, c := range strings.Split(fields[1], ",") {
			if c == "memory" {
				v1Path = fields[2]
				haveV1 = true
			}
		}
	}

	if haveV1 {
		return cgroupV1Limit(root, v1Path)
	}
	if v2Path != "" {
		return cgroupV2Limit(root, v2Path)
	}
	return 0
}

// cgroupV2Limit returns the lowest memory.max from the process's cgroup up to
// the root, since a limit on a parent, such as a slice, applies to every
// cgroup below it.
func cgroupV2Limit(root, cgPath string) int64 {
	var limit int64
	dir := filepath.Join(root, cgPath)
	for {
		if l := readLimit(filepath.Join(dir, "memory.max")); l > 0 &&
			(limit == 0 || l < limit) {
			limit = l
		}
		if dir == root || !strings.HasPrefix(dir, root) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return limit
}

// cgroupV1Limit reads memory.limit_in_bytes, falling back to the top of the
// memory hierarchy for a container whose cgroup path is not visible inside it.
func cgroupV1Limit(root, cgPath string) int64 {
	for _, p := range []string{
		filepath.Join(root, "memory", cgPath, "memory.limit_in_bytes"),
		filepath.Join(root, "memory", "memory.limit_in_bytes"),
	} {
		if l := readLimit(p); l > 0 {
			return l
		}
	}
	return 0
}

// readLimit parses a cgroup memory limit file, returning 0 for no limit. Cgroup
// v2 writes "max"; v1 writes a number near the largest int64.
func readLimit(p string) int64 {
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || v <= 0 || v >= math.MaxInt64/2 {
		return 0
	}
	return v
}
