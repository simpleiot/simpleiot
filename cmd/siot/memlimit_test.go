package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCgroupMemoryLimit(t *testing.T) {
	tests := []struct {
		name   string
		proc   string
		files  map[string]string
		expect int64
	}{
		{
			name: "v2 unit limit",
			proc: "0::/system.slice/siot.service\n",
			files: map[string]string{
				"memory.max":                           "max\n",
				"system.slice/memory.max":              "max\n",
				"system.slice/siot.service/memory.max": "471859200\n",
			},
			expect: 471859200,
		},
		{
			name: "v2 slice limit lower than unit",
			proc: "0::/system.slice/siot.service\n",
			files: map[string]string{
				"system.slice/memory.max":              "268435456\n",
				"system.slice/siot.service/memory.max": "471859200\n",
			},
			expect: 268435456,
		},
		{
			name: "v2 no limit",
			proc: "0::/system.slice/siot.service\n",
			files: map[string]string{
				"system.slice/siot.service/memory.max": "max\n",
			},
			expect: 0,
		},
		{
			name: "v2 container namespace",
			proc: "0::/\n",
			files: map[string]string{
				"memory.max": "536870912\n",
			},
			expect: 536870912,
		},
		{
			name: "v1 limit",
			proc: "12:cpu,cpuacct:/system.slice/siot.service\n" +
				"4:memory:/system.slice/siot.service\n0::/system.slice/siot.service\n",
			files: map[string]string{
				"memory/system.slice/siot.service/memory.limit_in_bytes": "471859200\n",
			},
			expect: 471859200,
		},
		{
			name: "v1 no limit",
			proc: "4:memory:/system.slice/siot.service\n",
			files: map[string]string{
				"memory/system.slice/siot.service/memory.limit_in_bytes": "9223372036854771712\n",
			},
			expect: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			root := filepath.Join(dir, "cgroup")
			proc := filepath.Join(dir, "proc-cgroup")
			writeFile(t, proc, tc.proc)
			for p, c := range tc.files {
				writeFile(t, filepath.Join(root, p), c)
			}
			if got := cgroupMemoryLimit(proc, root); got != tc.expect {
				t.Errorf("got %d, expected %d", got, tc.expect)
			}
		})
	}
}

func TestCgroupMemoryLimitMissing(t *testing.T) {
	dir := t.TempDir()
	if got := cgroupMemoryLimit(filepath.Join(dir, "none"), dir); got != 0 {
		t.Errorf("got %d, expected 0", got)
	}
}
