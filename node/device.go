package node

import (
	"log"
	"runtime/metrics"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/prometheus/procfs"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

// RootDevice is used to manage the device SIOT is running on
type RootDevice struct {
	// data associated with running the bus
	id string
	nc *natsgo.Conn
}

// NewRootDevice is used to create a new root device
func NewRootDevice(nc *natsgo.Conn, id string) *RootDevice {
	ret := &RootDevice{
		id: id,
		nc: nc,
	}

	go func(id string) {
		samples := make([]metrics.Sample, 3)
		samples[0].Name = "/sched/goroutines:goroutines"
		samples[1].Name = "/memory/classes/total:bytes"
		for {
			time.Sleep(10 * time.Second)
			metrics.Read(samples)
			numGoRoutines := samples[0].Value.Uint64()
			mem := samples[1].Value.Uint64()
			err := ret.SendPoint(id, data.PointTypeMetricGoGoroutines, float64(numGoRoutines))
			if err != nil {
				log.Println("Error sending go routine count metric: ", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricGoTotalBytes, float64(mem))
			if err != nil {
				log.Println("Error sending mem metric: ", err)
			}

			p, err := procfs.Self()
			if err != nil {
				log.Fatalf("could not get process: %s", err)
			}

			smap, err := p.ProcSMapsRollup()
			if err != nil {
				log.Fatalf("could not get process smap rollup: %s", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricProcRss, float64(smap.Rss))
			if err != nil {
				log.Println("Error sending proc rss: ", err)
			}

			fs, err := procfs.NewDefaultFS()
			if err != nil {
				log.Fatal("cound not get procfs: ", err)
			}

			load, err := fs.LoadAvg()
			if err != nil {
				log.Fatal("cound not get load avg: ", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricSysLoad1, float64(load.Load1))
			if err != nil {
				log.Fatal("Error sending sys load1: ", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricSysLoad5, float64(load.Load5))
			if err != nil {
				log.Fatal("Error sending sys load1: ", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricSysLoad15, float64(load.Load15))
			if err != nil {
				log.Fatal("Error sending sys load15: ", err)
			}
		}
	}(id)

	return ret
}

// SendPoint sends a point over nats
func (rd *RootDevice) SendPoint(nodeID, pointType string, value float64) error {
	// send the point
	p := data.Point{
		Time:  time.Now(),
		Type:  pointType,
		Value: value,
	}

	return nats.SendNodePoint(rd.nc, nodeID, p, false)
}
