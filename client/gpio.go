package client

import (
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// GPIO describes a single line on a Linux GPIO character device. A node is
// added for each line the application uses; lines are not detected, because a
// chip exposes far more lines than any one application cares about.
type GPIO struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Debug       int    `point:"debug"`

	// Line selection. Chip is a chip name ("gpiochip0"), a chip label, a full
	// device path, or "sim" for a line with no hardware behind it. Line is a
	// line offset ("17") or the kernel's name for the line ("FLOAT_SW").
	Chip string `point:"chip"`
	Line string `point:"line"`

	// Line configuration
	Direction    string `point:"direction"`
	Bias         string `point:"bias"`
	Drive        string `point:"drive"`
	ActiveLow    bool   `point:"activeLow"`
	Debounce     int    `point:"debounce"`
	InitialValue bool   `point:"initialValue"`
	PollPeriod   int    `point:"pollPeriod"`

	// State
	Value    bool `point:"value"`
	ValueSet bool `point:"valueSet"`

	// Status
	Connected       bool   `point:"connected"`
	LineOffset      int    `point:"lineOffset"`
	LineName        string `point:"lineName"`
	Error           string `point:"error"`
	ErrorCount      int    `point:"errorCount"`
	ErrorCountReset bool   `point:"errorCountReset"`
}

const (
	// gpioValueRefresh is how often the line value is published even when it
	// has not changed, so a graph or an upstream instance always has a recent
	// sample
	gpioValueRefresh = time.Minute * 10

	// gpioRetryMax bounds the backoff between attempts to request a line. A
	// line held by a driver that has not loaded yet recovers on its own, so
	// the client keeps trying.
	gpioRetryMax = time.Minute

	// gpioEdgeBuffer is the depth of the channel edge events are delivered on.
	// The kernel event handler must not block, so events are dropped when the
	// buffer is full; it is sized to cover the time the Run loop spends
	// handling one event.
	gpioEdgeBuffer = 16
)

// gpioLine is the part of a requested GPIO line this client uses. Both the
// character device and the simulator implement it, which is what lets the
// tests run on any platform.
type gpioLine interface {
	Value() (bool, error)
	SetValue(bool) error
	Close() error
}

// gpioLineInfo is what the chip reports about a resolved line
type gpioLineInfo struct {
	Offset int
	Name   string
}

// gpioLineConfig is a line request built from the node config
type gpioLineConfig struct {
	// Chip names the GPIO chip: a chip name, a chip label, a device path, or
	// "sim"
	Chip string
	// Line selects a line on that chip, either as an offset or as the
	// kernel's name for the line
	Line      string
	Output    bool
	Bias      string
	Drive     string
	ActiveLow bool
	Debounce  time.Duration
	Initial   bool
	// Edges is nil when the line is polled rather than edge driven
	Edges chan<- bool
}

// gpioRequest resolves the chip and line named in the config, requests the
// line, and returns it along with its resolved offset and kernel name.
func gpioRequest(cfg gpioLineConfig) (gpioLine, gpioLineInfo, error) {
	if cfg.Chip == data.PointValueSim {
		return gpioSimRequest(cfg)
	}

	return gpioCdevRequest(cfg)
}

// gpioRequestPoints are the point types that require the line to be released
// and requested again. The line could be reconfigured in place, but
// re-requesting is one path instead of two and covers a changed chip or line
// as well.
var gpioRequestPoints = map[string]bool{
	data.PointTypeChip:         true,
	data.PointTypeLine:         true,
	data.PointTypeDirection:    true,
	data.PointTypeBias:         true,
	data.PointTypeDrive:        true,
	data.PointTypeActiveLow:    true,
	data.PointTypeDebounce:     true,
	data.PointTypeInitialValue: true,
	data.PointTypePollPeriod:   true,
	data.PointTypeDisabled:     true,
}

// GPIOClient is a SIOT client that reads or drives one GPIO line. Each line is
// an independent request on the chip with its own file descriptor and its own
// edge stream, so a line is a client of its own rather than a child of a chip.
type GPIOClient struct {
	log           *log.Logger
	nc            *nats.Conn
	config        GPIO
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints

	// line is the requested line, nil while the node is disabled, incompletely
	// configured, or the request is failing
	line gpioLine
	// edges carries edge events from the kernel, and is nil for an output or a
	// polled input. A nil channel blocks forever in the Run loop select, which
	// is what a line with no edge events should do.
	edges chan bool

	// lastSent is when the value point was last published, which paces the
	// refresh
	lastSent time.Time

	// retryAttempts counts consecutive failed requests and drives the backoff
	retryAttempts int

	// idleLogged records that an incomplete configuration has been reported,
	// so it is logged once rather than on every point
	idleLogged bool
}

