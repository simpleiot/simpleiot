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
func NewAppHandler() http.Handler {
	c := make(chan []byte)
	go func() {
		x := 0
		y := 0
		v := true
		for {
			time.Sleep(time.Microsecond * 20)
			setPix := isdata.LcdSetPixel{X: x, Y: y, V: v}
			setPixJSON, err := json.Marshal(setPix)
			if err != nil {
				log.Println("Error encoding setPix: ", err)
				continue
			}

			c <- setPixJSON
			x++
			if x >= 128 {
				x = 0
				y++
				if y >= 64 {
					y = 0
					v = !v
				}
			}

		}
	}()

	return &App{
		PublicHandler:    http.FileServer(isfrontend.FileSystem()),
		IndexHandler:     NewIndexHandler(),
		WebsocketHandler: api.NewWebsocketHandler(c),
	}
}

// Server starts a API server instance
func Server() error {
	log.Println("Starting http server")

	port := os.Getenv("IS_PORT")
	if port == "" {
		port = "8090"
	}

	log.Println("Starting IS app on port: ", port)
	address := fmt.Sprintf(":%s", port)
	return http.ListenAndServe(address, NewAppHandler())
}
