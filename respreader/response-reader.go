package respreader

import (
	"errors"
	"io"
	"time"
)

// ReadWriteCloser is a convenience type that implements io.ReadWriteCloser.
// Write calls flush reader before writing the prompt.
type ReadWriteCloser struct {
	closer io.Closer
	writer io.Writer
	reader *Reader
}

// NewReadWriteCloser creates a new response reader
//
// timeout is used to specify an
// overall timeout. If this timeout is encountered, io.EOF is returned.
//
// chunkTimeout is used to specify the max timeout between chunks of data once
// the response is started. If a delay of chunkTimeout is encountered, the response
// is considered finished and the Read returns.
func NewReadWriteCloser(iorw io.ReadWriteCloser, timeout time.Duration, chunkTimeout time.Duration) *ReadWriteCloser {
	return &ReadWriteCloser{
		closer: iorw,
		writer: iorw,
		reader: NewReader(iorw, timeout, chunkTimeout),
	}
}

// Read response using chunkTimeout and timeout
func (rwc *ReadWriteCloser) Read(buffer []byte) (int, error) {
	return rwc.reader.Read(buffer)
}

// Write flushes all data from reader, and then passes through write call.
func (rwc *ReadWriteCloser) Write(buffer []byte) (int, error) {
	n, err := rwc.reader.Flush()
	if err != nil {
		return n, err
	}

	return rwc.writer.Write(buffer)
}

// SetTimeout can be used to update the reader timeout
func (rwc *ReadWriteCloser) SetTimeout(timeout, chunkTimeout time.Duration) {
	rwc.reader.SetTimeout(timeout, chunkTimeout)
}

// Close is a passthrough call.
func (rwc *ReadWriteCloser) Close() error {
	rwc.reader.closed = true
	return rwc.closer.Close()
}

// ReadCloser is a convenience type that implements io.ReadWriter. Write
// calls flush reader before writing the prompt.
type ReadCloser struct {
	closer io.Closer
	reader *Reader
}

// NewReadCloser creates a new response reader
//
// timeout is used to specify an
// overall timeout. If this timeout is encountered, io.EOF is returned.
//
// chunkTimeout is used to specify the max timeout between chunks of data once
// the response is started. If a delay of chunkTimeout is encountered, the response
// is considered finished and the Read returns.
func NewReadCloser(iorw io.ReadCloser, timeout time.Duration, chunkTimeout time.Duration) *ReadCloser {
	return &ReadCloser{
		closer: iorw,
		reader: NewReader(iorw, timeout, chunkTimeout),
	}
}

// Read response using chunkTimeout and timeout
func (rc *ReadCloser) Read(buffer []byte) (int, error) {
	return rc.reader.Read(buffer)
}

// Close is a passthrough call.
func (rc *ReadCloser) Close() error {
	rc.reader.closed = true
	return rc.closer.Close()
}

// SetTimeout can be used to update the reader timeout
func (rc *ReadCloser) SetTimeout(timeout, chunkTimeout time.Duration) {
	rc.reader.SetTimeout(timeout, chunkTimeout)
}

// ReadWriter is a convenience type that implements io.ReadWriter. Write
// calls flush reader before writing the prompt.
type ReadWriter struct {
	writer io.Writer
	reader *Reader
}

// NewReadWriter creates a new response reader
func NewReadWriter(iorw io.ReadWriter, timeout time.Duration, chunkTimeout time.Duration) *ReadWriter {
	return &ReadWriter{
		writer: iorw,
		reader: NewReader(iorw, timeout, chunkTimeout),
	}
}

// Read response
func (rw *ReadWriter) Read(buffer []byte) (int, error) {
	return rw.reader.Read(buffer)
}

// Write flushes all data from reader, and then passes through write call.
func (rw *ReadWriter) Write(buffer []byte) (int, error) {
	n, err := rw.reader.Flush()
	if err != nil {
		return n, err
	}

	return rw.writer.Write(buffer)
}

// SetTimeout can be used to update the reader timeout
func (rw *ReadWriter) SetTimeout(timeout, chunkTimeout time.Duration) {
	rw.reader.SetTimeout(timeout, chunkTimeout)
}

// Reader is used for prompt/response communication protocols where a prompt
// is sent, and some time later a response is received. Typically, the target takes
// some amount to formulate the response, and then streams it out. There are two delays:
// an overall timeout, and then an inter character timeout that is activated once the
// first byte is received. The thought is that once you received the 1st byte, all the
// data should stream out continuously and a short timeout can be used to determine the
// end of the packet.
type Reader struct {
	reader       io.Reader
	timeout      time.Duration
	chunkTimeout time.Duration
	size         int
	dataChan     chan []byte
	closed       bool
}

// NewReader creates a new response reader.
//
// timeout is used to specify an
// overall timeout. If this timeout is encountered, io.EOF is returned.
//
// chunkTimeout is used to specify the max timeout between chunks of data once
// the response is started. If a delay of chunkTimeout is encountered, the response
// is considered finished and the Read returns.
func NewReader(reader io.Reader, timeout time.Duration, chunkTimeout time.Duration) *Reader {
	r := Reader{
		reader:       reader,
		timeout:      timeout,
		chunkTimeout: chunkTimeout,
		size:         128,
		dataChan:     make(chan []byte),
	}
	// we have to start a reader goroutine here that lives for the life
	// of the reader because there is no
	// way to stop a blocked goroutine
	go r.readInput()
	return &r
}

// Read response
func (r *Reader) Read(buffer []byte) (int, error) {
	if len(buffer) <= 0 {
		return 0, errors.New("must supply non-zero length buffer")
	}

	timeout := time.NewTimer(r.timeout)
	count := 0

	for {
		select {
		case newData, ok := <-r.dataChan:
			// copy data from chan buffer to Read() buf
			for i := 0; count < len(buffer) && i < len(newData); i++ {
				buffer[count] = newData[i]
				count++
			}

			if !ok {
				return count, io.EOF
			}

			timeout.Reset(r.chunkTimeout)

		case <-timeout.C:
			if count > 0 {
				return count, nil
			}

			return count, io.EOF

		}
	}
}

// Flush is used to flush any input data
func (r *Reader) Flush() (int, error) {
	timeout := time.NewTimer(r.chunkTimeout)
	count := 0

	for {
		select {
		case newData, ok := <-r.dataChan:
			count += len(newData)
			if !ok {
				return count, io.EOF
			}

			timeout.Reset(r.chunkTimeout)

		case <-timeout.C:
			return count, nil
		}
	}
}

// SetTimeout can be used to update the reader timeout
func (r *Reader) SetTimeout(timeout, chunkTimeout time.Duration) {
	r.timeout = timeout
	r.chunkTimeout = chunkTimeout
}

// readInput is used by a goroutine to read data from the underlying io.Reader
func (r *Reader) readInput() {
	for {
		tmp := make([]byte, r.size)
		if r.closed {
			break
		}
		length, _ := r.reader.Read(tmp)
		if length > 0 {
			tmp = tmp[0:length]
			r.dataChan <- tmp
		}
	}
	close(r.dataChan)
}
