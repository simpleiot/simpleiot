package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// where releases are published -- the download URL serves the assets built by
// .goreleaser.yml. These are variables so tests can point at a local server.
var (
	updateLatestURL   = "https://api.github.com/repos/simpleiot/simpleiot/releases/latest"
	updateDownloadURL = "https://github.com/simpleiot/simpleiot/releases/download"
)

// name of the checksum file published with every release, set by the checksum
// section of .goreleaser.yml
const updateChecksums = "checksums.txt"

// the release information is small, but downloading a binary can take a while
// on a slow connection
var updateHTTP = &http.Client{Timeout: 10 * time.Minute}

// githubRelease is the part of the GitHub release response we use
type githubRelease struct {
	TagName string `json:"tag_name"`
}

func runUpdate(args []string, version string) {
	flags := flag.NewFlagSet("update", flag.ExitOnError)
	flagCheck := flags.Bool("check", false, "Check for a new release without installing it")

	if err := flags.Parse(args); err != nil {
		log.Fatal("error: ", err)
	}

	if err := update(version, *flagCheck); err != nil {
		log.Fatal("Error updating: ", err)
	}
}

// update replaces the running executable with the latest release published on
// GitHub. With check set, it reports what is available and installs nothing.
func update(currentVersion string, check bool) error {
	fmt.Println("Checking for updates ...")

	tag, err := latestRelease()
	if err != nil {
		return fmt.Errorf("error checking for updates: %w", err)
	}

	switch {
	case development(currentVersion):
		fmt.Printf("Running a development version (%v), the latest release is %v\n",
			currentVersion, tag)
	case strings.TrimPrefix(currentVersion, "v") == strings.TrimPrefix(tag, "v"):
		fmt.Printf("Already running the latest release (%v)\n", tag)
		return nil
	default:
		fmt.Printf("Updating from %v to %v\n", currentVersion, tag)
	}

	if check {
		return nil
	}

	execPath, err := executablePath()
	if err != nil {
		return err
	}

	err = downloadAndInstall(tag,
		assetName(tag, runtime.GOOS, runtime.GOARCH, goARM()), execPath)
	if err != nil {
		return err
	}

	fmt.Printf("Updated to %v\n", tag)
	fmt.Println("Restart Simple IoT to run the new version")

	return nil
}

// development reports whether the running binary was built from a working tree
// rather than from a release. Release builds are stamped with the version by
// goreleaser, and local builds are stamped with the output of git describe,
// which appends a commit count and hash when HEAD is past the last tag.
func development(version string) bool {
	if version == "" || version == "Development" {
		return true
	}

	return strings.Contains(version, "-g")
}

// latestRelease returns the tag of the most recent release, which excludes
// drafts and pre-releases
func latestRelease() (string, error) {
	resp, err := httpGet(updateLatestURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release githubRelease

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("error reading release information: %w", err)
	}

	if release.TagName == "" {
		return "", errors.New("no release tag found")
	}

	return release.TagName, nil
}

// assetName returns the release asset for a platform, matching the archive
// name_template in .goreleaser.yml
func assetName(tag, goos, goarch, goarm string) string {
	platform := goos
	if platform == "darwin" {
		platform = "macos"
	}

	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm":
		goarch += goarm
	}

	name := fmt.Sprintf("simpleiot-%v-%v-%v", tag, platform, goarch)

	if goos == "windows" {
		name += ".exe"
	}

	return name
}

// goARM returns the ARM architecture version the running binary was built for.
// The runtime package does not expose GOARM, but the build information does.
func goARM() string {
	if runtime.GOARCH != "arm" {
		return ""
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "GOARM" {
				// newer toolchains allow a floating point suffix, as
				// in "7,softfloat"
				v, _, _ := strings.Cut(s.Value, ",")
				return v
			}
		}
	}

	// armv6 binaries also run on armv7 hardware, so it is the safe default
	return "6"
}

// executablePath returns the location of the running executable with any
// symlinks resolved, so an update replaces the binary rather than a link to it
func executablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error finding the running executable: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("error resolving %v: %w", execPath, err)
	}

	return execPath, nil
}

// downloadAndInstall fetches a release asset, verifies it against the published
// checksum, and moves it into place at execPath
func downloadAndInstall(tag, asset, execPath string) error {
	sum, err := releaseChecksum(tag, asset)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("%v/%v/%v", updateDownloadURL, tag, asset)

	fmt.Println("Downloading", u)

	resp, err := httpGet(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// the download goes next to the current executable so it can be moved
	// into place without crossing a filesystem boundary
	tmp, err := os.CreateTemp(filepath.Dir(execPath), "siot-update-*")
	if err != nil {
		return fmt.Errorf("error creating a temporary file: %w", err)
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	h := sha256.New()

	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("error downloading %v: %w", asset, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("error writing %v: %w", tmpPath, err)
	}

	if got := hex.EncodeToString(h.Sum(nil)); got != sum {
		return fmt.Errorf("checksum mismatch for %v: expected %v, downloaded %v",
			asset, sum, got)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("error making %v executable: %w", tmpPath, err)
	}

	return replaceExecutable(tmpPath, execPath)
}

// releaseChecksum returns the SHA-256 sum published for a release asset
func releaseChecksum(tag, asset string) (string, error) {
	resp, err := httpGet(fmt.Sprintf("%v/%v/%v", updateDownloadURL, tag, updateChecksums))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseChecksums(resp.Body, asset)
}

// parseChecksums finds the sum for an asset in the checksums file, which has
// one "<sum>  <asset>" line per release asset
func parseChecksums(r io.Reader, asset string) (string, error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == asset {
			return fields[0], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading %v: %w", updateChecksums, err)
	}

	return "", fmt.Errorf("no checksum published for %v", asset)
}

// replaceExecutable moves the downloaded binary over the running one. Windows
// does not allow an executable that is running to be written, but it does allow
// it to be renamed, so the current binary is moved aside first.
func replaceExecutable(newPath, execPath string) error {
	previous := execPath + ".old"
	_ = os.Remove(previous)

	if err := os.Rename(execPath, previous); err != nil {
		return fmt.Errorf("error moving %v aside: %w", execPath, err)
	}

	if err := os.Rename(newPath, execPath); err != nil {
		// put the current version back so the installation is left as
		// it was found
		if e := os.Rename(previous, execPath); e != nil {
			return fmt.Errorf("error installing %v: %w (the previous version is at %v)",
				execPath, err, previous)
		}

		return fmt.Errorf("error installing %v: %w", execPath, err)
	}

	// Windows holds the file open while it runs, so removing it can fail
	// here; the next update cleans it up
	_ = os.Remove(previous)

	return nil
}

func httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "simpleiot/"+version)

	resp, err := updateHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %v: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("error fetching %v: %v", url, resp.Status)
	}

	return resp, nil
}
