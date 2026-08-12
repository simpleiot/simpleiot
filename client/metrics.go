package client

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

	// Prometheus scrape config, used when Type is prometheus
	URI          string   `point:"uri"`
	Prefixes     []string `point:"prefix"`
	CounterDelta bool     `point:"counterDelta"`
	MaxSeries    int      `point:"maxSeries"`
}

// MetricsClient is a SIOT client used to collect system or app metrics
type MetricsClient struct {
	nc            *nats.Conn
	config        Metrics
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints

	// previous counter values, so a Prometheus scrape can publish the change
	// since the last one. Keyed by metric name and point key.
	promCounters map[string]float64

	// metric names already reported as colliding with a node configuration
	// point type, so the log is written once per name rather than every
	// scrape
	promSkipped map[string]bool

	// whether a configured series limit above the ceiling has been logged, so
	// it is reported once rather than every scrape. The error point on the
	// node carries it for as long as the setting stands.
	promClampLogged bool

	// the error last published on the node, so an unchanged error is not
	// resent every period
	promError string
}

// NewMetricsClient ...
func NewMetricsClient(nc *nats.Conn, config Metrics) Client {
	return &MetricsClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		promCounters:  make(map[string]float64),
		promSkipped:   make(map[string]bool),
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
				data.NewPointFloat(data.PointTypePeriod, "", float64(m.config.Period)),
			}

			err := SendPoints(m.nc, SubjectNodePoints(m.config.ID), points, false)
			if err != nil {
				log.Println("Error sending metrics period:", err)
			}
		}
	}

	checkPeriod()

	if m.config.Type == data.PointValuePrometheus {
		m.checkPromDefaults()
	}

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
			case data.PointValuePrometheus:
				m.promPeriodic()
			default:
				log.Println("Metrics: Must select metric type")
			}

		case pts := <-m.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &m.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			for _, p := range pts.Points {
				switch p.Type {
				case data.PointTypePeriod:
					checkPeriod()
					sampleTicker.Reset(time.Duration(m.config.Period) *
						time.Second)
				case data.PointTypeType:
					switch m.config.Type {
					case data.PointValueSystem:
						m.sysStart()
					case data.PointValuePrometheus:
						m.checkPromDefaults()
					}
				case data.PointTypeURI:
					// a different endpoint is a different set of
					// counters, so what we remember of the old one
					// would produce a meaningless first delta
					m.promReset()
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
	// collect static host stats on startup
	hostStat, err := host.Info()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		// TODO, only send points if they have changed
		pts := data.Points{
			data.NewPointString(data.PointTypeHost, data.PointKeyHostname, hostStat.Hostname),
			data.NewPointFloat(data.PointTypeHostBootTime, "", float64(hostStat.BootTime)),
			data.NewPointString(data.PointTypeHost, data.PointKeyOS, hostStat.OS),
			data.NewPointString(data.PointTypeHost, data.PointKeyPlatform, hostStat.Platform),
			data.NewPointString(data.PointTypeHost, data.PointKeyPlatformFamily, hostStat.PlatformFamily),
			data.NewPointString(data.PointTypeHost, data.PointKeyPlatformVersion, hostStat.PlatformVersion),
			data.NewPointString(data.PointTypeHost, data.PointKeyKernelVersion, hostStat.KernelVersion),
			data.NewPointString(data.PointTypeHost, data.PointKeyKernelArch, hostStat.KernelArch),
			data.NewPointString(data.PointTypeHost, data.PointKeyVirtualizationSystem, hostStat.VirtualizationSystem),
			data.NewPointString(data.PointTypeHost, data.PointKeyVirtualizationRole, hostStat.VirtualizationRole),
		}
		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points:", err)
		}
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		pt := data.NewPointFloat(data.PointTypeMetricSysMem, data.PointKeyTotal, float64(vm.Total))

		err = SendNodePoint(m.nc, m.config.ID, pt, false)
		if err != nil {
			log.Println("Metrics: error sending points:", err)
		}
	}

	// the scale a cooling device state is measured against does not change
	// while the system runs, so collect it here rather than every period
	if pts := sysCoolingMax(); len(pts) > 0 {
		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points:", err)
		}
	}
}

