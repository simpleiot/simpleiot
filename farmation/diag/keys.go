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

	expectedKeys := []isdata.Key{
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

	fmt.Println("Press keys in sequence: sk1 sk2 sk3 sk4 left up right down enter")

	for {
		select {
		case m := <-keypadChan:
			switch m := m.(type) {
			case isdata.Key:
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
