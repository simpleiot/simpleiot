package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

// goreleaser will replace version with Git version. You can also pass version
// into the version into the go build:
//   go build -ldflags="-X main.version=1.2.3"
var version = "Development"

func main() {
	// global options
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagVersion := flags.Bool("version", false, "Print app version")
	flags.Usage = func() {
		fmt.Println("usage: siot [OPTION]... COMMAND [OPTION]...")
		fmt.Println("Global options:")
		flags.PrintDefaults()
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("  - serve (start the SIOT server)")
		fmt.Println("  - log (log SIOT messages)")
		fmt.Println("  - store (store maint, requires server to be running)")
	}

	flags.Parse(os.Args[1:])

	if *flagVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Printf("SimpleIOT %v\n", version)

	// extract sub command and its arguments
	args := flags.Args()

	if len(args) < 1 {
		// run serve command by default
		args = []string{"serve"}
	}

	switch args[0] {
	case "serve":
		if err := runServer(args[1:], version); err != nil {
			log.Println("Simple IoT stopped, reason: ", err)
		}
	case "log":
		runLog(args[1:])
	case "store":
		runStore(args[1:])
	default:
		log.Fatal("Unknown command; options: serve, log, store")
	}
}

func runServer(args []string, version string) error {
	options, err := server.Args(args)
	if err != nil {
		return err
	}

	options.AppVersion = version

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

func runLog(args []string) {
	defaultNatsServer := "nats://localhost:4222"
	flags := flag.NewFlagSet("log", flag.ExitOnError)
	flagNatsServer := flags.String("natsServer", defaultNatsServer, "NATS Server")
	flagAuthToken := flags.String("token", "", "Auth token")

	if err := flags.Parse(args); err != nil {
		log.Fatal("error: ", err)
	}

	// only consider env if command line option is something different
	// that default
	natsServer := *flagNatsServer
	if natsServer == defaultNatsServer {
		natsServerE := os.Getenv("SIOT_NATS_SERVER")
		if natsServerE != "" {
			natsServer = natsServerE
		}
	}

	client.Log(natsServer, *flagAuthToken)

	select {}
}

func runStore(args []string) {
	defaultNatsServer := "nats://localhost:4222"
	flags := flag.NewFlagSet("store", flag.ExitOnError)
	flagNatsServer := flags.String("natsServer", defaultNatsServer, "NATS Server")
	flagAuthToken := flags.String("token", "", "Auth token")
	flagCheck := flags.Bool("check", false, "Check store")
	flagFix := flags.Bool("fix", false, "Fix store")

	if err := flags.Parse(args); err != nil {
		log.Fatal("error: ", err)
	}

	// only consider env if command line option is something different
	// that default
	natsServer := *flagNatsServer
	if natsServer == defaultNatsServer {
		natsServerE := os.Getenv("SIOT_NATS_SERVER")
		if natsServerE != "" {
			natsServer = natsServerE
		}
	}

	opts := client.EdgeOptions{
		URI:       *flagNatsServer,
		AuthToken: *flagAuthToken,
		NoEcho:    true,
		Disconnected: func() {
			log.Println("NATS Disconnected")
		},
		Reconnected: func() {
			log.Println("NATS Reconnected")
		},
		Closed: func() {
			log.Println("NATS Closed")
			os.Exit(0)
		},
	}

	nc, err := client.EdgeConnect(opts)

	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	switch {
	case *flagCheck:
		err := client.AdminStoreVerify(nc)
		if err != nil {
			log.Println("DB verify failed: ", err)
		} else {
			log.Println("DB verified :-)")
		}

	case *flagFix:
		err := client.AdminStoreMaint(nc)
		if err != nil {
			log.Println("DB maint failed: ", err)
		} else {
			log.Println("DB maint success :-)")
		}

	default:
		fmt.Println("Error, no operation given.")
		flags.Usage()
	}
}
