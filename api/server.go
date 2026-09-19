package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

// maxBodyBytes bounds the body of an API request. Node and point payloads
// are small; the limit keeps a client from holding memory with one request.
const maxBodyBytes = 4 << 20

// App is a struct that implements http.Handler interface
type App struct {
	PublicHandler  http.Handler
	V1ApiHandler   http.Handler
	WebsocketProxy http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *App) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	setSecurityHeaders(res.Header())

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
			req.Body = http.MaxBytesReader(res, req.Body, maxBodyBytes)
			h.V1ApiHandler.ServeHTTP(res, req)
		case "fonts":
			// the font files are named for their face and never change
			// in place; embedded files carry no modification time, so
			// without this the browser fetches them on every use
			res.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			h.PublicHandler.ServeHTTP(res, req)
		default:
			h.PublicHandler.ServeHTTP(res, req)
		}
	}
}

// importMapHash is the SHA-256 of the import map in the UI's index.html,
// the one inline script the page has. The content security policy allows
// it by hash; a test checks the hash against the embedded file.
const importMapHash = "sha256-gPiKIrU1rMetb7cOogG/Un/8wJwuzqdVF4wUyJQ4Ty0="

// setSecurityHeaders puts the response headers every reply carries. The
// content security policy allows what the UI is built from: its own
// scripts plus the import map, styles (the Elm UI sets styles inline) and
// fonts, and a WebSocket back to the server; and refuses to be framed.
func setSecurityHeaders(h http.Header) {
	h.Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' '"+importMapHash+"'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"font-src 'self'; "+
			"img-src 'self' data:; "+
			"connect-src 'self' ws: wss:; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'; "+
			"form-action 'self'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "same-origin")
}

// NewAppHandler returns a new application (root) http handler
func NewAppHandler(args ServerArgs) http.Handler {
	v1 := NewV1Handler(args)
	if args.Debug {
		v1 = NewHTTPLogger("v1").Handler(v1)
	}

	var wsProxy http.Handler

	if args.NatsWSPort > 0 {
		var err error
		wsProxy, err = newWebsocketProxy(args.NatsWSPort, args.NatsTLSCert,
			args.DeviceAuthRequired)
		if err != nil {
			log.Println("Error setting up WebSocket proxy:", err)
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
	// Users authenticates and scopes signed-in users on the node API.
	Users      UserAuthority
	AuthToken  string
	NatsWSPort int
	// NatsTLSCert is the NATS server's certificate file when it serves
	// TLS; the WebSocket proxy pins it.
	NatsTLSCert string
	Nc          *nats.Conn
	// DeviceAuth resolves device tokens on the node API; nil accepts none.
	DeviceAuth DeviceAuthorizer
	// DeviceAuthRequired limits the shared token to loopback, on the API
	// routes and through the WebSocket proxy.
	DeviceAuthRequired bool
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
		return fmt.Errorf("error starting api server: %v", err)
	}

	// A connection that sends nothing, or sends slowly, is closed rather
	// than held open. A WebSocket clears the read deadline when it is
	// hijacked, so the proxy is not affected.
	srv := &http.Server{
		Handler:           NewAppHandler(s.args),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}

	chError := make(chan error)

	go func() {
		chError <- srv.Serve(s.ln)
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
