package client

import (
	"errors"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

const (
	// iioDefaultPollPeriod is used when a device has no poll period
	// configured
	iioDefaultPollPeriod = time.Second * 3

	// iioValueRefresh is how often a value point is published even when the
	// reading has not moved, so a graph and an upstream instance always have
	// a recent sample
	iioValueRefresh = time.Minute * 10
)

// IIO describes one Linux Industrial I/O device: an ADC, a DAC, or a sensor
// the kernel presents through the same interface. A device node is added by
// the person configuring the system, who sets the device name. The channels on
// that device are then detected and added as children.
type IIO struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Debug       int    `point:"debug"`

	// Device selects the IIO device: the driver's name for it ("ads1015"),
	// the sysfs directory name ("iio:device0"), or a full path. Matching by
	// name is preferred, because the device number depends on probe order and
	// is not stable across boots.
	Device string `point:"device"`

	PollPeriod int `point:"pollPeriod"`

	// Device level settings, written to sysfs when set and left alone when
	// empty. Both still matter for a polled client: many ADCs convert
	// continuously and a sysfs read returns the most recent conversion, so
	// the sample frequency decides how stale a reading can be, and the
	// oversampling ratio trades conversion time for noise.
	SampleFrequency float64 `point:"sampleFrequency"`
	Oversampling    int     `point:"oversampling"`

	// Status
	Connected       bool   `point:"connected"`
	DeviceName      string `point:"deviceName"`
	DevicePath      string `point:"devicePath"`
	Error           string `point:"error"`
	ErrorCount      int    `point:"errorCount"`
	ErrorCountReset bool   `point:"errorCountReset"`

	Channels []IIOChannel `child:"iioChannel"`
}

