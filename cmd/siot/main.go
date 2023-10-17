// This is the main Simple IoT Program
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/install"
	"github.com/simpleiot/simpleiot/server"
)

// goreleaser will replace version with Git version. You can also pass version
// into the version into the go build:
//
//	go build -ldflags="-X main.version=1.2.3"
var version = "Development"

func main() {
	// global options
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagVersion := flags.Bool("version", false, "Print app version")
	flagID := flags.String("id", "", "ID for the instance")
	flags.Usage = func() {
		fmt.Println("usage: siot [OPTION]... COMMAND [OPTION]...")
		fmt.Println("Global options:")
		flags.PrintDefaults()
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("  - serve (start the SIOT server)")
		fmt.Println("  - log (log SIOT messages)")
		fmt.Println("  - store (store maint, requires server to be running)")
		fmt.Println("  - install (install SIOT and register service)")
		fmt.Println("  - import (import nodes from YAML file)")
	}

	_ = flags.Parse(os.Args[1:])

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
		if err := runServer(args[1:], version, *flagID); err != nil {
			log.Println("Simple IoT stopped, reason: ", err)
		}
	case "log":
		runLog(args[1:])
	case "store":
		runStore(args[1:])
	case "install":
		runInstall(args[1:])
	case "import":
		runImport(args[1:])
	default:
		log.Fatal("Unknown command; options: serve, log, store")
	}
}

