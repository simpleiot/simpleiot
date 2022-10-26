package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

var version = "Development"

func main() {
	log.Printf("SimpleIOT %v\n", version)

	if err := runSiot(os.Args, version); err != nil {
		log.Println("Simple IoT stopped, reason: ", err)
	}
}

func runSiot(args []string, version string) error {
	options, err := server.Args(args, version)
	if err != nil {
		return err
	}

	if options.LogNats {
		client.Log(options.NatsServer, options.AuthToken)
		select {}
	}

	var g run.Group

	siot, nc, err := server.NewServer(options)

	if err != nil {
		siot.Stop(nil)
		return fmt.Errorf("Error starting server: %v", err)
	}

	g.Add(siot.Start, siot.Stop)

	g.Add(run.SignalHandler(context.Background(),
		syscall.SIGINT, syscall.SIGTERM))

	// Load the default SIOT clients -- you can replace this with a customized
	// list
	clients, err := client.DefaultClients(nc)
	siot.AddClient(clients)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*9)

	// add check to make sure server started
	chStartCheck := make(chan struct{})
	g.Add(func() error {
		err := siot.WaitStart(ctx)
		if err != nil {
			return errors.New("Timeout waiting for SIOT to start")
		}
		log.Println("SIOT started")
		<-chStartCheck
		return nil
	}, func(err error) {
		cancel()
		close(chStartCheck)
	})

	return g.Run()
}
