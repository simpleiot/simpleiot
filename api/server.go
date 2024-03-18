package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/koding/websocketproxy"
	"github.com/nats-io/nats.go"
)

// App is a struct that implements http.Handler interface
type App struct {
	PublicHandler  http.Handler
	V1ApiHandler   http.Handler
	WebsocketProxy http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *App) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/":
		headerUpgrade := req.Header["Upgrade"]
		if h.WebsocketProxy != nil && len(headerUpgrade) > 0 && headerUpgrade[0] == "websocket" {
			h.WebsocketProxy.ServeHTTP(res, req)
		} else {
			h.PublicHandler.ServeHTTP(res, req)
		}
	case "/sign-in":
		req.URL.Path = "/"
		h.PublicHandler.ServeHTTP(res, req)

	default:
		head, path := ShiftPath(req.URL.Path)
		switch head {
		case "v1":
			req.URL.Path = path
			h.V1ApiHandler.ServeHTTP(res, req)
		default:
			h.PublicHandler.ServeHTTP(res, req)
		}
	}
}

// NewAppHandler returns a new application (root) http handler
func NewAppHandler(args ServerArgs) http.Handler {
	v1 := NewV1Handler(args)
	if args.Debug {
		//args.Debug = false
		v1 = NewHTTPLogger("v1").Handler(v1)
	}

	var wsProxy http.Handler

	if args.NatsWSPort > 0 {
		uS := fmt.Sprintf("ws://localhost:%v", args.NatsWSPort)
		u, err := url.Parse(uS)
		if err != nil {
			log.Println("Error with WebSocket URL:", err)
		} else {
			wsProxy = websocketproxy.NewProxy(u)
		}
	}

	return &App{
		PublicHandler:  http.FileServer(args.Filesystem),
		V1ApiHandler:   v1,
		WebsocketProxy: wsProxy,
	}
}

// ServerArgs can be used to pass arguments to the server subsystem
type ServerArgs struct {
	Port       string
	Filesystem http.FileSystem
	Debug      bool
	JwtAuth    Authorizer
	AuthToken  string
	NatsWSPort int
	Nc         *nats.Conn
}

// Server represents the HTTP API server
type Server struct {
	args   ServerArgs
	ln     net.Listener
	chStop chan struct{}
}

// NewServer ..
func NewServer(args ServerArgs) *Server {
	return &Server{
		args:   args,
		chStop: make(chan struct{}),
	}
}

// Start the api server
func (s *Server) Start() error {
	log.Println("Starting http server, debug:", s.args.Debug)
	log.Println("Starting portal on port:", s.args.Port)
	address := fmt.Sprintf(":%s", s.args.Port)

	var err error
	s.ln, err = net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("Error starting api server: %v", err)
	}

	chError := make(chan error)

	go func() {
		chError <- http.Serve(s.ln, NewAppHandler(s.args))
	}()

	select {
	case <-s.chStop:
		s.ln.Close()
		err = <-chError
	case err = <-chError:
	}

	return err
}

// Stop HTTP API
func (s *Server) Stop(_ error) {
	close(s.chStop)
}
