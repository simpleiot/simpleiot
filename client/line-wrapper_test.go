package client

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// chunkReader feeds out data in caller-defined chunks so tests can control
// exactly how a line is split across Read calls.
type chunkReader struct {
	chunks [][]byte
	i      int
}

func (c *chunkReader) Read(b []byte) (int, error) {
	if c.i >= len(c.chunks) {
		return 0, io.EOF
	}
	n := copy(b, c.chunks[c.i])
	c.i++
	return n, nil
}

func (c *chunkReader) Write(b []byte) (int, error) { return len(b), nil }
func (c *chunkReader) Close() error                { return nil }

// readLines drains the wrapper until EOF and returns the lines it produced.
func readLines(t *testing.T, lw *LineWrapper, bufSize int) []string {
	t.Helper()
	var out []string
	for {
		buf := make([]byte, bufSize)
		c, err := lw.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, ErrLineTooLong) {
				t.Fatalf("unexpected read error: %v", err)
			}
			return out
		}
		out = append(out, string(buf[:c]))
	}
}

func TestLineWrapperTerminators(t *testing.T) {
	tests := []struct {
		name string
		in   string
		exp  []string
	}{
		{"lf", "one\ntwo\n", []string{"one", "two"}},
		{"crlf", "one\r\ntwo\r\n", []string{"one", "two"}},
		{"mixed", "one\r\ntwo\n", []string{"one", "two"}},
		{"blank lines skipped", "one\n\n\ntwo\n", []string{"one", "two"}},
		{"no trailing terminator drops partial", "one\ntwo", []string{"one"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lw := NewLineWrapper(&chunkReader{chunks: [][]byte{[]byte(tc.in)}}, 1024)
			got := readLines(t, lw, 1024)
			if len(got) != len(tc.exp) {
				t.Fatalf("got %d lines %q, expected %d %q", len(got), got, len(tc.exp), tc.exp)
			}
			for i := range got {
				if got[i] != tc.exp[i] {
					t.Errorf("line %d: got %q, expected %q", i, got[i], tc.exp[i])
				}
			}
		})
	}
}

func TestLineWrapperSplitAcrossReads(t *testing.T) {
	// a single line arriving in three pieces must assemble
	lw := NewLineWrapper(&chunkReader{chunks: [][]byte{
		[]byte("pt upti"),
		[]byte("me 0 INT 36"),
		[]byte("00\n"),
	}}, 1024)

	got := readLines(t, lw, 1024)
	if len(got) != 1 || got[0] != "pt uptime 0 INT 3600" {
		t.Fatalf("got %q, expected one assembled line", got)
	}
}

func TestLineWrapperMultipleLinesInOneRead(t *testing.T) {
	lw := NewLineWrapper(&chunkReader{chunks: [][]byte{
		[]byte("one\ntwo\nthree\n"),
	}}, 1024)

	got := readLines(t, lw, 1024)
	exp := []string{"one", "two", "three"}
	if len(got) != len(exp) {
		t.Fatalf("got %q, expected %q", got, exp)
	}
	for i := range exp {
		if got[i] != exp[i] {
			t.Errorf("line %d: got %q, expected %q", i, got[i], exp[i])
		}
	}
}

func TestLineWrapperStripsEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		exp  string
	}{
		{"color around text", "\x1b[1;32mhello\x1b[0m\n", "hello"},
		{"clear screen", "\x1b[2Jhello\n", "hello"},
		{"no escapes", "hello\n", "hello"},
		{"escape only", "\x1b[0m\x1b[1;32m\nkeep\n", "keep"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lw := NewLineWrapper(&chunkReader{chunks: [][]byte{[]byte(tc.in)}}, 1024)
			got := readLines(t, lw, 1024)
			if len(got) != 1 || got[0] != tc.exp {
				t.Fatalf("got %q, expected [%q]", got, tc.exp)
			}
		})
	}
}

func TestLineWrapperStripsPrompt(t *testing.T) {
	tests := []struct {
		name string
		in   string
		exp  []string
	}{
		{"prompt alone is blank", "uart:~$ \n", nil},
		{"prompt then output", "uart:~$ pt uptime 0 INT 5\n", []string{"pt uptime 0 INT 5"}},
		{"repeated prompt", "uart:~$ uart:~$ hello\n", []string{"hello"}},
		{"prompt mid line is kept", "hello uart:~$ there\n", []string{"hello uart:~$ there"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lw := NewLineWrapper(&chunkReader{chunks: [][]byte{[]byte(tc.in)}}, 1024)
			got := readLines(t, lw, 1024)
			if len(got) != len(tc.exp) {
				t.Fatalf("got %q, expected %q", got, tc.exp)
			}
			for i := range tc.exp {
				if got[i] != tc.exp[i] {
					t.Errorf("line %d: got %q, expected %q", i, got[i], tc.exp[i])
				}
			}
		})
	}
}

func TestLineWrapperOverlongLineResyncs(t *testing.T) {
	long := bytes.Repeat([]byte("x"), 200)
	lw := NewLineWrapper(&chunkReader{chunks: [][]byte{
		append(long, []byte("\ngood\n")...),
	}}, 64)

	// first read reports the overlong line
	buf := make([]byte, 1024)
	_, err := lw.Read(buf)
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("expected ErrLineTooLong, got %v", err)
	}

	// the following line must still read cleanly -- this is the resync
	c, err := lw.Read(buf)
	if err != nil {
		t.Fatalf("expected the next line to read cleanly, got %v", err)
	}
	if string(buf[:c]) != "good" {
		t.Errorf("got %q, expected \"good\" after resync", string(buf[:c]))
	}
}

func TestLineWrapperLineLongerThanCallerBuffer(t *testing.T) {
	lw := NewLineWrapper(&chunkReader{chunks: [][]byte{
		[]byte("0123456789abcdef\n"),
	}}, 1024)

	buf := make([]byte, 4)
	_, err := lw.Read(buf)
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("expected ErrLineTooLong for an undersized caller buffer, got %v", err)
	}
}

func TestLineWrapperWritePassesThrough(t *testing.T) {
	var sink bytes.Buffer
	lw := NewLineWrapper(&writeCapture{Buffer: &sink}, 1024)

	n, err := lw.Write([]byte("p uptime 0 INT 5\n"))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if n != 17 {
		t.Errorf("got %d bytes written, expected 17", n)
	}
	// no terminator is added or removed by the wrapper
	if sink.String() != "p uptime 0 INT 5\n" {
		t.Errorf("got %q, expected the bytes to pass through unchanged", sink.String())
	}
}

type writeCapture struct {
	*bytes.Buffer
}

func (w *writeCapture) Read(_ []byte) (int, error) { return 0, io.EOF }
func (w *writeCapture) Close() error               { return nil }
