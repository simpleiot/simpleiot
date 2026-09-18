package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// websocketProxy forwards a WebSocket opened on the HTTP port to the NATS
// WebSocket listener, so a browser needs one port. The NATS server sees
// every proxied connection arrive from loopback, which is why the proxy
// itself applies the loopback rule for the shared token: under device
// auth required, a CONNECT carrying a token from a remote address is
// refused here, before it can look local.
type websocketProxy struct {
	backend            string
	deviceAuthRequired bool
	upgrader           websocket.Upgrader
}

func newWebsocketProxy(backend string, deviceAuthRequired bool) *websocketProxy {
	return &websocketProxy{
		backend:            backend,
		deviceAuthRequired: deviceAuthRequired,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// the NATS server checks the page origin against its own
			// allowed list; the header is forwarded to it below
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (p *websocketProxy) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	header := http.Header{}
	if origin := req.Header.Get("Origin"); origin != "" {
		header.Set("Origin", origin)
	}
	for _, prot := range req.Header["Sec-Websocket-Protocol"] {
		header.Add("Sec-WebSocket-Protocol", prot)
	}

	back, resp, err := websocket.DefaultDialer.Dial(p.backend, header)
	if err != nil {
		log.Printf("WebSocket proxy: error dialing %v: %v", p.backend, err)
		status := http.StatusServiceUnavailable
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			status = resp.StatusCode
		}
		http.Error(res, http.StatusText(status), status)
		return
	}
	defer back.Close()

	upgradeHeader := http.Header{}
	if hdr := resp.Header.Get("Sec-Websocket-Protocol"); hdr != "" {
		upgradeHeader.Set("Sec-Websocket-Protocol", hdr)
	}

	front, err := p.upgrader.Upgrade(res, req, upgradeHeader)
	if err != nil {
		// Upgrade has written the response
		return
	}
	defer front.Close()

	remoteLoopback := remoteIsLoopback(req.RemoteAddr)

	errc := make(chan error, 2)

	// browser to NATS, with the first message inspected
	go func() {
		first := true
		for {
			typ, msg, err := front.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if first {
				first = false
				if p.deviceAuthRequired && !remoteLoopback && connectHasToken(msg) {
					log.Printf("WebSocket proxy: refusing token from %v, device auth is required",
						req.RemoteAddr)
					errc <- websocket.ErrCloseSent
					return
				}
			}
			if err := back.WriteMessage(typ, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// NATS to browser
	go func() {
		for {
			typ, msg, err := back.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := front.WriteMessage(typ, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	<-errc
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	_ = front.WriteMessage(websocket.CloseMessage, closeMsg)
	_ = back.WriteMessage(websocket.CloseMessage, closeMsg)
}

// connectHasToken reports whether a client's first message is a NATS
// CONNECT that presents a token. A user signs in with user and pass and a
// device with an nkey, so a token is only ever the shared one.
func connectHasToken(msg []byte) bool {
	const verb = "CONNECT "
	if len(msg) < len(verb) || !bytes.EqualFold(msg[:len(verb)], []byte(verb)) {
		return false
	}
	var opts struct {
		Token string `json:"auth_token"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(msg[len(verb):]), &opts); err != nil {
		return false
	}
	return opts.Token != ""
}
