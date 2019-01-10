package isapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/farmation/assets/isfrontend"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// IndexHandler is used to serve the index page
type IndexHandler struct{}

func (h *IndexHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("indexHandler")
	f := isfrontend.Asset("/index.html")
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
	PublicHandler    http.Handler
	IndexHandler     http.Handler
	WebsocketHandler http.Handler
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
		case "ws":
			h.WebsocketHandler.ServeHTTP(res, req)
		default:
			http.Error(res, "Not Found", http.StatusNotFound)
		}
	}
}

// NewAppHandler returns a new application (root) http handler
func NewAppHandler(wsTxChan chan []byte) http.Handler {
	return &App{
		PublicHandler:    http.FileServer(isfrontend.FileSystem()),
		IndexHandler:     NewIndexHandler(),
		WebsocketHandler: api.NewWebsocketHandler(wsTxChan),
	}
}

// Server starts a API server instance
func Server(in, out chan interface{}) {
	log.Println("Starting http server")

	wsTxChan := make(chan []byte, 100)

	port := os.Getenv("IS_PORT")
	if port == "" {
		port = "8090"
	}

	log.Println("Starting IS app on port: ", port)
	address := fmt.Sprintf(":%s", port)
	go http.ListenAndServe(address, NewAppHandler(wsTxChan))

	for {
		if len(wsTxChan) >= 99 {
			log.Println("Warning wsTxChan full, dropping data")
			<-wsTxChan
		}
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.LcdPixel, isdata.LcdBlt, isdata.LcdBltSolid:
				d, err := json.Marshal(m)
				if err != nil {
					log.Println("Error encoding LcdPixel: ", err)
				} else {
					wsTxChan <- d
				}
			default:
				log.Printf("web: unhandled message of type %T: %+v\n", m, m)
			}
		}
	}
}