func (m *MetricsClient) sysPeriodic() {
	var pts data.Points

	avg, err := load.Avg()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		pts = append(pts, data.Points{
			data.NewPointFloat(data.PointTypeMetricSysLoad, "1", avg.Load1),
			data.NewPointFloat(data.PointTypeMetricSysLoad, "5", avg.Load5),
			data.NewPointFloat(data.PointTypeMetricSysLoad, "15", avg.Load15),
		}...)

	}

	perc, err := cpu.Percent(time.Duration(m.config.Period)*time.Second, false)
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		pts = append(pts, data.NewPointFloat(data.PointTypeMetricSysCPUPercent, "", perc[0]))
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		pts = append(pts, data.Points{
			data.NewPointFloat(data.PointTypeMetricSysMemUsedPercent, "", vm.UsedPercent),
			data.NewPointFloat(data.PointTypeMetricSysMem, data.PointKeyAvailable, float64(vm.Available)),
			data.NewPointFloat(data.PointTypeMetricSysMem, data.PointKeyUsed, float64(vm.Used)),
			data.NewPointFloat(data.PointTypeMetricSysMem, data.PointKeyFree, float64(vm.Free)),
		}...)
	}

	parts, err := disk.Partitions(false)
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		for _, p := range parts {
			if strings.HasPrefix(p.Mountpoint, "/run/media") {
				// don't track stats for removable media
				continue
			}

			u, err := disk.Usage(p.Mountpoint)
			if err != nil {
				log.Println("Error getting disk usage:", err)
				continue
			}
			pts = append(pts, data.Points{
				data.NewPointFloat(data.PointTypeMetricSysDiskUsedPercent,
					data.SubjectSafeToken(u.Path), u.UsedPercent),
			}...)
		}
	}

	netio, err := net.IOCounters(true)
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		for _, io := range netio {
			// interface names can carry a period -- a VLAN is named
			// eth0.100 -- which a point key cannot hold
			name := data.SubjectSafeToken(io.Name)
			pts = append(pts, data.Points{
				data.NewPointFloat(data.PointTypeMetricSysNetBytesRecv, name, float64(io.BytesRecv)),
				data.NewPointFloat(data.PointTypeMetricSysNetBytesSent, name, float64(io.BytesSent)),
			}...)
		}

	}

	uptime, err := host.Uptime()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		pts = append(pts, data.NewPointFloat(data.PointTypeMetricSysUptime, "", float64(uptime)))
	}

	pts = append(pts, sysTemperatures()...)
	pts = append(pts, sysFans()...)
	pts = append(pts, sysCooling()...)
	pts = append(pts, sysPower()...)
	pts = append(pts, sysCPUFreqs()...)

	err = SendNodePoints(m.nc, m.config.ID, pts, false)
	if err != nil {
		log.Println("Metrics: error sending points:", err)
	}
}

// sysfs directories the metrics client reads directly. They are variables so
// tests can point them at fixture directories.
var (
	thermalPath = "/sys/class/thermal"
	hwmonPath   = "/sys/class/hwmon"
	cpuPath     = "/sys/devices/system/cpu"
)

// reading is a single sensor value along with the name it is published under
type reading struct {
	key string
	val float64
}

// readingPoints converts readings to points of the given type. A point is
// identified by its type and key, so two readings that share a key would
// overwrite each other. Sensor names are not guaranteed to be unique, so
// repeats are numbered: tmp451, tmp451_2, tmp451_3, and so on.
//
// Sensor names come from the kernel and carry characters a point key cannot
// hold -- a cooling device is named devfreq-17000000.gpu, for instance -- so
// they are made safe here. Uniqueness is checked after that, since two names
// can differ only in a character that gets replaced.
func readingPoints(typ string, readings []reading) data.Points {
	pts := make(data.Points, 0, len(readings))
	counts := make(map[string]int)

	for _, r := range readings {
		if r.key == "" {
			continue
		}

		r.key = data.SubjectSafeToken(r.key)

		counts[r.key]++

		key := r.key
		if c := counts[r.key]; c > 1 {
			key = key + "_" + strconv.Itoa(c)
		}

		pts = append(pts, data.NewPointFloat(typ, key, r.val))
	}

	return pts
}

