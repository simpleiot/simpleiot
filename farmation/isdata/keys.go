package isdata

// Key defines keypad inputs
type Key int

// define valid keys
const (
	KeyUnknown Key = iota
	KeyUp          //Press
	KeyUpRelease
	KeyDown
	KeyDownRelease
	KeyLeft
	KeyLeftRelease
	KeyRight
	KeyRightRelease
	KeyEnter
	KeyEnterRelease
	KeySK1
	KeySK1Release
	KeySK2
	KeySK2Release
	KeySK3
	KeySK3Release
	KeySK4
	KeySK4Release
	KeyArm
	KeyArmRelease
)

var keyToString = map[Key]string{
	KeyUp:           "KeyUp",
	KeyUpRelease:    "KeyUpRelease",
	KeyDown:         "KeyDown",
	KeyDownRelease:  "KeyDownRelease",
	KeyRight:        "KeyRight",
	KeyRightRelease: "KeyRightRelease",
	KeyLeft:         "KeyLeft",
	KeyLeftRelease:  "KeyLeftRelease",
	KeyEnter:        "KeyEnter",
	KeyEnterRelease: "KeyEnterRelease",
	KeySK1:          "KeySK1",
	KeySK1Release:   "KeySK1Release",
	KeySK2:          "KeySK2",
	KeySK2Release:   "KeySK2Release",
	KeySK3:          "KeySK3",
	KeySK3Release:   "KeySK3Release",
	KeySK4:          "KeySK4",
	KeySK4Release:   "KeySK4Release",
	KeyArm:          "KeyArm",
	KeyArmRelease:   "KeyArmRelease",
}

func (k Key) String() string {
	s := keyToString[k]
	if s == "" {
		s = "unknown"
	}
	return s
}

var stringToKey = map[string]Key{
	"KeyUp":           KeyUp,
	"KeyUpRelease":    KeyUpRelease,
	"KeyDown":         KeyDown,
	"KeyDownRelease":  KeyDownRelease,
	"KeyRight":        KeyRight,
	"KeyRightRelease": KeyRightRelease,
	"KeyLeft":         KeyLeft,
	"KeyLeftRelease":  KeyLeftRelease,
	"KeyEnter":        KeyEnter,
	"KeyEnterRelease": KeyEnterRelease,
	"KeySK1":          KeySK1,
	"KeySK1Release":   KeySK1Release,
	"KeySK2":          KeySK2,
	"KeySK2Release":   KeySK2Release,
	"KeySK3":          KeySK3,
	"KeySK3Release":   KeySK3Release,
	"KeySK4":          KeySK4,
	"KeySK4Release":   KeySK4Release,
	"KeyArm":          KeyArm,
	"KeyArmRelease":   KeyArmRelease,
}

// KeyFromString converts a string to a key
func KeyFromString(s string) Key {
	return stringToKey[s]
}