// IIOChannel describes one channel on an IIO device.
type IIOChannel struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`

	// Channel is the sysfs attribute prefix, without the trailing _raw:
	// "in_voltage0", "in_voltage0-voltage1" for a differential pair,
	// "in_accel_x", "in_temp", "out_voltage0". Filled in by detection.
	Channel string `point:"channel"`
	// ChannelType is the measured quantity parsed out of the channel name:
	// "voltage", "current", "temp", "accel", and so on. It selects the
	// default units and the milli conversion.
	ChannelType string `point:"channelType"`
	// Direction is "input" for an in_* channel and "output" for an out_*
	// channel, reusing the GPIO client's values. Only an output accepts
	// valueSet.
	Direction string `point:"direction"`

	// Conversion applied after the kernel's own scale and offset
	Scale     float64 `point:"scale"`
	Offset    float64 `point:"offset"`
	Units     string  `point:"units"`
	MinChange float64 `point:"minChange"`

	Value    float64 `point:"value"`
	ValueSet float64 `point:"valueSet"`

	Error           string `point:"error"`
	ErrorCount      int    `point:"errorCount"`
	ErrorCountReset bool   `point:"errorCountReset"`
}

// IIOClient is a SIOT client that reads the channels of one Linux IIO device.
// The channels are children of this client rather than clients of their own,
// because a channel is a set of files in a directory shared with every other
// channel on the device, alongside device level settings that apply to all of
// them at once. The device owns the poll timer, the channel detection, and the
// device level error count.
type IIOClient struct {
	log           *log.Logger
	nc            *nats.Conn
	config        IIO
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints

	// devicePath is captured when the client is created so a test that points
	// IIODevicePath at a fixture does not race with a running client.
	devicePath string

	// dev is the resolved device, valid only while resolved is true
	dev      iioDevice
	resolved bool

	// created records channels this client has asked to have a node created
	// for, so a channel detected again before the node appears is not added
	// twice.
	created map[string]bool
	// lastSent records when a value point was last published for each channel
	// node
	lastSent map[string]time.Time

	// idleLogged records that a device with no name set has been reported, so
	// it is logged once rather than on every poll
	idleLogged bool
}

// NewIIOClient returns a new IIO client for the given device node
func NewIIOClient(nc *nats.Conn, config IIO) Client {
	return &IIOClient{
		log:           log.New(os.Stderr, "iio: ", log.LstdFlags|log.Lmsgprefix),
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		devicePath:    IIODevicePath,
		created:       make(map[string]bool),
		lastSent:      make(map[string]time.Time),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (c *IIOClient) Run() error {
	c.log.Printf("Starting client: %v", c.config.Description)

	pollTimer := time.NewTicker(c.pollPeriod())

done:
	for {
		select {
		case <-c.stop:
			break done

		case pts := <-c.newPoints:
			if err := data.MergePoints(pts.ID, pts.Points, &c.config); err != nil {
				c.log.Printf("Error merging new points: %v", err)
			}

			if pts.ID == c.config.ID {
				c.devicePoints(pts.Points, pollTimer)
			} else {
				c.channelPoints(pts.ID, pts.Points)
			}

		case pts := <-c.newEdgePoints:
			if err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &c.config); err != nil {
				c.log.Printf("Error merging new edge points: %v", err)
			}

		case <-pollTimer.C:
			if c.config.Disabled {
				break
			}

			if !c.resolve() {
				break
			}

			c.detect()
			c.read()
		}
	}

	pollTimer.Stop()

	c.log.Printf("Stopped client: %v", c.config.Description)

	return nil
}

// Stop sends a signal to the Run function to exit
func (c *IIOClient) Stop(_ error) {
	close(c.stop)
}

// Points is called by the Manager when new points for this node are received
func (c *IIOClient) Points(nodeID string, points []data.Point) {
	c.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node are
// received
func (c *IIOClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	c.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

func (c *IIOClient) pollPeriod() time.Duration {
	if c.config.PollPeriod <= 0 {
		return iioDefaultPollPeriod
	}

	return time.Millisecond * time.Duration(c.config.PollPeriod)
}

// resolve finds the device the node names, and reports whether the client can
// read it. A device that is not there yet is retried on the next tick, which
// covers a driver that has not probed.
func (c *IIOClient) resolve() bool {
	if c.resolved {
		return true
	}

	if c.config.Device == "" {
		if !c.idleLogged {
			c.log.Printf("%v: waiting for the device to be set",
				c.config.Description)
			c.idleLogged = true
		}

		return false
	}

	c.idleLogged = false

	dev, err := iioFind(c.devicePath, c.config.Device)
	if err != nil {
		c.log.Printf("%v: error finding device: %v", c.config.Description, err)

		c.config.Connected = false
		c.sendDevicePoints(data.Points{
			data.NewPointFloat(data.PointTypeConnected, "", 0),
		})
		c.publishDeviceError(err.Error())

		return false
	}

	c.dev = dev
	c.resolved = true

	c.config.Connected = true
	c.config.DeviceName = dev.Name
	c.config.DevicePath = dev.Path
	c.config.Error = ""
	c.sendDevicePoints(data.Points{
		data.NewPointFloat(data.PointTypeConnected, "", 1),
		data.NewPointString(data.PointTypeDeviceName, "", dev.Name),
		data.NewPointString(data.PointTypeDevicePath, "", dev.Path),
		data.NewPointString(data.PointTypeError, "", ""),
	})

	c.writeDeviceSettings()

	return true
}

// writeDeviceSettings applies the device level settings that are set. A
// setting the driver does not publish is reported once and not counted as an
// error, because a device without an oversampling ratio is not a fault.
func (c *IIOClient) writeDeviceSettings() {
	if c.config.SampleFrequency != 0 {
		c.writeDeviceAttr("sampling_frequency",
			strconv.FormatFloat(c.config.SampleFrequency, 'f', -1, 64))
	}

	if c.config.Oversampling != 0 {
		c.writeDeviceAttr("oversampling_ratio",
			strconv.Itoa(c.config.Oversampling))
	}
}

// writeDeviceAttr writes one device level attribute, reporting an unsupported
// one without counting an error
func (c *IIOClient) writeDeviceAttr(attr, v string) {
	if !c.resolved {
		return
	}

	err := iioWriteAttr(c.dev, attr, v)
	if err == nil {
		return
	}

	if errors.Is(err, errIIOAttrMissing) {
		c.log.Printf("%v: %v", c.config.Description, err)
		return
	}

	c.log.Printf("%v: error writing %v: %v", c.config.Description, attr, err)
	c.publishDeviceError(err.Error())
}

// devicePoints reacts to points on the device node that have a side effect
// beyond updating the config, which the caller has already done.
func (c *IIOClient) devicePoints(pts data.Points, pollTimer *time.Ticker) {
	for _, p := range pts {
		switch p.Type {
		case data.PointTypePollPeriod:
			pollTimer.Reset(c.pollPeriod())

		case data.PointTypeDevice:
			// the next tick resolves the device the node now names
			c.resolved = false
			c.idleLogged = false

		case data.PointTypeSampleFrequency:
			c.writeDeviceAttr("sampling_frequency",
				strconv.FormatFloat(c.config.SampleFrequency, 'f', -1, 64))

		case data.PointTypeOversampling:
			c.writeDeviceAttr("oversampling_ratio",
				strconv.Itoa(c.config.Oversampling))

		case data.PointTypeErrorCountReset:
			if c.config.ErrorCountReset {
				c.config.ErrorCount = 0
				c.config.ErrorCountReset = false
				c.config.Error = ""
				c.sendDevicePoints(data.Points{
					data.NewPointFloat(data.PointTypeErrorCount, "", 0),
					data.NewPointFloat(data.PointTypeErrorCountReset, "", 0),
					data.NewPointString(data.PointTypeError, "", ""),
				})
			}
		}
	}
}

// channelPoints reacts to points on one of the channel nodes below the device.
func (c *IIOClient) channelPoints(id string, pts data.Points) {
	index := -1
	for i := range c.config.Channels {
		if c.config.Channels[i].ID == id {
			index = i
			break
		}
	}

	if index < 0 {
		// Adding or removing a channel restarts this client, so a point for
		// an unknown node means the restart has not happened yet.
		return
	}

	ch := &c.config.Channels[index]

	for _, p := range pts {
		switch p.Type {
		case data.PointTypeValueSet:
			c.setValue(ch)

		case data.PointTypeErrorCountReset:
			if ch.ErrorCountReset {
				ch.ErrorCount = 0
				ch.ErrorCountReset = false
				ch.Error = ""
				c.sendChannelPoints(ch.ID, data.Points{
					data.NewPointFloat(data.PointTypeErrorCount, "", 0),
					data.NewPointFloat(data.PointTypeErrorCountReset, "", 0),
					data.NewPointString(data.PointTypeError, "", ""),
				})
			}
		}
	}
}

// setValue drives an output channel to the requested value and publishes what
// the channel reads back
func (c *IIOClient) setValue(ch *IIOChannel) {
	if ch.Direction != data.PointValueOutput {
		c.log.Printf("%v: valueSet on an input channel", ch.Description)
		c.publishChannelError(ch, "valueSet is only supported on an output channel")
		return
	}

	if !c.resolve() {
		c.publishChannelError(ch, "cannot set value, the device is not connected")
		return
	}

	// the node's own scale and offset convert the physical unit into the
	// quantity the channel measures, so setting a value inverts them before
	// the kernel's conversion
	scale := ch.Scale
	if scale == 0 {
		scale = 1
	}

	if err := iioWrite(c.dev, ch.Channel, (ch.ValueSet-ch.Offset)/scale); err != nil {
		c.log.Printf("%v: error writing channel: %v", ch.Description, err)
		c.publishChannelError(ch, err.Error())
		return
	}

	c.readChannel(ch, true)
}

// detect adds a node for every channel on this device that does not have one
// yet.
func (c *IIOClient) detect() {
	chans, err := iioChannels(c.dev)
	if err != nil {
		c.log.Printf("%v: error detecting channels: %v", c.config.Description, err)
		return
	}

	for _, info := range chans {
		if c.created[info.Channel] {
			continue
		}

		found := false
		for _, ch := range c.config.Channels {
			if ch.Channel == info.Channel {
				found = true
				break
			}
		}

		if found {
			continue
		}

		c.log.Printf("Adding channel: %v", info.Channel)

		direction := data.PointValueInput
		if info.Output {
			direction = data.PointValueOutput
		}

		ne, err := data.Encode(IIOChannel{
			ID:          uuid.New().String(),
			Parent:      c.config.ID,
			Description: info.Channel,
			Channel:     info.Channel,
			ChannelType: info.Type,
			Direction:   direction,
			Units:       IIOUnits(info.Type),
			Scale:       1,
		})
		if err != nil {
			c.log.Printf("Error encoding new channel: %v", err)
			continue
		}

		if err := SendNode(c.nc, ne, c.config.ID); err != nil {
			c.log.Printf("Error sending new channel: %v", err)
			continue
		}

		c.created[info.Channel] = true
	}
}

// read reads every enabled channel on the device and publishes what moved.
func (c *IIOClient) read() {
	for i := range c.config.Channels {
		ch := &c.config.Channels[i]

		if ch.Disabled || ch.Channel == "" {
			continue
		}

		c.readChannel(ch, false)
	}
}

// readChannel reads one channel, applies the node's own scale and offset, and
// publishes the value when it has moved far enough or the refresh interval has
// passed. A forced publish reports the value regardless, which is how a write
// reports back what the channel now reads.
func (c *IIOClient) readChannel(ch *IIOChannel, force bool) {
	v, err := iioRead(c.dev, ch.Channel)
	if err != nil {
		if c.config.Debug > 0 {
			c.log.Printf("Error reading channel %v: %v", ch.Channel, err)
		}
		c.publishChannelError(ch, err.Error())
		return
	}

	scale := ch.Scale
	if scale == 0 {
		scale = 1
	}

	v = v*scale + ch.Offset

	if !c.shouldPublish(ch, v, force) {
		return
	}

	ch.Value = v

	c.sendChannelPoints(ch.ID, data.Points{
		data.NewPointFloat(data.PointTypeValue, "", v),
	})

	c.lastSent[ch.ID] = time.Now()
}

// shouldPublish reports whether a new reading is worth sending. An ADC's low
// bits dither on every sample, so a value that has not moved by at least
// minChange is held back, with the refresh underneath it so a graph and an
// upstream instance always have a recent sample. A forced publish reports the
// value regardless.
func (c *IIOClient) shouldPublish(ch *IIOChannel, v float64, force bool) bool {
	if force {
		return true
	}

	if time.Since(c.lastSent[ch.ID]) > iioValueRefresh {
		return true
	}

	return math.Abs(v-ch.Value) > ch.MinChange
}

// publishDeviceError records why the device could not be resolved or read and
// counts it
func (c *IIOClient) publishDeviceError(msg string) {
	c.config.Error = msg
	c.config.ErrorCount++

	c.sendDevicePoints(data.Points{
		data.NewPointString(data.PointTypeError, "", msg),
		data.NewPointFloat(data.PointTypeErrorCount, "",
			float64(c.config.ErrorCount)),
	})
}

// publishChannelError records a failed read or write against both the channel
// that saw it and the device it is on
func (c *IIOClient) publishChannelError(ch *IIOChannel, msg string) {
	ch.Error = msg
	ch.ErrorCount++
	c.config.ErrorCount++

	c.sendChannelPoints(ch.ID, data.Points{
		data.NewPointString(data.PointTypeError, "", msg),
		data.NewPointFloat(data.PointTypeErrorCount, "",
			float64(ch.ErrorCount)),
	})

	c.sendDevicePoints(data.Points{
		data.NewPointFloat(data.PointTypeErrorCount, "",
			float64(c.config.ErrorCount)),
	})
}

// sendDevicePoints publishes points on this client's own node
func (c *IIOClient) sendDevicePoints(pts data.Points) {
	if err := SendNodePoints(c.nc, c.config.ID, pts, false); err != nil {
		c.log.Printf("Error sending points: %v", err)
	}
}

// sendChannelPoints publishes points on one of the channel nodes below this
// client, which are marked with the device node's origin so the client manager
// passes them on
func (c *IIOClient) sendChannelPoints(nodeID string, pts data.Points) {
	for i := range pts {
		pts[i].Origin = c.config.ID
	}

	if err := SendNodePoints(c.nc, nodeID, pts, false); err != nil {
		c.log.Printf("Error sending channel points: %v", err)
	}
}
