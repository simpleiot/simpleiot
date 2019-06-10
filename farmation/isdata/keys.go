package isdata

// Key defines keypad inputs
type Key int

// define valid keys
const (
	KeyUnknown Key = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeySK1
	KeySK1Release
	KeySK1Hold
	KeySK2
	KeySK2Release
	KeySK3
	KeySK3Release
	KeySK4
	KeySK4Release
	KeyArm
)

var keyToString = map[Key]string{
	KeyUp:    "KeyUp",
	KeyDown:  "KeyDown",
	KeyRight: "KeyRight",
	KeyLeft:  "KeyLeft",
	KeyEnter: "KeyEnter",
	KeySK1:   "KeySK1",
	KeySK2:   "KeySK2",
	KeySK3:   "KeySK3",
	KeySK4:   "KeySK4",
	KeyArm:   "KeyArm",
}

func (k Key) String() string {
	s := keyToString[k]
	if s == "" {
		s = "unknown"
	}
	return s
}

var stringToKey = map[string]Key{
	"KeyUp":    KeyUp,
	"KeyDown":  KeyDown,
	"KeyRight": KeyRight,
	"KeyLeft":  KeyLeft,
	"KeyEnter": KeyEnter,
	"KeySK1":   KeySK1,
	"KeySK2":   KeySK2,
	"KeySK3":   KeySK3,
	"KeySK4":   KeySK4,
	"KeyArm":   KeyArm,
}

// KeyFromString converts a string to a key
func KeyFromString(s string) Key {
	return stringToKey[s]
}
