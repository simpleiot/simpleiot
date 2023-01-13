package client

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/simpleiot/simpleiot/data"
)

// Metrics represents the config of a metrics node type
type Metrics struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Type        string `point:"type"`
	Period      int    `point:"period"`
}

// MetricsClient is a SIOT client used to collect system or app metrics
type MetricsClient struct {
	nc            *nats.Conn
	config        Metrics
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	lastSend      time.Time
}

// NewMetricsClient ...
func NewMetricsClient(nc *nats.Conn, config Metrics) Client {
	return &MetricsClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Run the main logic for this client and blocks until stopped
func (m *MetricsClient) Run() error {

	checkPeriod := func() {
		if m.config.Period < 1 {
			m.config.Period = 120
			points := data.Points{
				{Type: data.PointTypePeriod, Value: float64(m.config.Period)},
			}

			err := SendPoints(m.nc, SubjectNodePoints(m.config.ID), points, false)
			if err != nil {
				log.Println("Error sending metrics period: ", err)
			}
		}
	}

	checkPeriod()

	sampleTicker := time.NewTicker(time.Duration(m.config.Period) * time.Second)

done:
	for {
		select {
		case <-m.stop:
			break done

		case <-sampleTicker.C:
			now := time.Now()
			var pts data.Points

			avg, err := load.Avg()
			if err != nil {
				log.Println("Metrics error: ", err)
			} else {
				pts = append(pts, data.Points{
					{Type: data.PointTypeMetricSysLoad,
						Time:  now,
						Key:   "1",
						Value: avg.Load1,
					},
					{Type: data.PointTypeMetricSysLoad,
						Time:  now,
						Key:   "5",
						Value: avg.Load5,
					},
					{Type: data.PointTypeMetricSysLoad,
						Time:  now,
						Key:   "15",
						Value: avg.Load15,
					},
				}...)

			}

			perc, err := cpu.Percent(time.Duration(m.config.Period)*time.Second, false)
			if err != nil {
				log.Println("Metrics error: ", err)
			} else {
				pts = append(pts, data.Point{Type: data.PointTypeMetricSysCPUPercent,
					Time:  now,
					Value: perc[0],
				})
			}

			vm, err := mem.VirtualMemory()
			if err != nil {
				log.Println("Metrics error: ", err)
			} else {
				pts = append(pts, data.Points{{Type: data.PointTypeMetricSysMem,
					Time:  now,
					Key:   data.PointKeyUsedPercent,
					Value: vm.UsedPercent,
				},
					{Type: data.PointTypeMetricSysMem,
						Time:  now,
						Key:   data.PointKeyAvailable,
						Value: float64(vm.Available),
					},
					{Type: data.PointTypeMetricSysMem,
						Time:  now,
						Key:   data.PointKeyUsed,
						Value: float64(vm.Used),
					},
					{Type: data.PointTypeMetricSysMem,
						Time:  now,
						Key:   data.PointKeyFree,
						Value: float64(vm.Free),
					},
				}...)
			}

			parts, err := disk.Partitions(false)
			if err != nil {
				log.Println("Metrics error: ", err)
			} else {
				for _, p := range parts {
					u, err := disk.Usage(p.Mountpoint)
					if err != nil {
						log.Println("Error getting disk usage: ", err)
						continue
					}
					pts = append(pts, data.Points{
						{Time: now,
							Type:  data.PointTypeMetricSysDiskUsedPercent,
							Key:   u.Path,
							Value: u.UsedPercent,
						},
					}...)
				}
			}

			err = SendNodePoints(m.nc, m.config.ID, pts, false)
			if err != nil {
				log.Println("Metrics: error sending points: ", err)
			}

		case pts := <-m.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypePeriod:
					checkPeriod()
					sampleTicker.Reset(time.Duration(m.config.Period) *
						time.Second)

				}
			}

		case pts := <-m.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points: ", err)
			}

		}
	}

	return nil
}

// Stop sends a signal to the Run function to exit
func (m *MetricsClient) Stop(err error) {
	close(m.stop)
}

// Points is called by the Manager when new points for this
// node are received.
func (m *MetricsClient) Points(nodeID string, points []data.Point) {
	m.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (m *MetricsClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	m.newEdgePoints <- NewPoints{nodeID, parentID, points}
}
