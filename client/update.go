package client

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Update represents the config of a metrics node type
type Update struct {
	ID          string   `node:"id"`
	Parent      string   `node:"parent"`
	Description string   `point:"description"`
	URI         string   `point:"uri"`
	OSUpdates   []string `point:"osUpdate"`
	Prefix      string   `point:"prefix"`
	Refresh     bool     `point:"refresh"`
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

// Run the main logic for this client and blocks until stopped
func (m *UpdateClient) Run() error {

	// fill in default prefix
	if m.config.Prefix == "" {
		p, err := os.Hostname()
		if err != nil {
			m.log.Println("Error getting hostname: ", err)
		} else {
			m.log.Println("Setting update prefix to: ", p)
			err := SendNodePoint(m.nc, m.config.ID, data.Point{
				Time: time.Now(),
				Type: data.PointTypePrefix,
				Key:  "0",
				Text: p}, false)
			if err != nil {
				m.log.Println("Error sending point: ", err)
			} else {
				m.config.Prefix = p
			}
		}
	}

	getUpdates := func() {
		p, err := url.JoinPath(m.config.URI, "files.txt")
		if err != nil {
			m.log.Println("URI error: ", err)
		}
		resp, err := http.Get(p)
		if err != nil {
			m.log.Println("Error getting updates: ", err)
			return
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			m.log.Println("Error reading http response: ", err)
			return
		}

		updates := strings.Split(string(body), "\n")

		updates = slices.DeleteFunc(updates, func(u string) bool {
			return !strings.HasPrefix(u, m.config.Prefix)
		})

		versions := semver.Versions{}

		re := regexp.MustCompile(`_(\d+\.\d+\.\d+)\.upd`)

		for _, u := range updates {
			matches := re.FindStringSubmatch(u)
			if len(matches) > 1 {
				version := matches[1]
				sv, err := semver.Parse(version)
				if err != nil {
					m.log.Printf("Error parsing version %v: %v", version, err)
				}
				versions = append(versions, sv)
			} else {
				m.log.Println("Version not found in filename: ", u)
			}
		}

		sort.Sort(versions)

		underflowCount := len(m.config.OSUpdates) - len(versions)

		// need to update versions available
		pts := data.Points{}
		now := time.Now()
		for i, v := range versions {
			pts = append(pts, data.Point{
				Time: now, Type: data.PointTypeOSUpdate, Text: v.String(), Key: strconv.Itoa(i),
			})
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			m.log.Println("Error sending version points: ", err)

		}

		if underflowCount > 0 {
			pts = data.Points{}
			for i := len(versions); i < len(versions)+underflowCount; i++ {
				pts = append(pts, data.Point{
					Time: now, Type: data.PointTypeOSUpdate, Key: strconv.Itoa(i), Tombstone: 1,
				})
			}
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			m.log.Println("Error sending version points: ", err)

		}
	}

	getUpdates()

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
				}
			}

		case pts := <-m.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

		}
	}

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