// sysTemperatures collects temperature readings from the hwmon sensors gopsutil
// finds as well as from the Linux thermal zones. The zones are read directly
// because gopsutil only consults them when a system has no hwmon temperature
// inputs at all. Boards that have both, such as the Jetson AGX Orin, report
// their SoC, CPU, and junction temperatures through the zones alone, and those
// are usually the readings worth watching.
func sysTemperatures() data.Points {
	var temps []reading

	// gopsutil returns the sensors it did read along with an error describing
	// the ones it could not, so use the readings that came back either way
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		log.Println("Metrics: some sensors were not read:", err)
	}

	for _, s := range sensors {
		temps = append(temps, reading{key: s.SensorKey, val: s.Temperature})
	}

	temps = append(temps, thermalZones(thermalPath)...)

	return readingPoints(data.PointTypeTemperature, temps)
}

// sysFans collects fan tachometer and drive levels from the hwmon interface
func sysFans() data.Points {
	rpm, pwm := fans(hwmonPath)

	pts := readingPoints(data.PointTypeMetricSysFanSpeed, rpm)

	return append(pts, readingPoints(data.PointTypeMetricSysFanPWM, pwm)...)
}

// sysPower collects the rail voltages, currents, and power the hwmon power
// monitors report
func sysPower() data.Points {
	volts, amps, watts := powerRails(hwmonPath)

	pts := readingPoints(data.PointTypeVoltage, volts)
	pts = append(pts, readingPoints(data.PointTypeCurrent, amps)...)

	return append(pts, readingPoints(data.PointTypePower, watts)...)
}

// sysCPUFreqs collects the current clock of each CPU
func sysCPUFreqs() data.Points {
	return readingPoints(data.PointTypeMetricSysCPUFreq, cpuFreqs(cpuPath))
}

// sysCooling collects the current state of each thermal cooling device
func sysCooling() data.Points {
	state, _ := coolingDevices(thermalPath)

	return readingPoints(data.PointTypeMetricSysCoolingState, state)
}

// sysCoolingMax collects the highest state each cooling device supports. This
// does not change while the system runs, so it is collected once at startup.
func sysCoolingMax() data.Points {
	_, stateMax := coolingDevices(thermalPath)

	return readingPoints(data.PointTypeMetricSysCoolingStateMax, stateMax)
}

// thermalZones reads every zone under the given sysfs thermal directory. A zone
// whose sensor is unavailable, which happens on SoCs when a rail is powered
// down, is skipped so that it does not cost us the zones that did read. Systems
// without the thermal interface simply return no readings.
func thermalZones(dir string) []reading {
	zones, err := filepath.Glob(filepath.Join(dir, "thermal_zone*"))
	if err != nil {
		// only happens if the pattern above is malformed
		log.Println("Metrics: error listing thermal zones:", err)
		return nil
	}

	ret := make([]reading, 0, len(zones))

	for _, z := range zones {
		typ, err := readSysfsString(filepath.Join(z, "type"))
		if err != nil {
			continue
		}

		milliC, err := readSysfsNumber(filepath.Join(z, "temp"))
		if err != nil {
			continue
		}

		ret = append(ret, reading{key: typ, val: milliC / 1000})
	}

	return ret
}

// coolingDevices reads the current and maximum state of every cooling device
// under the given sysfs thermal directory. A state above zero means the thermal
// governor is limiting the system: a cpufreq or devfreq device reports how many
// steps the clocks have been pulled back, and a fan reports how hard it has been
// asked to run. That is the reading that tells us whether a warm system is
// giving up performance, which temperature alone cannot answer.
func coolingDevices(dir string) (state, stateMax []reading) {
	devices, err := filepath.Glob(filepath.Join(dir, "cooling_device*"))
	if err != nil {
		// only happens if the pattern above is malformed
		log.Println("Metrics: error listing cooling devices:", err)
		return nil, nil
	}

	for _, d := range devices {
		typ, err := readSysfsString(filepath.Join(d, "type"))
		if err != nil {
			continue
		}

		if cur, err := readSysfsNumber(filepath.Join(d, "cur_state")); err == nil {
			state = append(state, reading{key: typ, val: cur})
		}

		if m, err := readSysfsNumber(filepath.Join(d, "max_state")); err == nil {
			stateMax = append(stateMax, reading{key: typ, val: m})
		}
	}

	return state, stateMax
}

