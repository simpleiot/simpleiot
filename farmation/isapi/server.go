package isapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/data"
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
func NewAppHandler(wsTxChan <-chan []byte, wsRxChan chan<- []byte, newConn chan<- bool) http.Handler {
	return &App{
		PublicHandler:    http.FileServer(isfrontend.FileSystem()),
		IndexHandler:     NewIndexHandler(),
		WebsocketHandler: api.NewWebsocketHandler(wsTxChan, wsRxChan, newConn),
	}
}

// Server starts a API server instance
func Server(in, out chan interface{}) {
	log.Println("Starting http server")

	wsTxChan := make(chan []byte, 100)
	wsRxChan := make(chan []byte, 100)
	newConn := make(chan bool)

	frameBuffer := isdata.NewLcdBlt(0, 0, 128, 64, true)

	port := os.Getenv("IS_PORT")
	if port == "" {
		port = "8090"
	}

	log.Println("Starting IS app on port: ", port)
	address := fmt.Sprintf(":%s", port)
	go http.ListenAndServe(address, NewAppHandler(wsTxChan, wsRxChan, newConn))

	for {
		if len(wsTxChan) >= 99 {
			log.Println("Warning wsTxChan full, dropping data")
			<-wsTxChan
		}
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.LcdPixel, isdata.LcdBlt, isdata.LcdBltSolid, isdata.State:
				d, err := json.Marshal(m)
				if err != nil {
					log.Println("Error encoding Lcd data: ", err)
				} else {
					wsTxChan <- d
				}
				switch m := m.(type) {
				case isdata.LcdPixel:
					frameBuffer.Set(m.X, m.Y, m.V)
				case isdata.LcdBlt:
					frameBuffer.Update(m)
				case isdata.LcdBltSolid:
					frameBuffer.UpdateSolid(m)
				}
			default:
				log.Printf("web: unhandled message of type %T: %+v\n", m, m)
			}
		case <-newConn:
			log.Println("New web client connection, sending current FB")
			d, err := json.Marshal(frameBuffer)
			if err != nil {
				log.Println("Error encoding Framebuffer: ", err)
			} else {
				// give the client time to get set up
				time.Sleep(10 * time.Millisecond)
				wsTxChan <- d
			}
		case m := <-wsRxChan:
			s := data.Sample{}
			if err := json.Unmarshal(m, &s); err != nil {
				log.Println("Error parsing websocket data: ", err)
			} else {
				out <- s
			}
		}
	}
}
