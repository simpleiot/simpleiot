package client

import (
	"bytes"
	"errors"
	"io"
	"log"

	"github.com/simpleiot/simpleiot/test"
)

// DefaultShellPrompt is the prompt Zephyr's shell uses on a UART backend.
// It is stripped from the start of received lines so a line that arrives
// right after a prompt is still parsed.
const DefaultShellPrompt = "uart:~$ "

// ErrLineTooLong indicates we received more data than maxMessageLength
// without seeing a line terminator.
var ErrLineTooLong = errors.New("line decode: too much data without a newline")

// LineWrapper wraps an io.ReadWriteCloser and turns a stream of console
// output into discrete lines. Each Read returns exactly one line with the
// terminator, any VT100 escape sequences, and any leading shell prompt
// removed.
//
// This is the framing layer for the Zephyr shell protocol, where the console
// carries log messages, banners, and command output alongside point data.
// Nothing here decides what a line means; it only delivers whole lines.
type LineWrapper struct {
	dev              io.ReadWriteCloser
	buf              bytes.Buffer
	debug            int
	maxMessageLength int
	prompt           []byte
	// overflow is set once a line exceeds maxMessageLength, and stays set
	// until the next terminator so the remainder is discarded rather than
	// returned as a spurious short line.
	overflow bool
}

// NewLineWrapper creates a new line wrapper.
func NewLineWrapper(dev io.ReadWriteCloser, maxMessageLength int) *LineWrapper {
	if maxMessageLength <= 0 {
		maxMessageLength = 1024
	}
	ret := LineWrapper{
		dev:              dev,
		maxMessageLength: maxMessageLength,
		prompt:           []byte(DefaultShellPrompt),
	}
	ret.buf.Grow(maxMessageLength)
	return &ret
}

// SetDebug sets the debug level. At >= 9 the raw data read from and written
// to the port is dumped.
func (lw *LineWrapper) SetDebug(debug int) {
	lw.debug = debug
}

// SetPrompt sets the shell prompt stripped from the start of received lines.
// Set to "" to disable prompt stripping.
func (lw *LineWrapper) SetPrompt(prompt string) {
	lw.prompt = []byte(prompt)
}

// stripEscapes removes VT100/ANSI escape sequences from a line. The connect
// handshake asks the shell to turn colors off, but firmware built with them
// on, or a session already in progress, should not produce garbage.
func stripEscapes(b []byte) []byte {
	if bytes.IndexByte(b, 0x1b) < 0 {
		return b
	}

	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] != 0x1b {
			out = append(out, b[i])
			continue
		}

		// ESC [ ... <final byte in 0x40-0x7e> is a CSI sequence
		if i+1 < len(b) && b[i+1] == '[' {
			i += 2
			for i < len(b) && (b[i] < 0x40 || b[i] > 0x7e) {
				i++
			}
			// i now indexes the final byte, which the loop skips
			continue
		}

		// ESC followed by a single character, or a trailing ESC
		i++
	}
	return out
}

// clean turns a raw line into the form the protocol layer expects: escape
// sequences removed, leading prompts removed, and surrounding whitespace
// trimmed. A prompt can appear more than once when the shell redraws.
func (lw *LineWrapper) clean(b []byte) []byte {
	b = stripEscapes(b)
	b = bytes.TrimRight(b, "\r\n\t ")

	// Match the prompt without its trailing space, since trimming above has
	// already removed that space on a line that is nothing but a prompt.
	promptTrim := bytes.TrimRight(lw.prompt, " \t")
	for len(promptTrim) > 0 && bytes.HasPrefix(b, promptTrim) {
		b = bytes.TrimLeft(b[len(promptTrim):], " \t")
	}

	return bytes.TrimSpace(b)
}

// takeLine pulls the next complete line out of the internal buffer, if there
// is one. Returns the cleaned line and whether a line was available.
func (lw *LineWrapper) takeLine() ([]byte, bool, error) {
	data := lw.buf.Bytes()
	i := bytes.IndexByte(data, '\n')
	if i < 0 {
		// no complete line yet -- fail early if it can never fit
		if lw.buf.Len() > lw.maxMessageLength {
			lw.buf.Reset()
			lw.overflow = true
			return nil, false, ErrLineTooLong
		}
		return nil, false, nil
	}

	line := make([]byte, i)
	copy(line, data[:i])
	lw.buf.Next(i + 1)

	if lw.overflow {
		// tail of a line we already reported as too long
		lw.overflow = false
		return nil, false, nil
	}

	// A complete but oversized line: the terminator arrived in the same read
	// as the body, so the incremental check above never fired. Report it and
	// drop it rather than passing a line the protocol layer cannot use.
	if i > lw.maxMessageLength {
		return nil, false, ErrLineTooLong
	}

	return lw.clean(line), true, nil
}

// Read returns the next line from the port. It blocks until a complete line
// is available or the underlying port returns an error. Blank lines are
// skipped, since a console produces plenty of them and they carry no meaning.
//
// b must be large enough to hold the line; a line that does not fit is
// reported as ErrLineTooLong rather than being split across reads.
func (lw *LineWrapper) Read(b []byte) (int, error) {
	for {
		line, ok, err := lw.takeLine()
		if err != nil {
			return 0, err
		}
		if ok {
			if len(line) == 0 {
				continue
			}
			if len(line) > len(b) {
				return 0, ErrLineTooLong
			}
			if lw.debug >= 9 {
				log.Println("SER RX LINE:", string(line))
			}
			return copy(b, line), nil
		}

		rd := make([]byte, 512)
		c, err := lw.dev.Read(rd)
		if c > 0 {
			if lw.debug >= 9 {
				log.Println("SER RX RAW:", test.HexDump(rd[:c]))
			}
			lw.buf.Write(rd[:c])
		}
		if err != nil {
			// surface any complete line already buffered before the error
			if line, ok, lerr := lw.takeLine(); lerr == nil && ok && len(line) > 0 {
				if len(line) > len(b) {
					return 0, ErrLineTooLong
				}
				return copy(b, line), nil
			}
			return 0, err
		}
	}
}

// Write passes data straight through. Callers supply their own line
// terminators.
func (lw *LineWrapper) Write(b []byte) (int, error) {
	if lw.debug >= 9 {
		log.Println("SER TX RAW:", test.HexDump(b))
	}
	return lw.dev.Write(b)
}

// Close the wrapped device.
func (lw *LineWrapper) Close() error {
	return lw.dev.Close()
}