// fans reads fan tachometer and drive levels from the hwmon devices under the
// given directory. Speed is reported in RPM through fan*_input on most drivers,
// while some, including the Tegra tachometer, use a plain rpm file instead.
// Drive level is the raw hwmon pwm value, which runs from 0 to 255.
func fans(dir string) (rpm, pwm []reading) {
	devices, err := filepath.Glob(filepath.Join(dir, "hwmon*"))
	if err != nil {
		// only happens if the pattern above is malformed
		log.Println("Metrics: error listing hwmon devices:", err)
		return nil, nil
	}

	for _, d := range devices {
		// the name file is what the driver calls itself, such as pwmfan
		name, err := readSysfsString(filepath.Join(d, "name"))
		if err != nil || name == "" {
			name = filepath.Base(d)
		}

		// fan1_input, fan2_input, ... and the nonstandard rpm
		speeds, _ := filepath.Glob(filepath.Join(d, "fan*_input"))
		speeds = append(speeds, filepath.Join(d, "rpm"))

		for _, f := range speeds {
			if v, err := readSysfsNumber(f); err == nil {
				rpm = append(rpm, reading{key: name, val: v})
			}
		}

		// pwm1, pwm2, ... but not the pwm1_enable and pwm1_mode settings
		// that sit alongside them
		drives, _ := filepath.Glob(filepath.Join(d, "pwm[0-9]"))

		for _, f := range drives {
			if v, err := readSysfsNumber(f); err == nil {
				pwm = append(pwm, reading{key: name, val: v})
			}
		}
	}

	return rpm, pwm
}

// powerRails reads the labeled channels of the hwmon power monitors under the
// given directory, such as the INA3221 devices on a Jetson. A channel is
// published when its driver gives it a label, which is how a board names the
// rail that channel measures. The unlabeled channels these devices also expose
// carry shunt voltages, which say little on their own. Readings are converted
// to volts, amps, and watts. Drivers that do not report power directly still
// give us voltage and current, so the product stands in for it.
func powerRails(dir string) (volts, amps, watts []reading) {
	devices, err := filepath.Glob(filepath.Join(dir, "hwmon*"))
	if err != nil {
		// only happens if the pattern above is malformed
		log.Println("Metrics: error listing hwmon devices:", err)
		return nil, nil, nil
	}

	for _, d := range devices {
		labels, _ := filepath.Glob(filepath.Join(d, "in[0-9]_label"))

		for _, l := range labels {
			name, err := readSysfsString(l)
			if err != nil || name == "" {
				continue
			}

			// a key with spaces in it is awkward to work with downstream
			name = strings.Join(strings.Fields(name), "_")

			// the label on in3 names channel 3, whose current is curr3
			ch := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(l), "_label"), "in")

			var v, a float64
			var haveV, haveA bool

			if mV, err := readSysfsNumber(filepath.Join(d, "in"+ch+"_input")); err == nil {
				v, haveV = mV/1000, true
				volts = append(volts, reading{key: name, val: v})
			}

			if mA, err := readSysfsNumber(filepath.Join(d, "curr"+ch+"_input")); err == nil {
				a, haveA = mA/1000, true
				amps = append(amps, reading{key: name, val: a})
			}

			if uW, err := readSysfsNumber(filepath.Join(d, "power"+ch+"_input")); err == nil {
				watts = append(watts, reading{key: name, val: uW / 1000000})
			} else if haveV && haveA {
				watts = append(watts, reading{key: name, val: v * a})
			}
		}
	}

	return volts, amps, watts
}

// cpuFreqs reads the current clock of each CPU under the given sysfs directory,
// in MHz. Alongside the cooling device states, this shows where the clocks
// actually settled once the thermal governor pulled them back. Cores that are
// offline drop out on their own, as their attribute cannot be read.
func cpuFreqs(dir string) []reading {
	cpus, err := filepath.Glob(filepath.Join(dir, "cpu[0-9]*"))
	if err != nil {
		// only happens if the pattern above is malformed
		log.Println("Metrics: error listing CPUs:", err)
		return nil
	}

	ret := make([]reading, 0, len(cpus))

	for _, c := range cpus {
		kHz, err := readSysfsNumber(filepath.Join(c, "cpufreq", "scaling_cur_freq"))
		if err != nil {
			continue
		}

		ret = append(ret, reading{key: filepath.Base(c), val: kHz / 1000})
	}

	return ret
}

