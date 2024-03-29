package client

// FIXME could probably find a better place for this file ...

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Note, this file is still in the api package (vs nats) as http bloats a build, and not
// all edge devices need http.

// Companion file in nats/file.go

// NatsSendFileFromHTTP fetchs a file using http and sends via nats. Callback provides % complete (0-100).
func NatsSendFileFromHTTP(nc *nats.Conn, deviceID string, url string, callback func(int)) error {
	var netClient = &http.Client{
		Timeout: time.Second * 60,
	}

	resp, err := netClient.Get(url)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Error reading file over http: " + resp.Status)
	}

	urlS := strings.Split(url, "/")
	if len(urlS) < 2 {
		return errors.New("Error parsing URL")
	}
	name := urlS[len(urlS)-1]

	return SendFile(nc, deviceID, resp.Body, name, func(bytesTx int) {
		callback(bytesTx)
	})
}
