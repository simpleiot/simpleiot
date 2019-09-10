package api

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/simpleiot/simpleiot/db"
)

// IndexHandler is used to serve the index page
type IndexHandler struct {
	getAsset func(string) []byte
}

func (h *IndexHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("indexHandler")
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
	PublicHandler http.Handler
	IndexHandler  http.Handler
	V1ApiHandler  http.Handler
	Debug         bool
}

// Top level handler for http requests in the coap-server process
func (h *App) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string

	if h.Debug {
		fmt.Printf("HTTP %v: %v\n", req.Method, req.URL.Path)
	}

	if req.URL.Path == "/" {
		h.IndexHandler.ServeHTTP(res, req)
	} else {
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
func NewAppHandler(db *db.Db, getAsset func(string) []byte,
	filesystem http.FileSystem, debug bool) http.Handler {
	return &App{
		PublicHandler: http.FileServer(filesystem),
		IndexHandler:  NewIndexHandler(getAsset),
		V1ApiHandler:  NewV1Handler(db),
		Debug:         debug,
	}
}

// Server starts a API server instance
func Server(getAsset func(string) []byte, filesystem http.FileSystem, debug bool) error {
	log.Println("Starting http server, debug: ", debug)

	port := os.Getenv("SIOT_PORT")
	if port == "" {
		port = "8080"
	}

	dataDir := os.Getenv("SIOT_DATA")
	if dataDir == "" {
		dataDir = "./"
	}

	db, err := db.NewDb(dataDir)
	if err != nil {
		log.Println("Error opening db: ", err)
		os.Exit(-1)
	}

	log.Println("Starting portal on port: ", port)
	address := fmt.Sprintf(":%s", port)
	return http.ListenAndServe(address, NewAppHandler(db, getAsset, filesystem, debug))
}
