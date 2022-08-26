package api

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/koding/websocketproxy"
	"github.com/nats-io/nats.go"
)

// IndexHandler is used to serve the index page
type IndexHandler struct {
	getAsset func(string) []byte
}

func (h *IndexHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	f := h.getAsset("/index.html")
	if f == nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		var reader = bytes.NewBuffer(f)
		io.Copy(rw, reader)
	}
}

// NewIndexHandler returns a new Index handler
func NewIndexHandler(getAsset func(string) []byte) http.Handler {
	return &IndexHandler{getAsset: getAsset}
}

// App is a struct that implements http.Handler interface
type App struct {
	PublicHandler  http.Handler
	IndexHandler   http.Handler
	V1ApiHandler   http.Handler
	WebsocketProxy http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *App) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string

	switch req.URL.Path {
	case "/":
		headerUpgrade := req.Header["Upgrade"]
		if h.WebsocketProxy != nil && len(headerUpgrade) > 0 && headerUpgrade[0] == "websocket" {
			h.WebsocketProxy.ServeHTTP(res, req)
		} else {
			h.IndexHandler.ServeHTTP(res, req)
		}
	case "/orgs", "/users", "/devices", "/sign-in", "/groups", "/msg":
		h.IndexHandler.ServeHTTP(res, req)

	default:
		head, req.URL.Path = ShiftPath(req.URL.Path)
		switch head {
		case "public":
			h.PublicHandler.ServeHTTP(res, req)
		case "v1":
			h.V1ApiHandler.ServeHTTP(res, req)
		default:
			http.Error(res, "Not Found", http.StatusNotFound)
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
			log.Println("Error with WS url: ", err)
		} else {
			wsProxy = websocketproxy.NewProxy(u)
		}
	}

	return &App{
		PublicHandler:  http.FileServer(args.Filesystem),
		IndexHandler:   NewIndexHandler(args.GetAsset),
		V1ApiHandler:   v1,
		WebsocketProxy: wsProxy,
	}
}

// ServerArgs can be used to pass arguments to the server subsystem
type ServerArgs struct {
	Port       string
	GetAsset   func(string) []byte
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
	lnLock sync.Mutex
}

// NewServer ..
func NewServer(args ServerArgs) *Server {
	return &Server{args: args}
}

// Start the api server
func (s *Server) Start() error {
	log.Println("Starting http server, debug: ", s.args.Debug)
	log.Println("Starting portal on port: ", s.args.Port)
	address := fmt.Sprintf(":%s", s.args.Port)

	var err error
	s.lnLock.Lock()
	s.ln, err = net.Listen("tcp", address)
	s.lnLock.Unlock()
	if err != nil {
		return fmt.Errorf("Error starting api server: %v", err)
	}

	return http.Serve(s.ln, NewAppHandler(s.args))
}

// Stop HTTP API
func (s *Server) Stop(_ error) {
	// the following is required if Stop() is called very quickly after Start()
	for s.ln == nil {
		time.Sleep(10 * time.Millisecond)
	}
	s.lnLock.Lock()
	defer s.lnLock.Unlock()
	s.ln.Close()
}
