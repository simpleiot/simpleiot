package respreader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

type dataSource struct {
	count int
}

func (ds *dataSource) Read(data []byte) (int, error) {
	ds.count++
	switch ds.count {
	case 1:
		time.Sleep(100 * time.Millisecond)
		data[0] = 0
		return 1, nil
	case 2, 3, 4, 5, 6, 7, 8, 9, 10:
		time.Sleep(5 * time.Millisecond)
		data[0] = 1
		return 1, nil
	default:
		time.Sleep(1000 * time.Hour)

	}

	return 0, nil
}

// the below test shows using a fifo with respreader
func TestWithFifo(t *testing.T) {
	fifo := "rrfifoo"
	os.Remove(fifo)
	err := exec.Command("mkfifo", fifo).Run()
	if err != nil {
		t.Error("mkfifo returned: ", err)
	}

	testString := "hi there"
	done := make(chan struct{})

	// need to open RDWR or open will block until fifo is opened for writing
	fread, err := os.OpenFile(fifo, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal("Error opening fifo: ", err)
	}
	reader := NewReadWriteCloser(fread, 2*time.Second, 50*time.Millisecond)

	// read function
	go func() {
		for {
			rdata := make([]byte, 128)
			c, err := reader.Read(rdata)
			if err == io.EOF {
				fmt.Println("Reader returned EOF, exiting read routine")
				break
			}
			if err != nil {
				t.Error("Read error: ", err)
			}
			if c > 0 {
				rdata = rdata[:c]
				if string(rdata) == testString {
					close(done)
					break
				}
			}
		}
	}()

	fwrite, err := os.OpenFile(fifo, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal("Error opening file for writing: ", err)
	}

	_, err = fwrite.Write([]byte(testString))

	if err != nil {
		t.Error("Write error: ", err)
	}

	select {
	case <-time.After(time.Second):
		t.Error("Timeout waiting for read to complete")
	case <-done:
		// all is well
	}

	fwrite.Close()
	fread.Close()

	err = os.Remove(fifo)
	if err != nil {
		t.Error("Error removing fifo")
	}
}

// the below test illustrates out the goroutine in the reader will close if you close
// the underlying file descriptor.
func TestReadCloser(t *testing.T) {
	fifo := "rrfifoo"
	os.Remove(fifo)
	err := exec.Command("mkfifo", fifo).Run()
	if err != nil {
		t.Error("mkfifo returned: ", err)
	}

	done := make(chan struct{})

	// need to open RDWR or open will block until fifo is opened for writing
	fread, err := os.OpenFile(fifo, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal("Error opening fifo: ", err)
	}
	reader := NewReadWriteCloser(fread, 50*time.Millisecond, 20*time.Millisecond)

	// read function
	go func() {
		for {
			rdata := make([]byte, 128)
			_, err := reader.Read(rdata)
			if err == io.EOF {
				close(done)
				break
			}
			if err != nil {
				t.Error("Read error: ", err)
			}
		}
	}()

	fread.Close()

	select {
	case <-time.After(time.Second):
		t.Error("Timeout waiting for read to complete")
	case <-done:
		// all is well
	}
}

/* the following test is for documentation only

// the below test illustrates out the goroutine in the reader will close if you close
// the underlying serial port descriptor.
func TestResponseReaderSerialPortClose(t *testing.T) {
	fmt.Println("=============================")
	fmt.Println("Testing serial port close")
	readCnt := make(chan int)

	serialPort := "/dev/ttyUSB1"

	options := serial.OpenOptions{
		PortName:              serialPort,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}

	go func(readCnt chan int) {
		fread, err := serial.Open(options)
		if err != nil {
			t.Error("Error opening serial port: ", err)
		}

		reader := NewReadWriteCloser(fread, 2*time.Second, 50*time.Millisecond)

		fmt.Println("reader created")

		fmt.Println("read thread")
		closed := false
		cnt := 0
		for {
			rdata := make([]byte, 128)
			fmt.Println("calling read")
			c, err := reader.Read(rdata)
			if err == io.EOF {
				fmt.Println("Reader returned EOF, exiting read routine")
				break
			}
			if err != nil {
				//t.Error("Read error: ", err)
				fmt.Println("Read error: ", err)
			}
			cnt = c
			fmt.Println("read count: ", c)
			if !closed && c > 0 {
				go func() {
					time.Sleep(20 * time.Millisecond)
					fmt.Println("closing read file")
					reader.Close()
					closed = true
				}()
			}
		}

		readCnt <- cnt
	}(readCnt)

	time.Sleep(500 * time.Millisecond)

	options.PortName = serialPort

	fwrite, err := serial.Open(options)
	if err != nil {
		t.Error("Error opening file for writing: ", err)
	}

	c, err := fwrite.Write([]byte("Hi there"))

	if err != nil {
		t.Error("Write error: ", err)
	}

	fmt.Printf("Wrote %v bytes\n", c)

	readCount := <-readCnt

	if readCount != 8 {
		t.Errorf("only read %v chars, expected 8", readCount)
	}

	fmt.Println("test all done")
}
*/

