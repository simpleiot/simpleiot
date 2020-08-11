package api

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// NatsListenForCmd listens for a file sent from server
func NatsListenForCmd(nc *nats.Conn, deviceID string, callback func(cmd data.DeviceCmd)) error {
	_, err := nc.Subscribe(fmt.Sprintf("device.%v.cmd", deviceID), func(m *nats.Msg) {
		cmdPb := &pb.DeviceCmd{}

		err := proto.Unmarshal(m.Data, cmdPb)

		if err != nil {
			log.Println("Error decoding cmd: ", err)
			err := nc.Publish(m.Reply, []byte("error decoding"))
			if err != nil {
				log.Println("Error replying to file download: ", err)
				return
			}
		}

		cmd := data.DeviceCmd{
			ID:     cmdPb.Id,
			Cmd:    cmdPb.Cmd,
			Detail: cmdPb.Detail,
		}

		callback(cmd)

		err = nc.Publish(m.Reply, []byte("OK"))
		if err != nil {
			log.Println("Error replying to cmd: ", err)
		}
	})

	return err
}

// NatsSendCmd sends a command to device via NATS
func NatsSendCmd(nc *nats.Conn, cmd data.DeviceCmd) error {
	cmdPb := &pb.DeviceCmd{
		Id:     cmd.ID,
		Cmd:    cmd.Cmd,
		Detail: cmd.Detail,
	}

	out, err := proto.Marshal(cmdPb)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("device.%v.cmd", cmd.ID)

	retry := 0
	for ; retry < 3; retry++ {
		msg, err := nc.Request(subject, out, time.Minute)

		if err != nil {
			log.Println("Error sending cmd, retrying: ", retry, err)
			continue
		}

		msgS := string(msg.Data)

		if msgS != "OK" {
			log.Println("Error from device when sending cmd: ", retry, msgS)
			continue
		}

		// we must have sent OK, break out of loop
		break
	}

	if retry >= 3 {
		return fmt.Errorf("Error sending cmd %v to device %v", cmd.Cmd, cmd.ID)
	}

	return nil
}
