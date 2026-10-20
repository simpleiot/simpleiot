package client

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/dim13/cobs"
	"github.com/simpleiot/simpleiot/test"
)

func TestCobs(t *testing.T) {
	// we expect COBS encoding to work as detailed here:
	// https://en.wikipedia.org/wiki/Consistent_Overhead_Byte_Stuffing
	testCases := []struct {
		dec, enc []byte
	}{
		{[]byte{0x0}, []byte{0x1, 0x1, 0x0}},
		{[]byte{0x0, 0x0}, []byte{0x1, 0x1, 0x1, 0x0}},
		{[]byte{0x0, 0x11, 0x0}, []byte{0x1, 0x2, 0x11, 0x1, 0x0}},
		{[]byte{0x11, 0x22, 0x00, 0x33}, []byte{0x3, 0x11, 0x22, 0x2, 0x33, 0x00}},
		{[]byte{0x11, 0x22, 0x33, 0x44}, []byte{0x5, 0x11, 0x22, 0x33, 0x44, 0x00}},
		{[]byte{0x11, 0x00, 0x00, 0x00}, []byte{0x2, 0x11, 0x1, 0x1, 0x1, 0x00}},
	}

	for _, tc := range testCases {
		e := cobsEncode(tc.dec)

		fmt.Printf("Encoding %v -> %v\n", test.HexDump(tc.dec), test.HexDump(e))

		if !bytes.Equal(tc.enc, e) {
			t.Fatalf("enc failed for %v, got: %v, exp: %v",
				test.HexDump(tc.dec), test.HexDump(e), test.HexDump(tc.enc))
		}

		c, err := cobsDecodeInplace(e)

		if err != nil {
			t.Fatal("Error decoding: ", err)
		}

		e = e[:c]

		if !bytes.Equal(tc.dec, e) {
			t.Fatalf("Decode failed: %v -> %v",
				test.HexDump(tc.dec), test.HexDump(e))
		}
	}
}

func TestCobsLong(t *testing.T) {
	b := make([]byte, 300)
	for i := range b {
		b[i] = 5
	}

	e := cobsEncode(b)

	c, err := cobsDecodeInplace(e)
	if err != nil {
		t.Fatal("Error decoding: ", err)
	}

	e = e[:c]

	if !bytes.Equal(b, e) {
		t.Fatalf("Decode failed: %v", test.HexDump(e))
	}
}

func TestCobs254Run(t *testing.T) {
	// a run of 254 non-zero bytes fills a 0xff block, which has no
	// implied zero, so a following zero needs a block of its own
	run := make([]byte, 254)
	for i := range run {
		run[i] = byte(i + 1)
	}

	cat := func(s ...[]byte) []byte {
		return bytes.Join(s, nil)
	}

	testCases := []struct {
		name     string
		dec, enc []byte
	}{
		{"run then zero", cat(run, []byte{0x0, 0x11}),
			cat([]byte{0xff}, run, []byte{0x1, 0x2, 0x11, 0x0})},
		{"run at end", run,
			cat([]byte{0xff}, run, []byte{0x1, 0x0})},
		{"zero then run", cat([]byte{0x0}, run),
			cat([]byte{0x1, 0xff}, run, []byte{0x1, 0x0})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := cobsEncode(tc.dec)
			if !bytes.Equal(tc.enc, e) {
				t.Fatalf("enc failed, got: %v, exp: %v",
					test.HexDump(e), test.HexDump(tc.enc))
			}

			c, err := cobsDecodeInplace(e)
			if err != nil {
				t.Fatal("Error decoding: ", err)
			}

			if !bytes.Equal(tc.dec, e[:c]) {
				t.Fatalf("Decode failed: %v", test.HexDump(e[:c]))
			}
		})
	}

	// a 254 run at the end may also be sent without the final 0x01 block
	e := cat([]byte{0xff}, run, []byte{0x0})
	c, err := cobsDecodeInplace(e)
	if err != nil {
		t.Fatal("Error decoding: ", err)
	}
	if !bytes.Equal(run, e[:c]) {
		t.Fatalf("Decode of short form failed: %v", test.HexDump(e[:c]))
	}
}

func TestCobsRead(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	_, _ = a.Write(append([]byte{0}, cobsEncode(d)...))

	buf := make([]byte, 500)

	chBuf := make(chan struct{})

	go func() {
		c, err := cw.Read(buf)
		if err != nil {
			fmt.Println("Error reading cw: ", err)
		}
		buf = buf[0:c]
		chBuf <- struct{}{}
	}()

	select {
	case <-chBuf:
		// all is well
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for data")
	}

	if !reflect.DeepEqual(buf, d) {
		t.Fatal("Read data does not match")
	}
}

