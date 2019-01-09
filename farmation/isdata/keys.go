package isdata

// Key defines keypad inputs
type Key int

// define valid keys
const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeySK1
	KeySK2
	KeySK3
	KeySK4
)

func (k Key) String() string {
	switch k {
	case KeyUp:
		return "KeyUp"
	case KeyDown:
		return "KeyDown"
	case KeyRight:
		return "KeyRight"
	case KeyLeft:
		return "KeyLeft"
	case KeyEnter:
		return "KeyEnter"
	case KeySK1:
		return "KeySK1"
	case KeySK2:
		return "KeySK2"
	case KeySK3:
		return "KeySK3"
	case KeySK4:
		return "KeySK4"
	default:
		return "unknown key"
	}
}
