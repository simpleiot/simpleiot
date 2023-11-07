package client

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// SignalGenerator config
type SignalGenerator struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disable     bool   `point:"disable"`
	SyncParent  bool   `point:"syncParent"`
	Units       string `point:"units"`
	// SignalType must be one of: "sine", "square", "triangle", or "random walk"
	SignalType string  `point:"signalType"`
	MinValue   float64 `point:"minValue"`
	MaxValue   float64 `point:"maxValue"`
	// InitialValue is the starting value for the signal generator.
	// For random walk, this must be between MinValue and MaxValue. For wave
	// functions, this must be in radians (i.e. between 0 and 2 * Pi).
	InitialValue float64 `point:"initialValue"`
	RoundTo      float64 `point:"roundTo"`
	// SampleRate in Hz.
	SampleRate float64 `point:"sampleRate"`
	// HighRate flag indicates that the points should be emitted on the NATS
	// subject for high-rate points
	HighRate bool `point:"highRate"`
	// BatchPeriod is the batch timer interval in ms. When the timer fires, it
	// generates a batch of points at the specified SampleRate. If not set,
	// timer will fire for each sample at SampleRate.
	BatchPeriod int `point:"batchPeriod"`
	// Frequency for wave functions (in Hz.)
	Frequency float64 `point:"frequency"`
	// Min./Max. increment amount for random walk function
	MinIncrement float64 `point:"minIncrement"`
	MaxIncrement float64 `point:"maxIncrement"`
	// Current value
	Value float64 `point:"value"`
}

/* TODO: Optimization

Note that future designs may keep track of all running SignalGeneratorClients
and manage only a single batch period timer that runs at a scheduled time.
One way to do this is to keep track of each clients' next scheduled run time in
a sorted list. Then, we schedule the timer to run at the soonest scheduled time.
At that time, we process all clients' batches of points and reschedule another
timer.

*/

// BatchSizeLimit is the largest number of points generated per batch.
// If the number of points to be generated by a SignalGenerator exceed this
// limit, the remaining points will be dropped and generated wave signals may
// experience a phase shift.
const BatchSizeLimit = 1000000

// SignalGeneratorClient for signal generator nodes
type SignalGeneratorClient struct {
	log           *log.Logger
	nc            *nats.Conn
	config        SignalGenerator
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewSignalGeneratorClient ...
func NewSignalGeneratorClient(nc *nats.Conn, config SignalGenerator) Client {
	return &SignalGeneratorClient{
		log:           log.New(os.Stderr, "signalGenerator: ", log.LstdFlags|log.Lmsgprefix),
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}, 1),
		newPoints:     make(chan NewPoints, 1),
		newEdgePoints: make(chan NewPoints, 1),
	}
}

// clamp clamps val to fall in the range [min, max].
func clamp(val, min, max float64) float64 {
	// Ensure range of val is [min, max]
	if val < min {
		return min
	} else if val > max {
		return max
	}
	return val
}

// round rounds val to the nearest to.
// When `to` is 0.1, `val` is rounded to the nearest tenth, for example.
// No rounding occurs if to <= 0
func round(val, to float64) float64 {
	if to > 0 {
		return math.Round(val/to) * to
	}
	return val
}

