package api

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// NatsHandler implements the SIOT NATS api
type NatsHandler struct {
	Nc        *nats.Conn
	db        *db.Db
	authToken string
	lock      sync.Mutex
	updates   map[string]time.Time
}

// NewNatsHandler creates a new NATS client for handling SIOT requests
func NewNatsHandler(db *db.Db, authToken string) *NatsHandler {
	return &NatsHandler{
		db:        db,
		authToken: authToken,
		updates:   make(map[string]time.Time),
	}
}

// Connect to NATS server and set up handlers for things we are interested in
func (nh *NatsHandler) Connect(server string) error {
	nc, err := nats.Connect(server,
		nats.Timeout(10*time.Second),
		nats.PingInterval(60*5*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5*1024*1024),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		nats.Token(nh.authToken),
	)

	if err != nil {
		log.Fatal("Error connecting to nats server: ", err)
	}

	nh.Nc = nc

	if _, err := nc.Subscribe("device.*.samples", nh.handleSamples); err != nil {
		return fmt.Errorf("Subscribe device samples error: %w", err)
	}

	if _, err := nc.Subscribe("device.*.version", nh.handleVersion); err != nil {
		return fmt.Errorf("Subscribe device version error: %w", err)
	}

	return nil
}

// StartUpdate starts an update
func (nh *NatsHandler) StartUpdate(id, url string) error {
	nh.lock.Lock()
	defer nh.lock.Unlock()

	if _, ok := nh.updates[id]; ok {
		return fmt.Errorf("Update already in process for dev: %v", id)
	}

	nh.updates[id] = time.Now()

	err := nh.db.DeviceSetSwUpdateState(id, data.SwUpdateState{
		Running: true,
	})

	if err != nil {
		delete(nh.updates, id)
		return err
	}

	go func() {
		err := NatsSendFileFromHTTP(nh.Nc, id, url, func(bytesTx int) {
			err := nh.db.DeviceSetSwUpdateState(id, data.SwUpdateState{
				Running:     true,
				PercentDone: bytesTx,
			})

			if err != nil {
				log.Println("Error setting update status in DB: ", err)
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

		nh.lock.Lock()
		delete(nh.updates, id)
		nh.lock.Unlock()

		err = nh.db.DeviceSetSwUpdateState(id, state)
		if err != nil {
			log.Println("Error setting sw update state: ", err)
		}
	}()

	return nil
}

func (nh *NatsHandler) handleSamples(msg *nats.Msg) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		log.Println("Error decoding device asmples subject: ", msg.Subject)
		return
	}
	deviceID := chunks[1]
	samples, err := data.PbDecodeSamples(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb Samples: ", err)
		return
	}

	err = nh.db.DeviceActivity(deviceID)
	if err != nil {
		log.Println("Error updating device activity: ", err)
		return
	}
	for _, s := range samples {
		err = nh.db.DeviceSample(deviceID, s)
		if err != nil {
			log.Println("Error writting sample to Db: ", err)
			return
		}
	}

}

func (nh *NatsHandler) handleVersion(msg *nats.Msg) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		log.Println("Error decoding device version subject: ", msg.Subject)
		return
	}
	deviceID := chunks[1]

	vPb := &pb.DeviceVersion{}
	err := proto.Unmarshal(msg.Data, vPb)

	err = nh.db.DeviceActivity(deviceID)
	if err != nil {
		log.Println("Error updating device activity: ", err)
		return
	}

	v := data.DeviceVersion{
		OS:  vPb.Os,
		App: vPb.App,
		HW:  vPb.Hw,
	}

	err = nh.db.DeviceSetVersion(deviceID, v)
	if err != nil {
		log.Println("Error setting device version: ", err)
	}
}
