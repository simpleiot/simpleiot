package client

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
)

// AdminStoreVerify can be used verify the store
func AdminStoreVerify(nc *nats.Conn) error {
	resp, err := nc.Request("admin.storeVerify", nil, time.Second*20)
	if err != nil {
		return err
	}

	if len(resp.Data) > 0 {
		return errors.New(string(resp.Data))
	}

	return nil
}

// AdminStoreMaint can be used fix store issues
func AdminStoreMaint(nc *nats.Conn) error {
	resp, err := nc.Request("admin.storeMaint", nil, time.Second*20)
	if err != nil {
		return err
	}

	if len(resp.Data) > 0 {
		return errors.New(string(resp.Data))
	}

	return nil
}
