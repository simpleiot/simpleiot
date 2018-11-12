package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/data"
)

// IndexHandler is used to serve the index page
type IndexHandler struct{}

func (h *IndexHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("indexHandler")
	f := frontend.Asset("/index.html")
	if f == nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		var reader = bytes.NewBuffer(f)
		io.Copy(rw, reader)
	}
}

// NewIndexHandler returns a new Index handler
func NewIndexHandler() http.Handler {
	return &IndexHandler{}
}

// App is a struct that implements http.Handler interface
type App struct {
	PublicHandler http.Handler
	IndexHandler  http.Handler
	V1ApiHandler  http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *App) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string

	fmt.Println("Path: ", req.URL.Path)

	if req.URL.Path == "/" {
		h.IndexHandler.ServeHTTP(res, req)
	} else {
		head, req.URL.Path = api.ShiftPath(req.URL.Path)
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
func NewAppHandler(state *data.State) http.Handler {
	return &App{
		PublicHandler: http.FileServer(frontend.FileSystem()),
		IndexHandler:  NewIndexHandler(),
		V1ApiHandler:  api.NewV1Handler(state),
	}
}

func httpServer(port string, state *data.State) error {
	address := fmt.Sprintf(":%s", port)
	log.Println("Starting http server")
	return http.ListenAndServe(address, NewAppHandler(state))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	state := data.State{}

	log.Println("Starting portal on port: ", port)
	err := httpServer(port, &state)
	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
