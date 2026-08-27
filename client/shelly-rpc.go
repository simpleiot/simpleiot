package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Simple IoT identifies itself as the source of every RPC request. A Gen2
// device will not send notifications over a WebSocket until it has seen a
// request frame carrying a src, so this string is what starts the flow of
// status updates as well as what the device addresses replies to.
const shellyRPCSrc = "siot"

const (
	// shellyWSReconnect is how long to wait before dialing a device again after
	// the WebSocket drops.
	shellyWSReconnect = time.Minute
	// shellyWSPing is how often to ping a connected device. A device that stops
	// answering is treated as offline without waiting for a status update that
	// may never come.
	shellyWSPing = time.Second * 30
	// shellyWSTimeout is how long to wait for any message, including a pong,
	// before giving up on the connection.
	shellyWSTimeout = shellyWSPing * 3
)

// shellyRPCRequest is a JSON-RPC request frame. The same frame works over HTTP
// and over the WebSocket.
type shellyRPCRequest struct {
	ID     int         `json:"id"`
	Src    string      `json:"src"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// shellyRPCError is the error object a device returns for a failed call.
type shellyRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *shellyRPCError) Error() string {
	return fmt.Sprintf("shelly rpc error %v: %v", e.Code, e.Message)
}

// shellyRPCFrame is a frame received from a device. A response carries Result
// or Error; a notification carries Method and Params and no ID.
type shellyRPCFrame struct {
	ID     int             `json:"id"`
	Src    string          `json:"src"`
	Dst    string          `json:"dst"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *shellyRPCError `json:"error"`
}

// rpc makes a Gen2 RPC call over HTTP and decodes the result into result when
// one is supplied.
func (sio *ShellyIo) rpc(method string, params interface{}, result interface{}) error {
	body, err := json.Marshal(shellyRPCRequest{
		ID: 1, Src: shellyRPCSrc, Method: method, Params: params,
	})
	if err != nil {
		return err
	}

	res, err := httpClient.Post("http://"+sio.IP+"/rpc", "application/json",
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("shelly %v returned status %v", method, res.StatusCode)
	}

	var frame shellyRPCFrame
	if err := json.NewDecoder(res.Body).Decode(&frame); err != nil {
		return err
	}
	if frame.Error != nil {
		return frame.Error
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(frame.Result, result)
}

// shellyStatusUpdate is what the WebSocket listener delivers to the client. A
// full update replaces the cached status; a partial one carries only the
// components that changed.
type shellyStatusUpdate struct {
	connected bool
	full      bool
	status    map[string]json.RawMessage
	err       error
}

// shellyWatch keeps a WebSocket open to a Gen2 or later device and delivers the
// status the device pushes.
//
// The device pushes NotifyStatus with the fields that changed and
// NotifyFullStatus after it restarts, so status arrives when it changes rather
// than on a timer. Only one request goes out over the socket, the
// Shelly.GetStatus that both fetches the starting status and, by carrying a
// src, tells the device to start notifying. Everything else Simple IoT asks of
// a device goes over HTTP, which keeps this side of the connection read-only.
//
// shellyWatch blocks until stop is closed, redialing on a timer whenever the
// connection drops.
type shellyWatcher struct {
	ip      string
	updates chan shellyStatusUpdate
	stop    chan struct{}

	lock sync.Mutex
	conn *websocket.Conn
}

func newShellyWatcher(ip string) *shellyWatcher {
	return &shellyWatcher{
		ip:      ip,
		updates: make(chan shellyStatusUpdate, 8),
		stop:    make(chan struct{}),
	}
}

// Start runs the watcher until Stop is called.
func (w *shellyWatcher) Start() {
	go func() {
		for {
			err := w.session()
			select {
			case <-w.stop:
				return
			default:
			}
			w.send(shellyStatusUpdate{connected: false, err: err})
			select {
			case <-w.stop:
				return
			case <-time.After(shellyWSReconnect):
			}
		}
	}()
}

// Stop ends the watcher and closes any open connection.
func (w *shellyWatcher) Stop() {
	close(w.stop)
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.conn != nil {
		w.conn.Close()
	}
}

// send delivers an update unless the watcher is stopping or the client has
// fallen behind, in which case the next update carries the same state.
func (w *shellyWatcher) send(u shellyStatusUpdate) {
	select {
	case w.updates <- u:
	case <-w.stop:
	default:
	}
}

// session dials the device and reads frames until the connection fails.
func (w *shellyWatcher) session() error {
	dialer := websocket.Dialer{HandshakeTimeout: time.Second * 10}
	conn, _, err := dialer.Dial("ws://"+w.ip+"/rpc", nil)
	if err != nil {
		return err
	}

	w.lock.Lock()
	select {
	case <-w.stop:
		w.lock.Unlock()
		conn.Close()
		return nil
	default:
	}
	w.conn = conn
	w.lock.Unlock()

	defer func() {
		w.lock.Lock()
		if w.conn == conn {
			w.conn = nil
		}
		w.lock.Unlock()
		conn.Close()
	}()

	if err := w.writeJSON(conn, shellyRPCRequest{
		ID: 1, Src: shellyRPCSrc, Method: "Shelly.GetStatus",
	}); err != nil {
		return err
	}

	if err := conn.SetReadDeadline(time.Now().Add(shellyWSTimeout)); err != nil {
		return err
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(shellyWSTimeout))
	})

	pinger := time.NewTicker(shellyWSPing)
	defer pinger.Stop()
	pingDone := make(chan struct{})
	defer close(pingDone)
	go func() {
		for {
			select {
			case <-pingDone:
				return
			case <-pinger.C:
				w.lock.Lock()
				err := conn.WriteControl(websocket.PingMessage, nil,
					time.Now().Add(time.Second*5))
				w.lock.Unlock()
				if err != nil {
					conn.Close()
					return
				}
			}
		}
	}()

	for {
		var frame shellyRPCFrame
		if err := conn.ReadJSON(&frame); err != nil {
			return err
		}
		if err := conn.SetReadDeadline(time.Now().Add(shellyWSTimeout)); err != nil {
			return err
		}
		w.handleFrame(frame)
	}
}

// writeJSON serializes writes so the ping goroutine and this one cannot
// interleave frames.
func (w *shellyWatcher) writeJSON(conn *websocket.Conn, v interface{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	return conn.WriteJSON(v)
}

func (w *shellyWatcher) handleFrame(frame shellyRPCFrame) {
	var raw json.RawMessage
	full := false

	switch {
	case frame.Error != nil:
		log.Println("Shelly rpc error from", w.ip, frame.Error)
		return
	case frame.Result != nil:
		// The reply to the Shelly.GetStatus sent on connect.
		raw, full = frame.Result, true
	case frame.Method == "NotifyFullStatus":
		raw, full = frame.Params, true
	case frame.Method == "NotifyStatus":
		raw, full = frame.Params, false
	default:
		// NotifyEvent and anything else the device sends carries no status.
		return
	}

	var status map[string]json.RawMessage
	if err := json.Unmarshal(raw, &status); err != nil {
		log.Println("Shelly: error decoding status from", w.ip, err)
		return
	}
	// A NotifyStatus carries a timestamp alongside the components that changed.
	delete(status, "ts")

	w.send(shellyStatusUpdate{connected: true, full: full, status: status})
}
