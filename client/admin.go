package client

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
)

// AdminDbVerify can be used verify the database
func AdminDbVerify(nc *nats.Conn) error {
	resp, err := nc.Request("admin.dbVerify", nil, time.Second*20)
	if err != nil {
		return err
	}

	if len(resp.Data) > 0 {
		return errors.New(string(resp.Data))
	}

	return nil
}
