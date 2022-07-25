package test

import (
	"testing"
	"time"
)

func TestFifo(t *testing.T) {
	a, err := NewFifoA("fifo")
	if err != nil {
		t.Fatal("Creating A side failed: ", err)
	}
	defer a.Close()

	b, err := NewFifoB("fifo")
	if err != nil {
		t.Fatal("Creating B side failed: ", err)
	}
	defer b.Close()

	testString := "hi there"

	_, err = a.Write([]byte(testString))
	if err != nil {
		t.Fatal("Error writing a: ", err)
	}

	buf := make([]byte, 500)

	c, err := b.Read(buf)
	if err != nil {
		t.Fatal("Error reading b: ", err)
	}

	buf = buf[0:c]

	if string(buf) != testString {
		t.Fatal("did not get test string back")
	}

	_, err = b.Write([]byte(testString))
	if err != nil {
		t.Fatal("Error writing b: ", err)
	}

	buf = make([]byte, 500)

	c, err = a.Read(buf)
	if err != nil {
		t.Fatal("Error reading b: ", err)
	}

	buf = buf[0:c]

	if string(buf) != testString {
		t.Fatal("did not get test string back")
	}

	// verfy fifo reads with no data block
	readReturned := make(chan struct{})
	go func() {
		a.Read(buf)
		close(readReturned)
	}()

	select {
	case <-readReturned:
		t.Error("Read should have never returned")
	case <-time.After(time.Millisecond * 10):
		// all is well
	}

}
