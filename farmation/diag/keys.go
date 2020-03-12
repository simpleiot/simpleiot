package diag

import (
	"errors"
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/keypad"
)

type keys struct{}

func (d keys) String() string {
	return "keys"
}

func (d keys) Run() error {

	appChan := make(chan interface{}, 100)
	keypadChan := make(chan interface{}, 100)

	go keypad.Run(appChan, keypadChan, 1)

	ignoredKeys := []isdata.Key{
		isdata.KeyUpHold,
		isdata.KeyUpRelease,
		isdata.KeyDownHold,
		isdata.KeyDownRelease,
		isdata.KeyLeftHold,
		isdata.KeyLeftRelease,
		isdata.KeyRightHold,
		isdata.KeyRightRelease,
		isdata.KeyEnterHold,
		isdata.KeyEnterRelease,
		isdata.KeySK1Hold,
		isdata.KeySK1Release,
		isdata.KeySK2Hold,
		isdata.KeySK2Release,
		isdata.KeySK3Hold,
		isdata.KeySK3Release,
		isdata.KeySK4Hold,
		isdata.KeySK4Release,
		isdata.KeyArmHold,
		isdata.KeyArmRelease,
		isdata.KeyArmKpHold,
		isdata.KeyArmKpRelease,
		isdata.KeyPumpHold,
		isdata.KeyPumpRelease,
	}

	isIgnored := func(key isdata.Key) bool {
		for _, k := range ignoredKeys {
			if key == k {
				return true
			}
		}

		return false
	}

	expectedKeys := []isdata.Key{
		isdata.KeyArmKp,
		isdata.KeyPump,
		isdata.KeySK1,
		isdata.KeySK2,
		isdata.KeySK3,
		isdata.KeySK4,
		isdata.KeyLeft,
		isdata.KeyUp,
		isdata.KeyRight,
		isdata.KeyDown,
		isdata.KeyEnter,
	}

	keyIndex := 0

	timeout := time.NewTimer(time.Second * 30)

	fmt.Println("Press keys in sequence: arm pump sk1 sk2 sk3 sk4 left up right down enter")

	for {
		select {
		case m := <-keypadChan:
			switch m := m.(type) {
			case isdata.Key:
				if isIgnored(m) {
					continue
				}
				fmt.Println("Key: ", m)
				if m != expectedKeys[keyIndex] {
					return errors.New("did not get key: " + expectedKeys[keyIndex].String())
				}
				keyIndex++

				if keyIndex >= len(expectedKeys) {
					fmt.Println("all keys detected")
					return nil
				}
			}
		case <-timeout.C:
			return errors.New("Timeout waiting for keys")
		}
	}
}

func init() {
	Register(keys{})
}
