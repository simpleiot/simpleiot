package api

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

type fileDownload struct {
	id   string
	name string
	data []byte
	seq  int32
}

// NatsListenForFile listens for a file sent from server
func NatsListenForFile(nc *nats.Conn, deviceID string, callback func(path string)) error {
	dl := fileDownload{}
	_, err := nc.Subscribe(fmt.Sprintf("device.%v.file", deviceID), func(m *nats.Msg) {
		chunk := &pb.FileChunk{}

		err := proto.Unmarshal(m.Data, chunk)

		if err != nil {
			log.Println("Error decoding file chunk: ", err)
			err := nc.Publish(m.Reply, []byte("error decoding"))
			if err != nil {
				log.Println("Error replying to file download: ", err)
				return
			}
		}

		if chunk.Seq == 0 {
			// we are starting a new stream
			dl.name = chunk.FileName
			dl.data = []byte{}
			dl.seq = 0
		} else if chunk.Seq != dl.seq+1 {
			log.Println("Seq # error in file download: ", dl.seq, chunk.Seq)
			err := nc.Publish(m.Reply, []byte("seq error"))
			if err != nil {
				log.Println("Error replying to file download: ", err)
				return
			}
		}

		// process data from server
		dl.data = append(dl.data, chunk.Data...)
		dl.seq = chunk.Seq

		switch chunk.State {
		case pb.FileChunk_ERROR:
			log.Println("Server error getting chunk")
			// reset download
			dl = fileDownload{}
		case pb.FileChunk_DONE:
			err := ioutil.WriteFile(dl.name, dl.data, 0644)
			if err != nil {
				log.Println("Error writing dl file: ", err)
				err := nc.Publish(m.Reply, []byte("error writing"))
				if err != nil {
					log.Println("Error replying to file download: ", err)
					return
				}
			}

			callback(dl.name)
		}

		err = nc.Publish(m.Reply, []byte("OK"))
		if err != nil {
			log.Println("Error replying to file download: ", err)
		}
	})

	return err
}

// NatsSendFileFromHTTP fetchs a file using http and sends via nats
func NatsSendFileFromHTTP(nc *nats.Conn, deviceID string, url string) error {
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

	return NatsSendFile(nc, deviceID, resp.Body, name)
}

// NatsSendFile can be used to send a file to a device
func NatsSendFile(nc *nats.Conn, deviceID string, reader io.Reader, name string) error {
	done := false
	seq := int32(0)

	// send file in chunks
	for {
		var err error
		data := make([]byte, 50*1024)
		c, err := reader.Read(data)
		data = data[:c]

		chunk := &pb.FileChunk{Seq: seq, Data: data}

		if seq == 0 {
			chunk.FileName = name
		}

		if err != nil {
			if err != io.EOF {
				chunk.State = pb.FileChunk_ERROR
			} else {
				chunk.State = pb.FileChunk_DONE
			}
			done = true
		}

		out, err := proto.Marshal(chunk)

		if err != nil {
			return err
		}

		subject := fmt.Sprintf("device.%v.file", deviceID)

		retry := 0
		for ; retry < 3; retry++ {
			msg, err := nc.Request(subject, out, time.Minute)

			if err != nil {
				log.Println("Error sending file, retrying: ", retry, err)
				continue
			}

			msgS := string(msg.Data)

			if msgS != "OK" {
				log.Println("Error from device when sending file: ", retry, msgS)
				continue
			}

			// we must have sent OK, break out of loop
			break
		}

		if retry >= 3 {
			return errors.New("Error sending file to device")
		}

		if done {
			break
		}

		seq++
	}

	return nil
}
