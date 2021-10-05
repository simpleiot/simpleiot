package node

import (
	"log"
	"runtime/metrics"
	"time"

	natsgo "github.com/nats-io/nats.go"
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
		samples[2].Name = "/gc/heap/allocs:bytes"
		for {
			time.Sleep(10 * time.Second)
			metrics.Read(samples)
			numGoRoutines := samples[0].Value.Uint64()
			mem := samples[1].Value.Uint64()
			heap := samples[2].Value.Uint64()
			err := ret.SendPoint(id, data.PointTypeMetricGoGoroutines, float64(numGoRoutines))
			if err != nil {
				log.Println("Error sending go routine count metric: ", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricGoTotalBytes, float64(mem))
			if err != nil {
				log.Println("Error sending mem metric: ", err)
			}

			err = ret.SendPoint(id, data.PointTypeMetricGoHeapAllocBytes, float64(heap))
			if err != nil {
				log.Println("Error sending heap alloc metric: ", err)
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