func TestCobsReadNoLeadingNull(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	_, _ = a.Write(cobsEncode(d))

	buf := make([]byte, 500)

	chBuf := make(chan struct{})

	go func() {
		c, err := cw.Read(buf)
		if err != nil {
			fmt.Println("Error reading cw: ", err)
		}
		buf = buf[0:c]
		chBuf <- struct{}{}
	}()

	select {
	case <-chBuf:
		// all is well
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for data")
	}

	if !reflect.DeepEqual(buf, d) {
		t.Fatal("Read data does not match")
	}
}

func TestCobsWrite(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	_, err := cw.Write(d)
	if err != nil {
		t.Fatal("Error write: ", err)
	}

	buf := make([]byte, 500)

	chBuf := make(chan struct{})

	go func() {
		c, err := a.Read(buf)
		if err != nil {
			fmt.Println("Error reading cw: ", err)
		}
		buf = buf[0:c]
		chBuf <- struct{}{}
	}()

	select {
	case <-chBuf:
		// all is well
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for data")
	}

	if buf[0] != 0 {
		t.Fatal("COBS encoded packet must start with 0")
	}

	if !reflect.DeepEqual(cobs.TrimNull(cobs.Decode(buf[1:len(buf)-1])), d) {
		t.Fatal("cw.Write, buf is not same")
	}
}

func TestCobsWrapperPartialPacket(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	de := cobsEncode(d)

	// write part of packet
	_, _ = a.Write(de[0:4])

	// start reader
	readData := make(chan []byte)
	errCh := make(chan error)

	go func() {
		buf := make([]byte, 500)
		c, err := cw.Read(buf)
		if err != nil {
			errCh <- err
		}
		buf = buf[0:c]
		readData <- buf
	}()

	// should time out as we don't have entire packet to decode yet
	select {
	case <-readData:
		t.Fatal("should not have read data yet")
	case err := <-errCh:
		t.Fatal("Read failed when it should have blocked: ", err)
	case <-time.After(time.Millisecond * 10):
		// all is well
	}

	// write the rest of the packet
	_, _ = a.Write(de[4:])

	// now look for packet
	select {
	case ret := <-readData:
		if !reflect.DeepEqual(ret, d) {
			t.Fatal("Read data does not match")
		}
	case err := <-errCh:
		t.Fatal("Read failed: ", err)
	case <-time.After(time.Millisecond * 10):
		t.Fatal("Timeout reading packet")
	}
}

func TestCobsMultipleLeadingNull(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}
	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	_, _ = a.Write(append([]byte{0, 0, 0, 0}, cobsEncode(d)...))

	buf := make([]byte, 500)

	chBuf := make(chan struct{})

	go func() {
		c, err := cw.Read(buf)
		if err != nil {
			fmt.Println("Error reading cw: ", err)
		}
		buf = buf[0:c]
		chBuf <- struct{}{}
	}()

	select {
	case <-chBuf:
		// all is well
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for data")
	}

	if !reflect.DeepEqual(buf, d) {
		t.Fatal("Read data does not match")
	}
}

func TestCobsPartialThenNew(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}
	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	de := append([]byte{0}, cobsEncode(d)...)

	// write partial packet
	_, _ = a.Write(de[0:4])

	// then start new packet
	_, _ = a.Write(de)

	buf := make([]byte, 500)
	c, err := cw.Read(buf)
	if err == nil {
		dump := buf[:c]
		t.Fatal("should have gotten an error reading partial packet, data: ", test.HexDump(dump))
	}

	c, err = cw.Read(buf)
	if err != nil {
		t.Fatal("got error reading full packet: ", err)
	}
	buf = buf[0:c]

	if !reflect.DeepEqual(buf, d) {
		t.Fatalf("Read data does not match, exp: %v, got: %v", test.HexDump(d), test.HexDump(buf))
	}
}

func TestCobsWriteTwoThenRead(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}
	a, b := test.NewIoSim()

	cw := NewCobsWrapper(b, 500)

	de := cobsEncode(d)

	// write two packets
	_, _ = a.Write(append(de, de...))

	for i := 2; i < 2; i++ {
		buf := make([]byte, 500)
		c, err := cw.Read(buf)
		if err != nil {
			t.Fatal("got error reading full packet: ", i, err)
		}
		buf = buf[0:c]

		if !reflect.DeepEqual(buf, d) {
			t.Fatal("Read data does not match, iter: ", i)
		}
	}
}
