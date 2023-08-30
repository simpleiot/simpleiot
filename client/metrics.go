package client

import (
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/simpleiot/simpleiot/data"
)

// Metrics represents the config of a metrics node type
type Metrics struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Type        string `point:"type"`
	Name        string `point:"name"`
	Period      int    `point:"period"`
}

// MetricsClient is a SIOT client used to collect system or app metrics
type MetricsClient struct {
	nc            *nats.Conn
	config        Metrics
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
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
	if m.config.Type == data.PointValueSystem {
		m.sysStart()
	}

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
			switch m.config.Type {
			case data.PointValueSystem:
				m.sysPeriodic()
			case data.PointValueApp:
				m.appPeriodic("")
			case data.PointValueProcess:
				m.appPeriodic(m.config.Name)
			case data.PointValueAllProcesses:
				m.allProcPeriodic()
			default:
				log.Println("Metrics: Must select metric type")
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
				case data.PointTypeType:
					if m.config.Type == data.PointValueSystem {
						m.sysStart()
					}
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
func (m *MetricsClient) Stop(_ error) {
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

func (m *MetricsClient) sysStart() {
	now := time.Now()
	// collect static host stats on startup
	hostStat, err := host.Info()
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		// TODO, only send points if they have changed
		pts := data.Points{
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyHostname,
				Text: hostStat.Hostname,
			},
			{
				Type:  data.PointTypeHostBootTime,
				Time:  now,
				Value: float64(hostStat.BootTime),
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyOS,
				Text: hostStat.OS,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyPlatform,
				Text: hostStat.Platform,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyPlatformFamily,
				Text: hostStat.PlatformFamily,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyPlatformVersion,
				Text: hostStat.PlatformVersion,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyKernelVersion,
				Text: hostStat.KernelVersion,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyKernelArch,
				Text: hostStat.KernelArch,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyVirtualizationSystem,
				Text: hostStat.VirtualizationSystem,
			},
			{
				Type: data.PointTypeHost,
				Time: now,
				Key:  data.PointKeyVirtualizationRole,
				Text: hostStat.VirtualizationRole,
			},
		}
		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points: ", err)
		}
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		pt := data.Point{
			Type:  data.PointTypeMetricSysMem,
			Time:  now,
			Key:   data.PointKeyTotal,
			Value: float64(vm.Total),
		}

		err = SendNodePoint(m.nc, m.config.ID, pt, false)
		if err != nil {
			log.Println("Metrics: error sending points: ", err)
		}
	}

}

