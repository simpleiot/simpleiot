package nats

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// ListenForCmd listens for a file sent from server
func ListenForCmd(nc *nats.Conn, deviceID string, callback func(cmd data.NodeCmd)) error {
	_, err := nc.Subscribe(fmt.Sprintf("device.%v.cmd", deviceID), func(m *nats.Msg) {
		cmdPb := &pb.NodeCmd{}

		err := proto.Unmarshal(m.Data, cmdPb)

		if err != nil {
			log.Println("Error decoding cmd: ", err)
			err := nc.Publish(m.Reply, []byte("error decoding"))
			if err != nil {
				log.Println("Error replying to file download: ", err)
				return
			}
		}

		cmd := data.NodeCmd{
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

// SendCmd sends a command to device via NATS
func SendCmd(nc *nats.Conn, cmd data.NodeCmd, timeout time.Duration) error {
	cmdPb := &pb.NodeCmd{
		Id:     cmd.ID,
		Cmd:    cmd.Cmd,
		Detail: cmd.Detail,
	}

	out, err := proto.Marshal(cmdPb)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("device.%v.cmd", cmd.ID)

	msg, err := nc.Request(subject, out, timeout)

	if err != nil {
		return err
	}

	msgS := string(msg.Data)

	if msgS != "OK" && msgS != "" {
		return fmt.Errorf("Error sending cmd to %v, received: %v", cmd.ID, msgS)
	}

	return nil
}
