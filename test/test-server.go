package test

import (
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/server"
)

// Server is used to set up test servers for unit tests that run
// all in memory.
type Server struct {
	server server.Server
}

// Stop a SIOT server
func (s *Server) Stop() {
	s.server.Stop(nil)
}

// StartServer is used to spin up a test siot store for testing
// we run everything on out of the way ports so we should not
// conflict with other running instances
func StartServer() (*Server, *nats.Conn, error) {

	var nc *nats.Conn

	s := &Server{}

	return s, nc, nil
}