func (m *MetricsClient) sysPeriodic() {
	now := time.Now()
	var pts data.Points

	avg, err := load.Avg()
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		pts = append(pts, data.Points{
			{
				Type:  data.PointTypeMetricSysLoad,
				Time:  now,
				Key:   "1",
				Value: avg.Load1,
			},
			{
				Type:  data.PointTypeMetricSysLoad,
				Time:  now,
				Key:   "5",
				Value: avg.Load5,
			},
			{
				Type:  data.PointTypeMetricSysLoad,
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
		pts = append(pts, data.Points{
			{
				Type:  data.PointTypeMetricSysMemUsedPercent,
				Time:  now,
				Value: vm.UsedPercent,
			},
			{
				Type:  data.PointTypeMetricSysMem,
				Time:  now,
				Key:   data.PointKeyAvailable,
				Value: float64(vm.Available),
			},
			{
				Type:  data.PointTypeMetricSysMem,
				Time:  now,
				Key:   data.PointKeyUsed,
				Value: float64(vm.Used),
			},
			{
				Type:  data.PointTypeMetricSysMem,
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
			if strings.HasPrefix(p.Mountpoint, "/run/media") {
				// don't track stats for removable media
				continue
			}

			u, err := disk.Usage(p.Mountpoint)
			if err != nil {
				log.Println("Error getting disk usage: ", err)
				continue
			}
			pts = append(pts, data.Points{
				{
					Time:  now,
					Type:  data.PointTypeMetricSysDiskUsedPercent,
					Key:   u.Path,
					Value: u.UsedPercent,
				},
			}...)
		}
	}

	netio, err := net.IOCounters(true)
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		for _, io := range netio {
			pts = append(pts, data.Points{
				{
					Time:  now,
					Type:  data.PointTypeMetricSysNetBytesRecv,
					Key:   io.Name,
					Value: float64(io.BytesRecv),
				},
				{
					Time:  now,
					Type:  data.PointTypeMetricSysNetBytesSent,
					Key:   io.Name,
					Value: float64(io.BytesSent),
				},
			}...)
		}

	}

	uptime, err := host.Uptime()
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		pts = append(pts, data.Point{
			Time:  now,
			Type:  data.PointTypeMetricSysUptime,
			Value: float64(uptime),
		})
	}

	temps, err := host.SensorsTemperatures()
	if err != nil {
		log.Println("Error reading sensors: ", err)
	} else {
		for _, t := range temps {
			pts = append(pts, data.Points{
				{
					Time:  now,
					Type:  data.PointTypeTemperature,
					Key:   t.SensorKey,
					Value: t.Temperature,
				},
			}...)
		}
	}

	err = SendNodePoints(m.nc, m.config.ID, pts, false)
	if err != nil {
		log.Println("Metrics: error sending points: ", err)
	}
}

// if procName is "", then collect stats for this app
func (m *MetricsClient) appPeriodic(procName string) {
	now := time.Now()

	if procName == "" {
		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		numGoRoutine := runtime.NumGoroutine()

		pts := data.Points{
			{
				Time:  now,
				Type:  data.PointTypeMetricAppAlloc,
				Value: float64(memStats.Alloc),
			},
			{
				Time:  now,
				Type:  data.PointTypeMetricAppNumGoroutine,
				Value: float64(numGoRoutine),
			},
		}

		err := SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points: ", err)
		}
	}

	pid := os.Getpid()

	procs, err := process.Processes()
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		var accumCPUPerc, accumMemPerc, accumMemRSS float64
		var procCount int
		for _, p := range procs {
			if procName != "" {
				name, err := p.Name()
				if err != nil {
					log.Println("Error getting process name: ", err)
					continue
				}
				if name != procName {
					continue
				}
			} else {
				if p.Pid != int32(pid) {
					continue
				}
			}

			procCount++

			cpuPerc, err := p.CPUPercent()
			if err != nil {
				log.Println("Error getting CPU percent for proc: ", err)
				break
			}

			accumCPUPerc += cpuPerc

			memPerc, err := p.MemoryPercent()
			if err != nil {
				log.Println("Error getting mem percent for proc: ", err)
				break
			}

			accumMemPerc += float64(memPerc)

			memInfo, err := p.MemoryInfo()
			if err != nil {
				log.Println("Error getting mem info: ", err)
				break
			}

			accumMemRSS += float64(memInfo.RSS)
		}

		pts := data.Points{
			{
				Time:  now,
				Type:  data.PointTypeMetricProcCPUPercent,
				Value: float64(accumCPUPerc),
			},
			{
				Time:  now,
				Type:  data.PointTypeMetricProcMemPercent,
				Value: float64(accumMemPerc),
			},
			{
				Time:  now,
				Type:  data.PointTypeMetricProcMemRSS,
				Value: float64(accumMemRSS),
			},
		}

		if procName != "" {
			pts = append(pts, data.Point{
				Time:  now,
				Type:  data.PointTypeCount,
				Value: float64(procCount),
			})
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points: ", err)
		}

	}
}

type procMetrics struct {
	count float64
	cpu   float64
	mem   float64
	rss   float64
}

func (m *MetricsClient) allProcPeriodic() {
	now := time.Now()

	metrics := make(map[string]procMetrics)

	procs, err := process.Processes()
	if err != nil {
		log.Println("Metrics error: ", err)
	} else {
		for _, p := range procs {
			name, err := p.Name()
			if err != nil {
				log.Println("Error getting process name: ", err)
				continue
			}

			m := metrics[name]

			m.count++

			cpuPerc, err := p.CPUPercent()
			if err != nil {
				log.Println("Error getting CPU percent for proc: ", err)
				break
			}

			m.cpu += cpuPerc

			memPerc, err := p.MemoryPercent()
			if err != nil {
				log.Println("Error getting mem percent for proc: ", err)
				break
			}

			m.mem += float64(memPerc)

			memInfo, err := p.MemoryInfo()
			if err != nil {
				log.Println("Error getting mem info: ", err)
				break
			}

			m.rss += float64(memInfo.RSS)

			metrics[name] = m
		}

		pts := make(data.Points, len(metrics)*4)
		var i int
		for k, v := range metrics {
			pts[i].Time = now
			pts[i].Key = k
			pts[i].Type = data.PointTypeMetricProcCPUPercent
			pts[i].Value = v.cpu
			i++

			pts[i].Time = now
			pts[i].Key = k
			pts[i].Type = data.PointTypeMetricProcMemPercent
			pts[i].Value = v.mem
			i++

			pts[i].Time = now
			pts[i].Key = k
			pts[i].Type = data.PointTypeMetricProcMemRSS
			pts[i].Value = v.rss
			i++

			pts[i].Time = now
			pts[i].Key = k
			pts[i].Type = data.PointTypeCount
			pts[i].Value = v.count
			i++
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points: ", err)
		}
	}
}
