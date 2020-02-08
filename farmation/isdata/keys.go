package isdata

// Key defines keypad inputs
type Key int

// define valid keys
const (
	KeyUnknown Key = iota
	KeyUp          //Press
	KeyUpHold
	KeyUpRelease
	KeyDown
	KeyDownHold
	KeyDownRelease
	KeyLeft
	KeyLeftHold
	KeyLeftRelease
	KeyRight
	KeyRightHold
	KeyRightRelease
	KeyEnter
	KeyEnterHold
	KeyEnterRelease
	KeySK1
	KeySK1Hold
	KeySK1Release
	KeySK2
	KeySK2Hold
	KeySK2Release
	KeySK3
	KeySK3Hold
	KeySK3Release
	KeySK4
	KeySK4Hold
	KeySK4Release
	KeyArm
	KeyArmHold
	KeyArmRelease
	KeyArmKp
	KeyArmKpHold
	KeyArmKpRelease
	KeyPump
	KeyPumpHold
	KeyPumpRelease
)

var keyToString = map[Key]string{
	KeyUp:           "KeyUp",
	KeyUpHold:       "KeyUpHold",
	KeyUpRelease:    "KeyUpRelease",
	KeyDown:         "KeyDown",
	KeyDownHold:     "KeyDownHold",
	KeyDownRelease:  "KeyDownRelease",
	KeyRight:        "KeyRight",
	KeyRightHold:    "KeyRightHold",
	KeyRightRelease: "KeyRightRelease",
	KeyLeft:         "KeyLeft",
	KeyLeftHold:     "KeyLeftHold",
	KeyLeftRelease:  "KeyLeftRelease",
	KeyEnter:        "KeyEnter",
	KeyEnterHold:    "KeyEnterHold",
	KeyEnterRelease: "KeyEnterRelease",
	KeySK1:          "KeySK1",
	KeySK1Hold:      "KeySK1Hold",
	KeySK1Release:   "KeySK1Release",
	KeySK2:          "KeySK2",
	KeySK2Hold:      "KeySK2Hold",
	KeySK2Release:   "KeySK2Release",
	KeySK3:          "KeySK3",
	KeySK3Hold:      "KeySK3Hold",
	KeySK3Release:   "KeySK3Release",
	KeySK4:          "KeySK4",
	KeySK4Hold:      "KeySK4Hold",
	KeySK4Release:   "KeySK4Release",
	KeyArm:          "KeyArm",
	KeyArmHold:      "KeyArmHold",
	KeyArmRelease:   "KeyArmRelease",
	KeyArmKp:        "KeyArmKp",
	KeyArmKpHold:    "KeyArmKpHold",
	KeyArmKpRelease: "KeyArmKpRelease",
	KeyPump:         "KeyPump",
	KeyPumpHold:     "KeyPumpHold",
	KeyPumpRelease:  "KeyPumpRelease",
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
	"KeyArmKp":        KeyArmKp,
	"KeyArmKpRelease": KeyArmKpRelease,
	"KeyPump":         KeyPump,
	"KeyPumpRelease":  KeyPumpRelease,
}

// KeyFromString converts a string to a key
func KeyFromString(s string) Key {
	return stringToKey[s]
}

// KeyReleaseFromString returns the corresponding
// ...Release key for any key string
func KeyReleaseFromString(s string) Key {
	return KeyFromString(s + "Release")
}
