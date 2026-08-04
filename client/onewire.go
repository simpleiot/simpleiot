package client

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// OneWireDevicePath is the root of the Linux 1-wire (w1) sysfs tree. It is a
// variable so that tests can point it at a fixture directory; nothing else
// should change it.
var OneWireDevicePath = "/sys/bus/w1/devices"

// oneWireDefaultPollPeriod is used when a bus has no poll period configured
const oneWireDefaultPollPeriod = time.Second * 3

// oneWireValueRefresh is how often a value point is published even when the
// reading has not changed
const oneWireValueRefresh = time.Minute * 10

// OneWire describes a 1-wire bus. A bus node is added by the person
// configuring the system, who sets the index of the bus controller. The
// devices on that bus are then detected and added as children.
type OneWire struct {
	ID              string      `node:"id"`
	Parent          string      `node:"parent"`
	Description     string      `point:"description"`
	Index           int         `point:"index"`
	PollPeriod      int         `point:"pollPeriod"`
	Debug           int         `point:"debug"`
	Disabled        bool        `point:"disabled"`
	ErrorCount      int         `point:"errorCount"`
	ErrorCountReset bool        `point:"errorCountReset"`
	IOs             []OneWireIO `child:"oneWireIO"`
}

// OneWireIO describes a device on a 1-wire bus
type OneWireIO struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	// DeviceID is the 1-wire address of the device, which is unrelated to the
	// SIOT node ID above.
	DeviceID        string  `point:"id"`
	Units           string  `point:"units"`
	Value           float64 `point:"value"`
	Disabled        bool    `point:"disabled"`
	ErrorCount      int     `point:"errorCount"`
	ErrorCountReset bool    `point:"errorCountReset"`
}

// OneWireClient is a SIOT client that reads the devices on a 1-wire bus. The
// devices are children of this client rather than clients of their own,
// because the bus owns the poll timer, the device detection, and the bus level
// error count.
type OneWireClient struct {
	nc            *nats.Conn
	config        OneWire
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints

	// devicePath is captured when the client is created so a test that points
	// OneWireDevicePath at a fixture does not race with a running client.
	devicePath string
	// created records devices this client has asked to have a node created
	// for, so a device detected again before the node appears is not added
	// twice.
	created map[string]bool
	// lastSent records when a value point was last published for each IO node
	lastSent map[string]time.Time
}

// NewOneWireClient returns a new 1-wire client for the given bus node
func NewOneWireClient(nc *nats.Conn, config OneWire) Client {
	return &OneWireClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		devicePath:    OneWireDevicePath,
		created:       make(map[string]bool),
		lastSent:      make(map[string]time.Time),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (c *OneWireClient) Run() error {
	log.Println("Starting 1-wire client:", c.config.Description)

	pollTimer := time.NewTicker(c.pollPeriod())

done:
	for {
		select {
		case <-c.stop:
			break done

		case pts := <-c.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &c.config)
			if err != nil {
				log.Println("1-wire: error merging new points:", err)
			}

			if pts.ID == c.config.ID {
				c.busPoints(pts.Points, pollTimer)
			} else {
				c.ioPoints(pts.ID, pts.Points)
			}

		case pts := <-c.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &c.config)
			if err != nil {
				log.Println("1-wire: error merging new edge points:", err)
			}

		case <-pollTimer.C:
			if c.config.Disabled {
				break
			}

			c.detect()
			c.read()
		}
	}

	log.Println("Stopping 1-wire client:", c.config.Description)
	pollTimer.Stop()

	return nil
}

// Stop sends a signal to the Run function to exit
func (c *OneWireClient) Stop(_ error) {
	close(c.stop)
}