func TestReader(t *testing.T) {
	source := &dataSource{}
	reader := NewReader(source, time.Second, time.Millisecond*50)

	start := time.Now()
	data := make([]byte, 100)
	count, err := reader.Read(data)

	dur := time.Since(start)

	if err != nil {
		t.Error("read failed: ", err)
	}

	if dur < 100*time.Millisecond || dur > 400*time.Millisecond {
		t.Error("expected dur to be around 150ms: ", dur)
	}

	if count != 10 {
		t.Error("expected count to be 10: ", count)
	}

	data = data[0:count]

	expData := []byte{0, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	if !reflect.DeepEqual(data, expData) {
		t.Error("expected: ", expData)
		t.Error("got     : ", data)
	}
}

type dataSourceTimeout struct {
}

func (ds *dataSourceTimeout) Read(data []byte) (int, error) {
	time.Sleep(1000 * time.Hour)

	return 0, nil
}

func TestResponseReaderTimeout(t *testing.T) {
	source := &dataSourceTimeout{}
	reader := NewReader(source, time.Second, time.Millisecond*10)

	start := time.Now()
	data := make([]byte, 100)
	count, err := reader.Read(data)

	dur := time.Since(start)

	if err != io.EOF {
		t.Error("expected timeout error, got: ", err)
	}

	if dur < 900*time.Millisecond || dur > 1100*time.Millisecond {
		t.Error("expected dur to be around 1s: ", dur)
	}

	if count != 0 {
		t.Error("expected count to be 0: ", count)
	}

	data = data[0:count]

	expData := []byte{}

	if !reflect.DeepEqual(data, expData) {
		t.Error("expected: ", expData)
		t.Error("got     : ", data)
	}
}

type dataSourceWrite struct {
	count     int
	writeData []byte
}

func (ds *dataSourceWrite) Read(data []byte) (int, error) {
	ds.count++
	switch ds.count {
	case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10:
		time.Sleep(5 * time.Millisecond)
		data[0] = 1
		return 1, nil
	default:
		time.Sleep(1000 * time.Hour)

	}

	return 0, nil
}

func (ds *dataSourceWrite) Write(data []byte) (int, error) {
	ds.writeData = data
	return len(data), nil
}

func TestReadWriter(t *testing.T) {
	source := &dataSourceWrite{}
	readWriter := NewReadWriter(source, time.Second, time.Millisecond*10)

	writeData := []byte{1, 2}
	readWriter.Write(writeData)

	start := time.Now()
	data := make([]byte, 100)
	count, err := readWriter.Read(data)

	dur := time.Since(start)

	if err != io.EOF {
		t.Error("expected timeout error: ", err)
	}

	if dur < 900*time.Millisecond || dur > 1100*time.Millisecond {
		t.Error("expected dur to be around 1s: ", dur)
	}

	if count != 0 {
		t.Error("expected count to be 0: ", count)
	}

	data = data[0:count]

	expData := []byte{}

	if !reflect.DeepEqual(data, expData) {
		t.Error("expected: ", expData)
		t.Error("got     : ", data)
	}

	if !reflect.DeepEqual(writeData, source.writeData) {
		t.Error("write data is not correct")
	}
}
