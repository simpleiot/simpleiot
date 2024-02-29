package client

// below is code that used to be in the store and is in process of being
// ported to a client

// StartUpdate starts an update
/*
func StartUpdate(id, url string) error {
	if _, ok := st.updates[id]; ok {
		return fmt.Errorf("Update already in process for dev: %v", id)
	}

	st.updates[id] = time.Now()

	err := st.setSwUpdateState(id, data.SwUpdateState{
		Running: true,
	})

	if err != nil {
		delete(st.updates, id)
		return err
	}

	go func() {
		err := NatsSendFileFromHTTP(st.nc, id, url, func(bytesTx int) {
			err := st.setSwUpdateState(id, data.SwUpdateState{
				Running:     true,
				PercentDone: bytesTx,
			})

			if err != nil {
				log.Println("Error setting update status in DB:", err)
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

		st.lock.Lock()
		delete(st.updates, id)
		st.lock.Unlock()

		err = st.setSwUpdateState(id, state)
		if err != nil {
			log.Println("Error setting sw update state:", err)
		}
	}()

	return nil
}
*/
