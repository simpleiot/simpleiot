package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebsocketHandler handles websocket connections
type WebsocketHandler struct {
	in chan []byte
}

func (h *WebsocketHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Websocket handler")

	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	// handle writing
	go func() {
		for {
			select {
			case m := <-h.in:
				err := ws.WriteMessage(websocket.TextMessage, m)
				if err != nil {
					log.Println("Error writing to websocket: ", err)
					return
				}
			}
		}
	}()

	// handle reading
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			log.Println("WS read error: ", err)
			break
		} else {
			log.Println("WS read: ", msgType, msg)
		}
	}

	log.Println("closing websocket")
	ws.Close()
}

// NewWebsocketHandler returns a new websocket handler
func NewWebsocketHandler(c chan []byte) http.Handler {
	return &WebsocketHandler{in: c}
}
