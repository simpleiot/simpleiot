package client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/nats-io/nats.go"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/system"
)

// Update represents the config of a metrics node type
type Update struct {
	ID              string   `node:"id"`
	Parent          string   `node:"parent"`
	Description     string   `point:"description"`
	VersionOS       string   `point:"versionOS"`
	URI             string   `point:"uri"`
	OSUpdates       []string `point:"osUpdate"`
	DownloadOS      string   `point:"downloadOS"`
	OSDownloaded    string   `point:"osDownloaded"`
	DiscardDownload string   `point:"discardDownload"`
	Prefix          string   `point:"prefix"`
	Directory       string   `point:"directory"`
	PollPeriod      int      `point:"pollPeriod"`
	Refresh         bool     `point:"refresh"`
	AutoDownload    bool     `point:"autoDownload"`
	AutoReboot      bool     `point:"autoReboot"`
}

// UpdateClient is a SIOT client used to collect system or app metrics
type UpdateClient struct {
	log           *log.Logger
	nc            *nats.Conn
	config        Update
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewUpdateClient ...
func NewUpdateClient(nc *nats.Conn, config Update) Client {
	return &UpdateClient{
		log:           log.New(os.Stderr, "Update: ", log.LstdFlags|log.Lmsgprefix),
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

func (m *UpdateClient) setError(err error) {
	errS := ""
	if err != nil {
		errS = err.Error()
		m.log.Println(err)
	}

	p := data.NewPointString(data.PointTypeError, "", errS)

	e := SendNodePoint(m.nc, m.config.ID, p, true)
	if e != nil {
		m.log.Println("error sending point:", e)
	}
}

var reUpd = regexp.MustCompile(`(.*)_(\d+\.\d+\.\d+)\.upd`)

// Limits on what the update client fetches. Update images can be large, so
// the download is given a long timeout, but not an unbounded one.
const (
	updateListTimeout     = 30 * time.Second
	updateListMaxBytes    = 1 << 20 // 1 MiB
	updateDownloadTimeout = time.Hour

	// updateDownloadMaxBytes caps an update image. It is typed because
	// the value does not fit an int on a 32-bit target. 1 GiB is above
	// any image the Yoe updater takes and below the storage on the
	// devices that run it, so the cap is reached before the disk is.
	updateDownloadMaxBytes int64 = 1 << 30 // 1 GiB

	// updateDownloadReserveBytes is free space left on the destination
	// filesystem after a download. Filling the disk is how a download
	// that is too large takes the rest of the system with it, so a
	// download that would not leave this much does not start.
	updateDownloadReserveBytes uint64 = 64 << 20 // 64 MiB
)

// updateURL joins name onto the update server URI. The URI must use https,
// since the update is not signed and would otherwise be replaceable on
// the wire.
func updateURL(base, name string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("URI error: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("update URI must use https: %v", base)
	}
	if u.Host == "" {
		return "", fmt.Errorf("update URI has no host: %v", base)
	}
	return u.JoinPath(name).String(), nil
}

// updateFetcher performs the HTTP requests for the update client. Tests
// supply an http.Client that trusts their server, and smaller limits than
// a real download would need.
type updateFetcher struct {
	list     *http.Client
	download *http.Client
	maxBytes int64
	reserve  uint64
}

func newUpdateFetcher() updateFetcher {
	return updateFetcher{
		list:     &http.Client{Timeout: updateListTimeout},
		download: &http.Client{Timeout: updateDownloadTimeout},
		maxBytes: updateDownloadMaxBytes,
		reserve:  updateDownloadReserveBytes,
	}
}

// checkSpace reports whether a download of size bytes fits in dir and
// still leaves the reserve free. A size of -1 means the server did not
// say how large the image is, and only the reserve is required, so a
// device that is already nearly full does not start a download that
// cannot finish.
func (f updateFetcher) checkSpace(dir string, size int64) error {
	usage, err := disk.Usage(dir)
	if err != nil {
		return fmt.Errorf("error checking free space in %v: %w", dir, err)
	}

	need := f.reserve
	if size > 0 {
		need += uint64(size)
	}

	if usage.Free < need {
		return fmt.Errorf("not enough space in %v: %v bytes free, %v needed",
			dir, usage.Free, need)
	}

	return nil
}

// get fetches u and returns the response after checking its status.
func (f updateFetcher) get(c *http.Client, u string) (*http.Response, error) {
	resp, err := c.Get(u)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%v: %v", u, resp.Status)
	}
	return resp, nil
}

// fetchList returns the lines of files.txt on the update server.
func (f updateFetcher) fetchList(base string) ([]string, error) {
	u, err := updateURL(base, "files.txt")
	if err != nil {
		return nil, err
	}
	resp, err := f.get(f.list, u)
	if err != nil {
		return nil, fmt.Errorf("error getting updates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, updateListMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("error reading http response: %w", err)
	}
	if len(body) > updateListMaxBytes {
		return nil, fmt.Errorf("files.txt is larger than %v bytes", updateListMaxBytes)
	}

	return strings.Split(string(body), "\n"), nil
}

// fetchFile downloads name from the update server into dir. A file that
// is empty, fails to download, or is larger than the cap is removed.
func (f updateFetcher) fetchFile(base, name, dir string) error {
	u, err := updateURL(base, name)
	if err != nil {
		return err
	}

	destPath := filepath.Join(dir, filepath.Base(name))

	resp, err := f.get(f.download, u)
	if err != nil {
		return fmt.Errorf("error fetching OS update: %w", err)
	}
	defer resp.Body.Close()

	// A server that declares a size too large is refused before anything
	// is written. The length is the server's claim, so the limit on the
	// copy below is what enforces the cap.
	if resp.ContentLength > f.maxBytes {
		return fmt.Errorf("failed to download %v: update is %v bytes, over the %v byte limit",
			u, resp.ContentLength, f.maxBytes)
	}

	if err := f.checkSpace(dir, resp.ContentLength); err != nil {
		return fmt.Errorf("failed to download %v: %w", u, err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating OS update file: %w", err)
	}

	c, err := io.Copy(out, io.LimitReader(resp.Body, f.maxBytes+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && c > f.maxBytes {
		err = fmt.Errorf("update is larger than %v bytes", f.maxBytes)
	}
	if err == nil && c <= 0 {
		err = errors.New("empty download")
	}
	if err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to download %v: %w", u, err)
	}

	return nil
}

// Run the main logic for this client and blocks until stopped
func (m *UpdateClient) Run() error {
	cDownloadFinished := make(chan struct{})
	// cSetError is used in any goroutines
	cSetError := make(chan error)

	fetch := newUpdateFetcher()

	download := func(v string) error {
		defer func() {
			cDownloadFinished <- struct{}{}
			_ = SendNodePoint(m.nc, m.config.ID,
				data.NewPointString(data.PointTypeDownloadOS, "", ""),
				false,
			)
			m.config.DownloadOS = ""
		}()

		name := m.config.Prefix + "_" + v + ".upd"
		m.log.Println("Downloading update: ", name)
		return fetch.fetchFile(m.config.URI, name, m.config.Directory)
	}

	// fill in default prefix
	if m.config.Prefix == "" {
		p, err := os.Hostname()
		if err != nil {
			m.log.Println("Error getting hostname: ", err)
		} else {
			m.log.Println("Setting update prefix to: ", p)
			err := SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypePrefix, "0", p), false)
			if err != nil {
				m.log.Println("Error sending point: ", err)
			} else {
				m.config.Prefix = p
			}
		}
	}

	if m.config.Directory == "" {
		d := "/data"
		m.log.Println("Setting directory to: ", d)
		err := SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypeDirectory, "0", d), false)
		if err != nil {
			m.log.Println("Error sending point: ", err)
		} else {
			m.config.Directory = d
		}
	}

	if m.config.PollPeriod <= 0 {
		p := 30
		m.log.Println("Setting poll period to: ", p)
		err := SendNodePoint(m.nc, m.config.ID, data.NewPointFloat(data.PointTypePollPeriod, "0", float64(p)), false)
		if err != nil {
			m.log.Println("Error sending point: ", err)
		} else {
			m.config.PollPeriod = p
		}
	}

	getUpdates := func() error {
		clearUpdateList := func() {
			cnt := len(m.config.OSUpdates)

			if cnt > 0 {
				pts := data.Points{}
				for i := 0; i < cnt; i++ {
					pts = append(pts, data.Point{
						Time: time.Now(), Type: data.PointTypeOSUpdate, Key: strconv.Itoa(i), Tombstone: 1,
					})
				}

				err := SendNodePoints(m.nc, m.config.ID, pts, false)
				if err != nil {
					m.log.Println("Error sending version points: ", err)
				}
			}
		}

		updates, err := fetch.fetchList(m.config.URI)
		if err != nil {
			clearUpdateList()
			return err
		}

		updates = slices.DeleteFunc(updates, func(u string) bool {
			return !strings.HasPrefix(u, m.config.Prefix)
		})

		versions := semver.Versions{}

		for _, u := range updates {
			matches := reUpd.FindStringSubmatch(u)
			if len(matches) > 1 {
				prefix := matches[1]
				version := matches[2]
				sv, err := semver.Parse(version)
				if err != nil {
					m.log.Printf("Error parsing version %v: %v\n", version, err)
				}
				if prefix == m.config.Prefix {
					versions = append(versions, sv)
				}
			} else {
				m.log.Println("Version not found in filename: ", u)
			}
		}

		sort.Sort(versions)

		// need to update versions available
		pts := data.Points{}
		now := time.Now()
		for i, v := range versions {
			pts = append(pts, data.NewPointString(data.PointTypeOSUpdate, strconv.Itoa(i), v.String()))
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			m.log.Println("Error sending version points: ", err)

		}

		err = data.MergePoints(m.config.ID, pts, &m.config)
		if err != nil {
			log.Println("error merging new points:", err)
		}

		underflowCount := len(m.config.OSUpdates) - len(versions)

		if underflowCount > 0 {
			pts := data.Points{}
			for i := len(versions); i < len(versions)+underflowCount; i++ {
				pts = append(pts, data.Point{
					Time: now, Type: data.PointTypeOSUpdate, Key: strconv.Itoa(i), Tombstone: 1,
				})
			}

			err = SendNodePoints(m.nc, m.config.ID, pts, false)
			if err != nil {
				m.log.Println("Error sending version points: ", err)
			}
		}
		return nil
	}

	cleanDownloads := func() error {
		files, err := os.ReadDir(m.config.Directory)
		var errRet error
		if err != nil {
			return fmt.Errorf("error getting files in data dir: %w", err)
		}

		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".upd" {
				p := filepath.Join(m.config.Directory, file.Name())
				err = os.Remove(p)
				if err != nil {
					m.log.Printf("Error removing %v: %v\n", file.Name(), err)
					errRet = err
				}
			}
		}

		m.config.OSDownloaded = ""
		err = SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypeOSDownloaded, "0", ""), true)
		if err != nil {
			m.log.Println("Error clearing downloaded point: ", err)
		}

		err = SendNodePoints(m.nc, m.config.ID, data.Points{
			data.NewPointFloat(data.PointTypeDiscardDownload, "", 0),
		}, true)
		if err != nil {
			m.log.Println("Error discarding download: ", err)
		}

		return errRet
	}

	checkDownloads := func() error {
		files, err := os.ReadDir(m.config.Directory)
		if err != nil {
			return fmt.Errorf("error getting files in data dir: %w", err)
		}

		updFiles := []string{}
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".upd" {
				updFiles = append(updFiles, file.Name())
			}
		}

		versions := semver.Versions{}
		for _, u := range updFiles {

			matches := reUpd.FindStringSubmatch(u)
			if len(matches) > 1 {
				prefix := matches[1]
				version := matches[2]
				sv, err := semver.Parse(version)
				if err != nil {
					m.log.Printf("Error parsing version %v: %v\n", version, err)
				}
				if prefix == m.config.Prefix {
					versions = append(versions, sv)
				}
			} else {
				m.log.Println("Version not found in filename: ", u)
			}
		}

		sort.Sort(versions)

		if len(versions) > 0 {
			m.config.OSDownloaded = versions[len(versions)-1].String()
			err := SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypeOSDownloaded, "0", m.config.OSDownloaded), true)

			if err != nil {
				m.log.Println("Error sending point: ", err)
			}
		} else {
			m.config.OSDownloaded = ""
			err = SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypeOSDownloaded, "0", ""), true)
			if err != nil {
				m.log.Println("Error clearing downloaded point: ", err)
			}
		}
		return nil
	}

	reboot := func() {
		err := exec.Command("reboot").Run()
		if err != nil {
			m.log.Println("Error rebooting: ", err)
		} else {
			m.log.Println("Rebooting ...")
		}
	}

	autoDownload := func() error {
		newestUpdate := ""
		if len(m.config.OSUpdates) > 0 {
			newestUpdate = m.config.OSUpdates[len(m.config.OSUpdates)-1]
		} else {
			return nil
		}

		currentOSV, err := semver.Parse(m.config.VersionOS)
		if err != nil {
			return fmt.Errorf("autodownload, error parsing current OS version: %w", err)
		}

		newestUpdateV, err := semver.Parse(newestUpdate)
		if err != nil {
			return fmt.Errorf("autodownload: Error parsing newest OS update version: %w", err)
		}

		if newestUpdateV.GT(currentOSV) &&
			newestUpdate != m.config.OSDownloaded &&
			newestUpdate != m.config.DownloadOS {
			// download a newer update
			err := SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypeDownloadOS, "", newestUpdate), true)
			if err != nil {
				return fmt.Errorf("error sending point: %w", err)
			}
			m.config.DownloadOS = newestUpdate

			go func(f string) {
				err := download(f)
				if err != nil {
					cSetError <- fmt.Errorf("error downloading update: %w", err)
				}
			}(newestUpdate)
		}
		return nil
	}

	m.setError(nil)
	err := getUpdates()
	if err != nil {
		m.setError(err)
	}
	err = checkDownloads()
	if err != nil {
		m.setError(err)
	}

	osVersion, err := system.ReadOSVersion("VERSION_ID")
	if err != nil {
		m.log.Println("Error reading OS version: ", err)
	} else {
		err := SendNodePoint(m.nc, m.config.ID, data.NewPointString(data.PointTypeVersionOS, "0", osVersion.String()), true)

		if err != nil {
			m.log.Println("Error sending OS version point: ", err)
		}

		m.config.VersionOS = osVersion.String()
	}

	if m.config.DownloadOS != "" {
		go func() {
			err := download(m.config.DownloadOS)
			if err != nil {
				cSetError <- fmt.Errorf("error downloading file: %w", err)
			}
		}()
	}

	checkTickerTime := pointDuration(float64(m.config.PollPeriod), time.Minute,
		30*time.Minute, 10*time.Second)
	checkTicker := time.NewTicker(checkTickerTime)
	if m.config.AutoDownload {
		m.setError(nil)
		err := getUpdates()
		if err != nil {
			m.setError(err)
		} else {
			err := autoDownload()
			if err != nil {
				m.setError(err)
			}
		}
	}

