package api

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebsocketHandler handles websocket connections
type WebsocketHandler struct {
	clients map[*websocket.Conn]bool
	lock    *sync.RWMutex
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

	h.lock.Lock()
	h.clients[ws] = true
	h.lock.Unlock()

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
	h.lock.Lock()
	ws.Close()
	delete(h.clients, ws)
	h.lock.Unlock()
}

// NewWebsocketHandler returns a new websocket handler
func NewWebsocketHandler(wsTx chan []byte) http.Handler {
	clients := make(map[*websocket.Conn]bool)
	var lock sync.RWMutex
	go func() {
		for {
			select {
			case m := <-wsTx:
				for client := range clients {
					lock.RLock()
					err := client.WriteMessage(websocket.TextMessage, m)
					lock.RUnlock()
					if err != nil {
						log.Println("Error writing to websocket: ", err)
						lock.Lock()
						client.Close()
						delete(clients, client)
						lock.Unlock()
					}
				}
			}
		}
	}()
	return &WebsocketHandler{clients: clients, lock: &lock}
}