// Points is called by the Manager when new points for this node are received.
func (c *OneWireClient) Points(nodeID string, points []data.Point) {
	c.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node are
// received.
func (c *OneWireClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	c.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

func (c *OneWireClient) pollPeriod() time.Duration {
	if c.config.PollPeriod <= 0 {
		return oneWireDefaultPollPeriod
	}

	return time.Millisecond * time.Duration(c.config.PollPeriod)
}

// busPoints reacts to points on the bus node that have a side effect beyond
// updating the config, which the caller has already done.
func (c *OneWireClient) busPoints(pts data.Points, pollTimer *time.Ticker) {
	for _, p := range pts {
		switch p.Type {
		case data.PointTypePollPeriod:
			pollTimer.Reset(c.pollPeriod())

		case data.PointTypeErrorCountReset:
			if c.config.ErrorCountReset {
				c.config.ErrorCount = 0
				c.config.ErrorCountReset = false
				c.sendResetPoints(c.config.ID)
			}
		}
	}
}

// ioPoints reacts to points on one of the device nodes below the bus.
func (c *OneWireClient) ioPoints(id string, pts data.Points) {
	index := -1
	for i := range c.config.IOs {
		if c.config.IOs[i].ID == id {
			index = i
			break
		}
	}

	if index < 0 {
		// Adding or removing a device restarts this client, so a point for an
		// unknown node means the restart has not happened yet.
		return
	}

	io := &c.config.IOs[index]

	for _, p := range pts {
		if p.Type == data.PointTypeErrorCountReset && io.ErrorCountReset {
			io.ErrorCount = 0
			io.ErrorCountReset = false
			c.sendResetPoints(io.ID)
		}
	}
}

// sendResetPoints zeroes an error count and clears the reset request that
// triggered it.
func (c *OneWireClient) sendResetPoints(nodeID string) {
	pts := data.Points{
		data.NewPointFloat(data.PointTypeErrorCount, "", 0),
		data.NewPointFloat(data.PointTypeErrorCountReset, "", 0),
	}

	if nodeID != c.config.ID {
		for i := range pts {
			pts[i].Origin = c.config.ID
		}
	}

	if err := SendNodePoints(c.nc, nodeID, pts, true); err != nil {
		log.Println("1-wire: error sending reset points:", err)
	}
}

// detect adds a node for every device on this bus that does not have one yet.
func (c *OneWireClient) detect() {
	ids, err := oneWireDetect(c.devicePath, c.config.Index)
	if err != nil {
		log.Println("1-wire: error detecting devices:", err)
		return
	}

	for _, id := range ids {
		if c.created[id] {
			continue
		}

		found := false
		for _, io := range c.config.IOs {
			if io.DeviceID == id {
				found = true
				break
			}
		}

		if found {
			continue
		}

		log.Println("Adding 1-wire device:", id)

		ne, err := data.Encode(OneWireIO{
			ID:          uuid.New().String(),
			Parent:      c.config.ID,
			DeviceID:    id,
			Description: "New IO, please edit",
		})
		if err != nil {
			log.Println("1-wire: error encoding new device:", err)
			continue
		}

		if err := SendNode(c.nc, ne, c.config.ID); err != nil {
			log.Println("1-wire: error sending new device:", err)
			continue
		}

		c.created[id] = true
	}
}

// read reads every enabled device on the bus and publishes what changed.
func (c *OneWireClient) read() {
	for i := range c.config.IOs {
		io := &c.config.IOs[i]

		if io.Disabled || io.DeviceID == "" {
			continue
		}

		v, err := oneWireRead(c.devicePath, io.DeviceID, io.Units)
		if err != nil {
			if c.config.Debug > 0 {
				log.Printf("Error reading 1-wire device %v: %v\n", io.DeviceID, err)
			}
			c.logError(io)
			continue
		}

		if v == io.Value && time.Since(c.lastSent[io.ID]) <= oneWireValueRefresh {
			continue
		}

		io.Value = v

		p := data.NewPointFloat(data.PointTypeValue, "", v)
		p.Origin = c.config.ID

		if err := SendNodePoint(c.nc, io.ID, p, false); err != nil {
			log.Println("1-wire: error sending value:", err)
			continue
		}

		c.lastSent[io.ID] = time.Now()
	}
}

// logError counts a failed read against the bus and the device that saw it.
func (c *OneWireClient) logError(io *OneWireIO) {
	c.config.ErrorCount++
	io.ErrorCount++

	busPoint := data.NewPointFloat(data.PointTypeErrorCount, "", float64(c.config.ErrorCount))
	if err := SendNodePoint(c.nc, c.config.ID, busPoint, false); err != nil {
		log.Println("1-wire: error sending bus error count:", err)
	}

	ioPoint := data.NewPointFloat(data.PointTypeErrorCount, "", float64(io.ErrorCount))
	ioPoint.Origin = c.config.ID
	if err := SendNodePoint(c.nc, io.ID, ioPoint, false); err != nil {
		log.Println("1-wire: error sending device error count:", err)
	}
}

// oneWireDetect returns the IDs of the DS18B20 sensors on one bus. Detection
// is scoped to the bus controller, because the flat device directory lists
// every sensor on every controller, and with two controllers each bus would
// otherwise claim all of them.
func oneWireDetect(root string, index int) ([]string, error) {
	pattern := filepath.Join(root, fmt.Sprintf("w1_bus_master%v", index), "28-*")

	dirs, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var ids []string

	for _, dir := range dirs {
		f, err := os.Stat(dir)
		if err != nil || !f.IsDir() {
			continue
		}
		ids = append(ids, filepath.Base(dir))
	}

	return ids, nil
}

// oneWireRead reads the temperature of one sensor in degrees Celsius, or in
// Fahrenheit when units is "F". The read uses the flat device directory, which
// is the same path regardless of which controller the sensor is on.
func oneWireRead(root, deviceID, units string) (float64, error) {
	path := filepath.Join(root, deviceID, "temperature")

	d, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	s := strings.TrimSpace(string(d))
	if s == "" {
		return 0, fmt.Errorf("no data in %v", path)
	}

	raw, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("error parsing %v: %w", path, err)
	}

	v := float64(raw) / 1000

	if units == "F" {
		v = v*1.8 + 32
	}

	return v, nil
}