done:
	for {
		select {
		case <-m.stop:
			break done

		case pts := <-m.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDownloadOS:
					if p.Txt() != "" {
						go func(f string) {
							err := download(f)
							if err != nil {
								cSetError <- fmt.Errorf("error downloading update: %w", err)
							}
						}(p.Txt())
					}
				case data.PointTypeDiscardDownload:
					if p.Val() != 0 {
						m.setError(nil)
						err := cleanDownloads()
						if err != nil {
							m.setError(fmt.Errorf("error cleaning downloads: %w", err))
						}
						err = checkDownloads()
						if err != nil {
							m.setError(err)
						}
					}
				case data.PointTypeReboot:
					err := SendNodePoints(m.nc, m.config.ID, data.Points{
						data.NewPointFloat(data.PointTypeReboot, "", 0),
					}, true)
					if err != nil {
						m.log.Println("Error clearing reboot point: ", err)
					}

					reboot()

				case data.PointTypeRefresh:
					err := SendNodePoints(m.nc, m.config.ID, data.Points{
						data.NewPointFloat(data.PointTypeRefresh, "", 0),
					}, true)
					if err != nil {
						m.log.Println("Error clearing reboot reboot point: ", err)
					}

					m.setError(nil)
					err = getUpdates()
					if err != nil {
						m.setError(err)
					}

				case data.PointTypePollPeriod:
					checkTickerTime := pointDuration(p.Val(), time.Minute,
						30*time.Minute, 10*time.Second)
					checkTicker.Reset(checkTickerTime)

				case data.PointTypeAutoDownload:
					if p.Val() == 1 {
						m.setError(nil)
						err := getUpdates()
						if err != nil {
							m.setError(err)
						} else {
							err :=
								autoDownload()
							if err != nil {
								m.setError(err)
							}
						}
					}

				case data.PointTypePrefix:
					m.setError(nil)
					err := cleanDownloads()
					if err != nil {
						m.setError(fmt.Errorf("error cleaning downloads: %w", err))
					}
					err = checkDownloads()
					if err != nil {
						m.setError(err)
					}
					err = getUpdates()
					if err != nil {
						m.setError(err)
					}
				case data.PointTypeURI:
					m.setError(nil)
					err := getUpdates()
					if err != nil {
						m.setError(err)
					}
				}
			}

		case pts := <-m.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

		case <-cDownloadFinished:
			err := checkDownloads()
			if err != nil {
				m.setError(err)
			}

			pts := data.Points{
				data.NewPointString(data.PointTypeDownloadOS, "", ""),
				data.NewPointString(data.PointTypeOSDownloaded, "", m.config.OSDownloaded),
			}
			err = SendNodePoints(m.nc, m.config.ID, pts, true)
			if err != nil {
				m.log.Println("Error sending node points: ", err)
			}
			m.log.Println("Download process finished")

			if m.config.AutoReboot {
				// make sure points have time to stick
				time.Sleep(2 * time.Second)
				reboot()
			}

		case <-checkTicker.C:
			m.setError(nil)
			err := getUpdates()
			if err != nil {
				m.setError(err)
				break
			}
			if m.config.AutoDownload {
				err := autoDownload()
				if err != nil {
					m.setError(err)
				}
			}
			err = checkDownloads()
			if err != nil {
				m.setError(err)
			}

		case err := <-cSetError:
			m.setError(err)
		}
	}

	close(cDownloadFinished)
	close(cSetError)

	return nil
}