// Run the main logic for this client and blocks until stopped
func (sgc *SignalGeneratorClient) Run() error {
	sgc.log.Println("Starting client:", sgc.config.Description)

	chStopGen := make(chan struct{})

	generator := func(config SignalGenerator) {
		configValid := true
		amplitude := config.MaxValue - config.MinValue
		lastValue := config.InitialValue

		if config.Disable {
			sgc.log.Printf("%v: disabled\n", config.Description)
			configValid = false
		}

		// Validate type
		switch config.SignalType {
		case "sine":
			fallthrough
		case "square":
			fallthrough
		case "triangle":
			if config.Frequency <= 0 {
				sgc.log.Printf("%v: Frequency must be set\n", config.Description)
				configValid = false
			}
			// Note: lastValue is in radians; let's just sanitize it a bit
			lastValue = math.Mod(lastValue, (2 * math.Pi))
		case "random walk":
			if config.MaxIncrement <= config.MinIncrement {
				sgc.log.Printf("%v: MaxIncrement must be larger than MinIncrement\n", config.Description)
				configValid = false
			}
			lastValue = clamp(config.InitialValue, config.MinValue, config.MaxValue)
		default:
			sgc.log.Printf("%v: Type %v is invalid\n", config.Description, config.SignalType)
			configValid = false
		}

		if amplitude <= 0 {
			sgc.log.Printf("%v: MaxValue %v must be larger than MinValue %v\n", config.Description, config.MaxValue, config.MinValue)
			configValid = false
		}

		if config.SampleRate <= 0 {
			sgc.log.Printf("%v: SampleRate must be set\n", config.Description)
			configValid = false
		}

		if config.HighRate && config.BatchPeriod <= 0 {
			sgc.log.Printf("%v: BatchPeriod must be set for high-rate data\n", config.Description)
			configValid = false
		}

		// Determine NATS subject for points based on config settings
		var natsSubject string
		if config.HighRate {
			natsSubject = fmt.Sprintf("phrup.%v.%v", config.Parent, config.ID)
		} else if config.SyncParent {
			natsSubject = SubjectNodePoints(config.Parent)
		} else {
			natsSubject = SubjectNodePoints(config.ID)
		}

		lastBatchTime := time.Now()
		t := time.NewTicker(time.Hour)
		t.Stop()

		// generateBatch generates a batch of points for the time interval
		// [start, stop) based on the signal generator parameters.
		var generateBatch func(start, stop time.Time) (data.Points, time.Time)

		if configValid {
			if config.SignalType == "random walk" {
				sampleInterval := time.Duration(
					float64(time.Second) / config.SampleRate,
				)
				generateBatch = func(start, stop time.Time) (data.Points, time.Time) {
					numPoints := int(
						stop.Sub(start).Seconds() * config.SampleRate,
					)
					endTime := start.Add(time.Duration(numPoints) * sampleInterval)
					if numPoints > BatchSizeLimit {
						numPoints = BatchSizeLimit
					}
					pts := make(data.Points, numPoints)
					for i := 0; i < numPoints; i++ {
						val := lastValue + config.MinIncrement + rand.Float64()*
							(config.MaxIncrement-config.MinIncrement)
						pts[i] = data.Point{
							Type: data.PointTypeValue,
							Time: start.Add(time.Duration(i) * sampleInterval),
							Value: clamp(
								round(val, config.RoundTo),
								config.MinValue,
								config.MaxValue,
							),
							Origin: config.ID,
						}
						lastValue = clamp(val, config.MinValue, config.MaxValue)
					}
					return pts, endTime
				}
			} else {
				// waveFunc converts radians into a scaled wave output
				var waveFunc func(float64) float64
				switch config.SignalType {
				case "sine":
					waveFunc = func(x float64) float64 {
						return (math.Sin(x)+1)/2*amplitude + config.MinValue
					}
				case "square":
					waveFunc = func(x float64) float64 {
						if x >= math.Pi {
							return config.MaxValue
						}
						return config.MinValue
					}
				case "triangle":
					// https://stackoverflow.com/a/22400799/360539
					waveFunc = func(x float64) float64 {
						const p = math.Pi // p is the half-period
						return (amplitude/p)*
							(p-math.Abs(math.Mod(x, (2*p))-p)) +
							config.MinValue
					}
				}

				// dx is the change in x per point
				// Taking SampleRate samples should give Frequency cycles
				dx := 2 * math.Pi * config.Frequency / config.SampleRate
				sampleInterval := time.Duration(
					float64(time.Second) / config.SampleRate,
				)
				generateBatch = func(start, stop time.Time) (data.Points, time.Time) {
					numPoints := int(
						stop.Sub(start).Seconds() * config.SampleRate,
					)
					endTime := start.Add(time.Duration(numPoints) * sampleInterval)
					if numPoints > BatchSizeLimit {
						numPoints = BatchSizeLimit
					}
					pts := make(data.Points, numPoints)
					for i := 0; i < numPoints; i++ {
						// Note: lastValue is in terms of x (i.e. time)
						lastValue += dx
						if lastValue >= 2*math.Pi {
							// Prevent lastValue from growing large
							lastValue -= 2 * math.Pi
						}
						y := waveFunc(lastValue)
						y = clamp(
							round(y, config.RoundTo),
							config.MinValue,
							config.MaxValue,
						)
						pts[i] = data.Point{
							Type:   data.PointTypeValue,
							Time:   start.Add(time.Duration(i) * sampleInterval),
							Value:  y,
							Origin: config.ID,
						}
					}
					return pts, endTime
				}
			}

			// Start batch timer
			batchD := time.Duration(config.BatchPeriod) * time.Millisecond
			sampleD := time.Duration(float64(time.Second) / config.SampleRate)
			if batchD > 0 && batchD > sampleD {
				t.Reset(batchD)
			} else {
				t.Reset(sampleD)
			}
		}

		for {
			select {
			case stopTime := <-t.C:
				pts, endTime := generateBatch(lastBatchTime, stopTime)
				// Send points
				if pts.Len() > 0 {
					lastBatchTime = endTime
					err := SendPoints(sgc.nc, natsSubject, pts, false)
					if err != nil {
						sgc.log.Println("Error sending points:", err)
					}
				}
			case <-chStopGen:
				return
			}
		}
	}

	go generator(sgc.config)

done:
	for {
		select {
		case <-sgc.stop:
			chStopGen <- struct{}{}
			sgc.log.Println("Stopped client: ", sgc.config.Description)
			break done
		case pts := <-sgc.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &sgc.config)
			if err != nil {
				sgc.log.Println("Error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypeDisable,
					data.PointTypeSignalType,
					data.PointTypeMinValue,
					data.PointTypeMaxValue,
					data.PointTypeInitialValue,
					data.PointTypeRoundTo,
					data.PointTypeSampleRate,
					data.PointTypeHighRate,
					data.PointTypeBatchPeriod,
					data.PointTypeFrequency,
					data.PointTypeMinIncrement,
					data.PointTypeMaxIncrement:
					// restart generator
					chStopGen <- struct{}{}
					go generator(sgc.config)
				}
			}

		case pts := <-sgc.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &sgc.config)
			if err != nil {
				sgc.log.Println("Error merging new points: ", err)
			}
		}
	}

	// clean up
	return nil
}

// Stop sends a signal to the Run function to exit
func (sgc *SignalGeneratorClient) Stop(_ error) {
	close(sgc.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (sgc *SignalGeneratorClient) Points(nodeID string, points []data.Point) {
	sgc.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (sgc *SignalGeneratorClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	sgc.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
