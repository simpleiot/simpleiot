package device

import (
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// Manager is responsible for maintaining device state
func Manager(db *db.Db) {
	for {
		db.DeviceEach(func(device *data.Device) error {
			changed := device.UpdateState()
			if changed {
				err := db.DeviceSetState(device.ID, device.State.SysState)
				if err != nil {
					log.Println("Error updating device state: ", err)
				}
			}
			return nil
		})

		time.Sleep(10 * time.Second)

	}
}
