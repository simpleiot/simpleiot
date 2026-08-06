package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// the asset names here are the ones .goreleaser.yml publishes -- if the
// name_template changes, these need to change with it
func TestAssetName(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		goarm  string
		expect string
	}{
		{"linux", "amd64", "", "simpleiot-v0.19.0-linux-x86_64"},
		{"linux", "arm64", "", "simpleiot-v0.19.0-linux-arm64"},
		{"linux", "arm", "6", "simpleiot-v0.19.0-linux-arm6"},
		{"linux", "arm", "7", "simpleiot-v0.19.0-linux-arm7"},
		{"linux", "riscv64", "", "simpleiot-v0.19.0-linux-riscv64"},
		{"darwin", "amd64", "", "simpleiot-v0.19.0-macos-x86_64"},
		{"darwin", "arm64", "", "simpleiot-v0.19.0-macos-arm64"},
		{"windows", "amd64", "", "simpleiot-v0.19.0-windows-x86_64.exe"},
		{"windows", "arm64", "", "simpleiot-v0.19.0-windows-arm64.exe"},
	}

	for _, test := range tests {
		got := assetName("v0.19.0", test.goos, test.goarch, test.goarm)
		if got != test.expect {
			t.Errorf("%v/%v%v: expected %v, got %v",
				test.goos, test.goarch, test.goarm, test.expect, got)
		}
	}
}

func TestParseChecksums(t *testing.T) {
	checksums := `7b9156ae2146a42cd0dd3549771b1e4e755eeb980746e3c7fe29d69e715f3611  simpleiot-v0.19.0-linux-arm6
533e0e3a05992a48cf94bbf39420b414b393086060e09d83971eac66fc03a5bc  simpleiot-v0.19.0-linux-arm64
35b6cc4f3e720e4e08129a6aff9f33a58a9ec527f514453fe2fe62b544a0d8ba  simpleiot-v0.19.0-linux-x86_64
`

	sum, err := parseChecksums(strings.NewReader(checksums), "simpleiot-v0.19.0-linux-arm64")
	if err != nil {
		t.Fatal("Error parsing checksums: ", err)
	}

	expect := "533e0e3a05992a48cf94bbf39420b414b393086060e09d83971eac66fc03a5bc"
	if sum != expect {
		t.Errorf("expected %v, got %v", expect, sum)
	}

	// an asset that is not in the list is an error rather than an empty sum
	_, err = parseChecksums(strings.NewReader(checksums), "simpleiot-v0.19.0-macos-arm64")
	if err == nil {
		t.Error("expected an error for an asset with no published checksum")
	}
}

// serveRelease starts a server that publishes a release the way GitHub does,
// and points the download URL at it
func serveRelease(t *testing.T, tag, asset string, binary, sum []byte) {
	t.Helper()

	checksums := fmt.Sprintf("%v  %v\n", hex.EncodeToString(sum), asset)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + tag + "/" + updateChecksums:
			_, _ = w.Write([]byte(checksums))
		case "/" + tag + "/" + asset:
			_, _ = w.Write(binary)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	downloadURL := updateDownloadURL
	updateDownloadURL = server.URL
	t.Cleanup(func() { updateDownloadURL = downloadURL })
}

func TestDownloadAndInstall(t *testing.T) {
	binary := []byte("the new version")
	sum := sha256.Sum256(binary)

	serveRelease(t, "v0.19.0", "simpleiot-v0.19.0-linux-x86_64", binary, sum[:])

	execPath := filepath.Join(t.TempDir(), "siot")
	if err := os.WriteFile(execPath, []byte("the running version"), 0755); err != nil {
		t.Fatal("Error creating test executable: ", err)
	}

	err := downloadAndInstall("v0.19.0", "simpleiot-v0.19.0-linux-x86_64", execPath)
	if err != nil {
		t.Fatal("Error installing update: ", err)
	}

	installed, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal("Error reading installed binary: ", err)
	}

	if string(installed) != string(binary) {
		t.Errorf("expected %q to be installed, got %q", binary, installed)
	}

	info, err := os.Stat(execPath)
	if err != nil {
		t.Fatal("Error checking installed binary: ", err)
	}

	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("installed binary is not executable: %v", info.Mode())
	}
}

// a download that does not match the published checksum is discarded rather
// than installed
func TestDownloadChecksumMismatch(t *testing.T) {
	sum := sha256.Sum256([]byte("what the release should contain"))

	serveRelease(t, "v0.19.0", "simpleiot-v0.19.0-linux-x86_64",
		[]byte("something else entirely"), sum[:])

	execPath := filepath.Join(t.TempDir(), "siot")
	running := []byte("the running version")

	if err := os.WriteFile(execPath, running, 0755); err != nil {
		t.Fatal("Error creating test executable: ", err)
	}

	err := downloadAndInstall("v0.19.0", "simpleiot-v0.19.0-linux-x86_64", execPath)
	if err == nil {
		t.Fatal("expected an error for a checksum mismatch")
	}

	installed, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal("Error reading executable: ", err)
	}

	if string(installed) != string(running) {
		t.Errorf("the running version was replaced with %q", installed)
	}
}

func TestDevelopmentVersion(t *testing.T) {
	tests := []struct {
		version string
		expect  bool
	}{
		// the default when no version is stamped in
		{"Development", true},
		// git describe output for a tree past the last tag
		{"v0.18.5-42-gb3ee632a", true},
		// what goreleaser stamps into a release build
		{"0.18.5", false},
		// git describe output for a tree at the tag
		{"v0.18.5", false},
	}

	for _, test := range tests {
		if got := development(test.version); got != test.expect {
			t.Errorf("%v: expected %v, got %v", test.version, test.expect, got)
		}
	}
}