func runServer(args []string, version string, id string) error {
	options, err := server.Args(args)
	if err != nil {
		return err
	}

	options.AppVersion = version
	options.ID = id

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

	g.Add(siot.Run, siot.Stop)

	g.Add(run.SignalHandler(context.Background(),
		syscall.SIGINT, syscall.SIGTERM))

	// Load the default SIOT clients -- you can replace this with a customized
	// list
	clients, err := client.DefaultClients(nc)
	if err != nil {
		return err
	}
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

var defaultNatsServer = "nats://127.0.0.1:4222"

func runLog(args []string) {
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

	authToken := *flagAuthToken
	if authToken == "" {
		authTokenE := os.Getenv("SIOT_AUTH_TOKEN")
		if authTokenE != "" {
			authToken = authTokenE
		}
	}

	client.Log(natsServer, authToken)

	select {}
}

func runStore(args []string) {
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

	authToken := *flagAuthToken
	if authToken == "" {
		authTokenE := os.Getenv("SIOT_AUTH_TOKEN")
		if authTokenE != "" {
			authToken = authTokenE
		}
	}

	opts := client.EdgeOptions{
		URI:       natsServer,
		AuthToken: authToken,
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
		Connected: func() {
			log.Println("NATS Connected")
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

func runCommand(cmd string) (string, error) {
	c := exec.Command("sh", "-c", cmd)
	ret, err := c.CombinedOutput()
	return string(ret), err
}

type serviceData struct {
	SiotData      string
	SiotPath      string
	SystemdTarget string
}

func runInstall(args []string) {
	flags := flag.NewFlagSet("install", flag.ExitOnError)

	if err := flags.Parse(args); err != nil {
		log.Fatal("error: ", err)
	}

	if runtime.GOOS != "linux" {
		log.Fatal("Install is only supported on Linux systems")
	}

	currentUser, err := user.Current()
	if err != nil {
		log.Fatal("Error getting user: ", err)
	}

	isRoot := false
	if currentUser.Username == "root" {
		isRoot = true
	}

	serviceDir := path.Join(currentUser.HomeDir, ".config/systemd/user")
	dataDir := path.Join(currentUser.HomeDir, ".local/share/siot")

	if isRoot {
		serviceDir = path.Join("/etc/systemd/system")
		dataDir = "/var/lib/siot"
	}

	mkdirs := []string{serviceDir, dataDir}

	for _, d := range mkdirs {
		err := os.MkdirAll(d, 0755)
		if err != nil {
			log.Fatalf("Error creating dir %v: %v\n", d, err)
		}
	}

	servicePath := path.Join(serviceDir, "siot.service")

	siotPath, err := os.Executable()
	if err != nil {
		log.Fatal("Error getting SIOT path: ", err)
	}

	log.Println("Installing service file: ", servicePath)
	log.Println("SIOT executable location: ", siotPath)
	log.Println("SIOT data location: ", dataDir)

	_, err = os.Stat(servicePath)

	if err == nil {
		log.Println("Service file exists, do you want to replace it? (yes/no)")

		var input string

		_, err := fmt.Scan(&input)
		if err != nil {
			log.Fatal("Error getting input: ", err)
		}

		input = strings.ToLower(input)

		if input != "yes" {
			log.Fatal("Exitting install")
		}
	}

	siotService, err := install.Content.ReadFile("siot.service")
	if err != nil {
		log.Fatal("Error reading embedded service file: ", err)
	}

	t, err := template.New("service").Parse(string(siotService))
	if err != nil {
		log.Fatal("Error parsing service template", err)
	}

	serviceOut, err := os.Create(servicePath)
	if err != nil {
		log.Fatal("Error creating service file: ", err)
	}

	sd := serviceData{
		SiotPath:      siotPath,
		SiotData:      dataDir,
		SystemdTarget: "default.target",
	}

	if isRoot {
		sd.SystemdTarget = "multi-user.target"
	}

	err = t.Execute(serviceOut, sd)

	if err != nil {
		log.Fatal("Error installing service file: ", err)
	}

	// start and enable service
	startCmd := "systemctl start siot"
	enableCmd := "systemctl enable siot"
	reloadCmd := "systemctl daemon-reload"

	if !isRoot {
		startCmd += " --user"
		enableCmd += " --user"
		reloadCmd += " --user"
	}

	cmds := []string{startCmd, enableCmd, reloadCmd}

	for _, c := range cmds {
		_, err := runCommand(c)
		if err != nil {
			log.Fatalf("Error running command: %v: %v\n", c, err)
		}
	}

	log.Println("Install success!")
	log.Println("Please update ports in service file if you want someting other than defaults")
}

func runImport(args []string) {
	flags := flag.NewFlagSet("import", flag.ExitOnError)

	flagParentID := flags.String("parentID", "", "Parent ID for import under. Default is root device")
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

	authToken := *flagAuthToken
	if authToken == "" {
		authTokenE := os.Getenv("SIOT_AUTH_TOKEN")
		if authTokenE != "" {
			authToken = authTokenE
		}
	}

	opts := client.EdgeOptions{
		URI:       natsServer,
		AuthToken: authToken,
		NoEcho:    true,
		Disconnected: func() {
			log.Println("NATS Disconnected")
		},
		Reconnected: func() {
			log.Println("NATS Reconnected")
		},
		Closed: func() {
			log.Fatal("NATS Closed")
		},
		Connected: func() {
			log.Println("NATS Connected")
		},
	}

	nc, err := client.EdgeConnect(opts)
	if err != nil {
		log.Fatal("Error connecting to NATS server: ", err)
	}

	yamlChan := make(chan []byte)

	go func() {
		// read YAML file from STDIN
		yaml, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal("Error reading YAML from stdin: ", err)
		}
		yamlChan <- yaml
	}()

	var yaml []byte

	select {
	case yaml = <-yamlChan:
	case <-time.After(time.Second * 2):
		log.Fatal("Error: timeout reading YAML from STDIN")
	}

	if *flagParentID == "" {
		root, err := client.GetRootNode(nc)
		if err != nil {
			log.Fatal("Error getting root node: ", err)
		}
		*flagParentID = root.ID
	}

	err = client.ImportNodes(nc, *flagParentID, yaml, "import", false)
	if err != nil {
		log.Fatal("Error importing nodes: ", err)
	}

	log.Println("Import success!")
}
