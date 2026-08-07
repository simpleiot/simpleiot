package client

import (
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"encoding/base64"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// File represents a file that a user uploads or is present in some location
type File struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Name        string `point:"name"`
	Data        string `point:"data"`
	Size        string `point:"size"`
	Binary      bool   `point:"binary"`
	Hash        string `point:"hash"`
	// Created is when the file node came into existence, in Unix seconds. It
	// is written once and never rewritten, so that a file keeps its place in
	// the order provisioning applies uploads in even after its contents are
	// replaced.
	Created float64 `point:"created"`
	// ProvisionHash and Error are written by provisioning when a file node is
	// used as a provisioning source, and are empty otherwise.
	ProvisionHash string `point:"provisionHash"`
	Error         string `point:"error"`
}

// GetContents reads the file contents and does any decoding necessary if it is a binary file
func (f *File) GetContents() ([]byte, error) {
	var ret []byte
	var err error

	if f.Binary {
		ret, err = base64.StdEncoding.DecodeString(f.Data)
	} else {
		ret = []byte(f.Data)
	}

	return ret, err
}

// FileClient is used to manage files
type FileClient struct {
	nc            *nats.Conn
	config        File
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
}

// NewFileClient ...
func NewFileClient(nc *nats.Conn, config File) Client {
	return &FileClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
	}
}

// Run the main logic for the file client
func (f *FileClient) Run() error {
	// stamp when this file came into existence, once and only once. Nodes
	// that predate this point get it the first time they are seen.
	if f.config.Created == 0 {
		p := data.NewPointFloat(data.PointTypeCreated, "", float64(time.Now().Unix()))

		err := SendNodePoints(f.nc, f.config.ID, data.Points{p}, false)
		if err != nil {
			log.Println("File: error sending created point: ", err)
		}
	}

exitFileClient:

	for {
		select {
		case <-f.stop:
			break exitFileClient

		case points := <-f.newPoints:
			// Update local configuration
			err := data.MergePoints(points.ID, points.Points, &f.config)
			if err != nil {
				return fmt.Errorf("merging points: %w", err)
			}

			for _, p := range points.Points {
				if p.Type == data.PointTypeData {
					// update md5 hash
					var fileData []byte

					fileData, err := f.config.GetContents()
					if err != nil {
						log.Println("Error decoding file contents: ", err)
						break
					}

					hash := md5.Sum(fileData)
					hashS := fmt.Sprintf("%x", hash)

					pts := data.Points{
						data.NewPointString(data.PointTypeHash, "", hashS),
						data.NewPointFloat(data.PointTypeSize, "", float64(len(fileData))),
					}

					e := SendNodePoints(f.nc, f.config.ID, pts, true)
					if e != nil {
						log.Println("File: error sending hash point: ", err)
					}
				}
			}
		}
	}

	return nil
}

// Stop stops the File Client
func (f *FileClient) Stop(error) {
	close(f.stop)
}

// Points is called when the client's node points are updated
func (f *FileClient) Points(nodeID string, points []data.Point) {
	f.newPoints <- NewPoints{
		ID:     nodeID,
		Points: points,
	}
}

// EdgePoints is called when the client's node edge points are updated
func (f *FileClient) EdgePoints(
	_ string, _ string, _ []data.Point,
) {
	// Do nothing
}