// NewGPIOClient returns a new GPIOClient using its configuration read from the
// Client Manager
func NewGPIOClient(nc *nats.Conn, config GPIO) Client {
	return &GPIOClient{
		log:           log.New(os.Stderr, "gpio: ", log.LstdFlags|log.Lmsgprefix),
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// output reports whether the line is configured as an output. An unset
// direction is an input, which is the safe default for a line whose wiring is
// not yet described.
func (c *GPIOClient) output() bool {
	return c.config.Direction == data.PointValueOutput
}

// pollPeriod returns the configured poll period, or zero when the line should
// be driven by edge events instead
func (c *GPIOClient) pollPeriod() time.Duration {
	if c.config.PollPeriod <= 0 {
		return 0
	}

	return time.Millisecond * time.Duration(c.config.PollPeriod)
}

// idle reports whether the node is deliberately not holding a line, either
// because it is disabled or because the chip and line have not been set yet.
// An idle node is not retried, because there is nothing to retry until a point
// changes.
func (c *GPIOClient) idle() bool {
	return c.config.Disabled || c.config.Chip == "" || c.config.Line == ""
}

// Run runs the main logic for this client and blocks until stopped
func (c *GPIOClient) Run() error {
	c.log.Printf("Starting client: %v", c.config.Description)

	// a ticker cannot be created stopped, so it is created and stopped, then
	// reset whenever the line is configured for polling
	pollTimer := time.NewTicker(time.Hour)
	pollTimer.Stop()

	refreshTimer := time.NewTicker(gpioValueRefresh)

	retryTimer := time.NewTimer(time.Hour)
	if !retryTimer.Stop() {
		<-retryTimer.C
	}

	stopRetry := func() {
		if !retryTimer.Stop() {
			select {
			case <-retryTimer.C:
			default:
			}
		}
	}

	// request releases any line the client holds and requests it again from
	// the current configuration, then matches the timers to the result
	request := func() {
		stopRetry()

		c.requestLine()

		if c.line != nil && c.pollPeriod() > 0 {
			pollTimer.Reset(c.pollPeriod())
		} else {
			pollTimer.Stop()
		}

		if c.line == nil && !c.idle() {
			retryTimer.Reset(ExpBackoff(c.retryAttempts, gpioRetryMax))
			c.retryAttempts++
		}
	}

	request()

done:
	for {
		select {
		case <-c.stop:
			break done

		case pts := <-c.newPoints:
			if err := data.MergePoints(pts.ID, pts.Points, &c.config); err != nil {
				c.log.Printf("Error merging new points: %v", err)
			}

			rerequest := false
			setValue := false

			for _, p := range pts.Points {
				if gpioRequestPoints[p.Type] {
					rerequest = true
				}

				switch p.Type {
				case data.PointTypeValueSet:
					setValue = true

				case data.PointTypeErrorCountReset:
					if p.Bool() {
						c.resetErrorCount()
					}
				}
			}

			// request once even when several configuration points arrive
			// together
			if rerequest {
				request()
			}

			if setValue {
				c.setValue()
			}

		case pts := <-c.newEdgePoints:
			if err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &c.config); err != nil {
				c.log.Printf("Error merging new edge points: %v", err)
			}

		case v := <-c.edges:
			if c.config.Debug > 0 {
				c.log.Printf("%v: edge, value %v", c.config.Description, v)
			}
			c.publishValue(v, false)

		case <-pollTimer.C:
			// a tick queued before the line was reconfigured is ignored
			if c.line != nil && c.pollPeriod() > 0 {
				c.readAndPublish(false)
			}

		case <-refreshTimer.C:
			c.readAndPublish(true)

		case <-retryTimer.C:
			request()
		}
	}

	pollTimer.Stop()
	refreshTimer.Stop()
	stopRetry()
	c.closeLine()

	c.log.Printf("Stopped client: %v", c.config.Description)

	return nil
}

// Stop sends a signal to the Run function to exit
func (c *GPIOClient) Stop(_ error) {
	close(c.stop)
}

// Points is called by the Manager when new points for this node are received
func (c *GPIOClient) Points(nodeID string, points []data.Point) {
	c.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node are
// received
func (c *GPIOClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	c.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

// requestLine releases the line the client holds and requests it again. The
// status points are published to match, so the node always describes what the
// client is actually holding.
func (c *GPIOClient) requestLine() {
	c.closeLine()

	if c.idle() {
		if !c.config.Disabled && !c.idleLogged {
			c.log.Printf("%v: waiting for chip and line to be set",
				c.config.Description)
			c.idleLogged = true
		}

		c.config.Connected = false
		c.config.Error = ""
		c.sendPoints(data.Points{
			data.NewPointFloat(data.PointTypeConnected, "", 0),
			data.NewPointString(data.PointTypeError, "", ""),
		})

		return
	}

	c.idleLogged = false

	// an input with no poll period is driven by kernel edge events, which
	// reach the point stream in about a millisecond with no timer running
	var edges chan<- bool
	if !c.output() && c.pollPeriod() <= 0 {
		c.edges = make(chan bool, gpioEdgeBuffer)
		edges = c.edges
	}

	line, info, err := gpioRequest(gpioLineConfig{
		Chip:      c.config.Chip,
		Line:      c.config.Line,
		Output:    c.output(),
		Bias:      c.config.Bias,
		Drive:     c.config.Drive,
		ActiveLow: c.config.ActiveLow,
		Debounce:  time.Millisecond * time.Duration(c.config.Debounce),
		Initial:   c.config.InitialValue,
		Edges:     edges,
	})
	if err != nil {
		c.edges = nil
		c.log.Printf("%v: error requesting line: %v", c.config.Description, err)

		c.config.Connected = false
		c.sendPoints(data.Points{
			data.NewPointFloat(data.PointTypeConnected, "", 0),
		})
		c.publishError(err.Error())

		return
	}

	c.line = line
	c.retryAttempts = 0

	c.config.Connected = true
	c.config.LineOffset = info.Offset
	c.config.LineName = info.Name
	c.config.Error = ""
	c.sendPoints(data.Points{
		data.NewPointFloat(data.PointTypeConnected, "", 1),
		data.NewPointFloat(data.PointTypeLineOffset, "", float64(info.Offset)),
		data.NewPointString(data.PointTypeLineName, "", info.Name),
		data.NewPointString(data.PointTypeError, "", ""),
	})

	// edge events only report changes, so the initial value is always read
	// and published
	c.readAndPublish(true)
}

// closeLine releases the line back to the kernel. The edge channel is dropped
// whether or not a line was held, so the Run loop never selects on a channel
// nothing can write to.
func (c *GPIOClient) closeLine() {
	c.edges = nil

	if c.line == nil {
		return
	}

	if err := c.line.Close(); err != nil {
		c.log.Printf("%v: error closing line: %v", c.config.Description, err)
	}

	c.line = nil
}

// setValue drives an output line to the requested state and publishes what the
// line reads back
func (c *GPIOClient) setValue() {
	if !c.output() {
		c.log.Printf("%v: valueSet on an input line", c.config.Description)
		c.publishError("valueSet is only supported on an output line")
		return
	}

	if c.line == nil {
		c.publishError("cannot set value, line is not connected")
		return
	}

	if err := c.line.SetValue(c.config.ValueSet); err != nil {
		c.log.Printf("%v: error setting value: %v", c.config.Description, err)
		c.publishError(err.Error())
		return
	}

	c.readAndPublish(true)
}

// readAndPublish reads the line and publishes the value. A forced publish
// reports the value even when it has not changed, which is how the initial
// read and the refresh keep a recent sample on the bus.
func (c *GPIOClient) readAndPublish(force bool) {
	if c.line == nil {
		return
	}

	v, err := c.line.Value()
	if err != nil {
		c.log.Printf("%v: error reading line: %v", c.config.Description, err)
		c.publishError(err.Error())
		return
	}

	c.publishValue(v, force)
}

// publishValue sends the value point, skipping an unchanged value unless the
// refresh interval has passed or the caller forces it
func (c *GPIOClient) publishValue(v bool, force bool) {
	if !force && v == c.config.Value &&
		time.Since(c.lastSent) <= gpioValueRefresh {
		return
	}

	c.config.Value = v
	c.lastSent = time.Now()

	c.sendPoints(data.Points{
		data.NewPointFloat(data.PointTypeValue, "", data.BoolToFloat(v)),
	})
}

// publishError records why the last request or access failed and counts it
func (c *GPIOClient) publishError(msg string) {
	c.config.Error = msg
	c.config.ErrorCount++

	c.sendPoints(data.Points{
		data.NewPointString(data.PointTypeError, "", msg),
		data.NewPointFloat(data.PointTypeErrorCount, "",
			float64(c.config.ErrorCount)),
	})
}

// resetErrorCount zeroes the error count and clears both the request that
// triggered it and the error text
func (c *GPIOClient) resetErrorCount() {
	c.config.ErrorCount = 0
	c.config.ErrorCountReset = false
	c.config.Error = ""

	c.sendPoints(data.Points{
		data.NewPointFloat(data.PointTypeErrorCount, "", 0),
		data.NewPointFloat(data.PointTypeErrorCountReset, "", 0),
		data.NewPointString(data.PointTypeError, "", ""),
	})
}

// sendPoints publishes points on this client's own node
func (c *GPIOClient) sendPoints(pts data.Points) {
	if err := SendNodePoints(c.nc, c.config.ID, pts, false); err != nil {
		c.log.Printf("Error sending points: %v", err)
	}
}
