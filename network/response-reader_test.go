package network

import (
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

func TestResponseReader(t *testing.T) {
	source := &dataSource{}
	reader := NewResponseReader(source, time.Second, time.Millisecond*10)

	start := time.Now()
	data := make([]byte, 100)
	count, err := reader.Read(data)

	dur := time.Since(start)

	if err != nil {
		t.Error("read failed: ", err)
	}

	if dur < 100*time.Millisecond || dur > 200*time.Millisecond {
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
	reader := NewResponseReader(source, time.Second, time.Millisecond*10)

	start := time.Now()
	data := make([]byte, 100)
	count, err := reader.Read(data)

	dur := time.Since(start)

	if err != ErrorTimeout {
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

func TestResponseReaderWrite(t *testing.T) {
	source := &dataSourceWrite{}
	readWriter := NewResponseReadWriter(source, time.Second, time.Millisecond*10)

	writeData := []byte{1, 2}
	readWriter.Write(writeData)

	start := time.Now()
	data := make([]byte, 100)
	count, err := readWriter.Read(data)

	dur := time.Since(start)

	if err != ErrorTimeout {
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