// readSysfsString reads a sysfs attribute and trims the trailing newline
func readSysfsString(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(raw)), nil
}

// readSysfsNumber reads a numeric sysfs attribute. Attributes whose sensor is
// unavailable return an error on read, so callers skip them individually rather
// than losing the readings that succeeded.
func readSysfsNumber(path string) (float64, error) {
	s, err := readSysfsString(path)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(s, 64)
}

// if procName is "", then collect stats for this app
func (m *MetricsClient) appPeriodic(procName string) {

	if procName == "" {
		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		numGoRoutine := runtime.NumGoroutine()

		pts := data.Points{
			data.NewPointFloat(data.PointTypeMetricAppAlloc, "", float64(memStats.Alloc)),
			data.NewPointFloat(data.PointTypeMetricAppNumGoroutine, "", float64(numGoRoutine)),
		}

		err := SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points:", err)
		}
	}

	pid := os.Getpid()

	procs, err := process.Processes()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		var accumCPUPerc, accumMemPerc, accumMemRSS float64
		var procCount int
		for _, p := range procs {
			if procName != "" {
				name, err := p.Name()
				if err != nil {
					log.Println("Error getting process name:", err)
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
				log.Println("Error getting CPU percent for proc:", err)
				break
			}

			accumCPUPerc += cpuPerc

			memPerc, err := p.MemoryPercent()
			if err != nil {
				log.Println("Error getting mem percent for proc:", err)
				break
			}

			accumMemPerc += float64(memPerc)

			memInfo, err := p.MemoryInfo()
			if err != nil {
				log.Println("Error getting mem info:", err)
				break
			}

			accumMemRSS += float64(memInfo.RSS)
		}

		pts := data.Points{
			data.NewPointFloat(data.PointTypeMetricProcCPUPercent, "", float64(accumCPUPerc)),
			data.NewPointFloat(data.PointTypeMetricProcMemPercent, "", float64(accumMemPerc)),
			data.NewPointFloat(data.PointTypeMetricProcMemRSS, "", float64(accumMemRSS)),
		}

		if procName != "" {
			pts = append(pts, data.NewPointFloat(data.PointTypeCount, "", float64(procCount)))
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points:", err)
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

	metrics := make(map[string]procMetrics)

	procs, err := process.Processes()
	if err != nil {
		log.Println("Metrics error:", err)
	} else {
		for _, p := range procs {
			name, err := p.Name()
			if err != nil {
				log.Println("Error getting process name:", err)
				continue
			}

			m := metrics[name]

			m.count++

			cpuPerc, err := p.CPUPercent()
			if err != nil {
				log.Println("Error getting CPU percent for proc:", err)
				break
			}

			m.cpu += cpuPerc

			memPerc, err := p.MemoryPercent()
			if err != nil {
				log.Println("Error getting mem percent for proc:", err)
				break
			}

			m.mem += float64(memPerc)

			memInfo, err := p.MemoryInfo()
			if err != nil {
				log.Println("Error getting mem info:", err)
				break
			}

			m.rss += float64(memInfo.RSS)

			metrics[name] = m
		}

		pts := make(data.Points, len(metrics)*4)
		var i int
		for k, v := range metrics {
			pts[i].Key = k
			pts[i].Type = data.PointTypeMetricProcCPUPercent
			pts[i].PutFloat(v.cpu)
			i++

			pts[i].Key = k
			pts[i].Type = data.PointTypeMetricProcMemPercent
			pts[i].PutFloat(v.mem)
			i++

			pts[i].Key = k
			pts[i].Type = data.PointTypeMetricProcMemRSS
			pts[i].PutFloat(v.rss)
			i++

			pts[i].Key = k
			pts[i].Type = data.PointTypeCount
			pts[i].PutFloat(v.count)
			i++
		}

		err = SendNodePoints(m.nc, m.config.ID, pts, false)
		if err != nil {
			log.Println("Metrics: error sending points:", err)
		}
	}
}