// Stop sends a signal to the Run function to exit
func (m *UpdateClient) Stop(_ error) {
	close(m.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (m *UpdateClient) Points(nodeID string, points []data.Point) {
	m.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (m *UpdateClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	m.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

// below is code that used to be in the store and is in process of being
// ported to a client

// StartUpdate starts an update
/*
func StartUpdate(id, url string) error {
	if _, ok := st.updates[id]; ok {
		return fmt.Errorf("Update already in process for dev: %v", id)
	}

	st.updates[id] = time.Now()

	err := st.setSwUpdateState(id, data.SwUpdateState{
		Running: true,
	})

	if err != nil {
		delete(st.updates, id)
		return err
	}

	go func() {
		err := NatsSendFileFromHTTP(st.nc, id, url, func(bytesTx int) {
			err := st.setSwUpdateState(id, data.SwUpdateState{
				Running:     true,
				PercentDone: bytesTx,
			})

			if err != nil {
				log.Println("Error setting update status in DB:", err)
			}
		})

		state := data.SwUpdateState{
			Running: false,
		}

		if err != nil {
			state.Error = "Error updating software"
			state.PercentDone = 0
		} else {
			state.PercentDone = 100
		}

		st.lock.Lock()
		delete(st.updates, id)
		st.lock.Unlock()

		err = st.setSwUpdateState(id, state)
		if err != nil {
			log.Println("Error setting sw update state:", err)
		}
	}()

	return nil
}
*/
