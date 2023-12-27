package api

// this code based on https://github.com/unrolled/logger, but expanded
// to optionally dump the req/resp body

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

// HTTPLogger can be used to log http requests
type HTTPLogger struct {
	prefix string
	*log.Logger
}

// NewHTTPLogger returns a http logger
func NewHTTPLogger(prefix string) *HTTPLogger {
	return &HTTPLogger{
		prefix: prefix,
		Logger: log.New(os.Stdout, prefix, 0),
	}
}

// Handler wraps an HTTP handler and logs the request as necessary.
func (l *HTTPLogger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var rdr io.ReadCloser
		buf, err := io.ReadAll(r.Body)
		if err == nil {
			rdr = io.NopCloser(bytes.NewBuffer(buf))
			rdr2 := io.NopCloser(bytes.NewBuffer(buf))
			r.Body = rdr2
		}

		crw := newCustomResponseWriter(w)
		next.ServeHTTP(crw, r)

		addr := r.RemoteAddr
		if err == nil {
			rBuf := bytes.Buffer{}
			_, _ = rBuf.ReadFrom(rdr)
			l.Printf("(%s) \"%s %s\" %d -> %v -> %v", addr, r.Method, r.RequestURI,
				crw.status, rBuf.String(), crw.buf.String())
		} else {
			l.Printf("(%s) \"%s %s\" %d", addr, r.Method, r.RequestURI, crw.status)
		}

	})
}

type customResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
	buf    bytes.Buffer
}

func (c *customResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

func (c *customResponseWriter) Write(b []byte) (int, error) {
	size, err := c.ResponseWriter.Write(b)
	c.buf.Write(b)
	c.size += size
	return size, err
}

func (c *customResponseWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *customResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement the Hijacker interface")
}

func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	// When WriteHeader is not called, it's safe to assume the status will be 200.
	return &customResponseWriter{
		ResponseWriter: w,
		status:         200,
	}
}
