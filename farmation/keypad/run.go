package keypad

import (
	"bytes"

	"github.com/pkg/term"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// Get character as a byte slice
func getch() []byte {
	t, _ := term.Open("/dev/tty")
	term.RawMode(t)
	bytes := make([]byte, 3)
	numRead, err := t.Read(bytes)
	t.Restore()
	t.Close()
	if err != nil {
		return nil
	}
	return bytes[0:numRead]
}

// Run goroutine for keypad code
func Run(in, out chan interface{}) {
	go func() {
		for {
			c := getch()
			switch {
			case bytes.Equal(c, []byte{3}):
				return
			case bytes.Equal(c, []byte{27, 91, 65}): // up arrow
				out <- isdata.KeyUp
			case bytes.Equal(c, []byte{27, 91, 66}): // down
				out <- isdata.KeyDown
			case bytes.Equal(c, []byte{27, 91, 67}): // right
				out <- isdata.KeyRight
			case bytes.Equal(c, []byte{27, 91, 68}): // left
				out <- isdata.KeyLeft
			case bytes.Equal(c, []byte{13}): // enter
				out <- isdata.KeyEnter
			case bytes.Equal(c, []byte{49}): // 1
				out <- isdata.KeySK1
			case bytes.Equal(c, []byte{50}): // 2
				out <- isdata.KeySK2
			case bytes.Equal(c, []byte{51}): // 3
				out <- isdata.KeySK3
			case bytes.Equal(c, []byte{52}): // 4
				out <- isdata.KeySK4
			}
		}
	}()
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.Sample:
				// ... todo
				_ = m
			}
		}
	}
}
