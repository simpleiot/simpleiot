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
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"text/template"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
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
		fmt.Println("  - export (export nodes to YAML file)")
		fmt.Println("  - dump (describe a running instance for troubleshooting)")
		fmt.Println("  - provision (check provisioning files, or print what they would do)")
		fmt.Println("  - update (update to the latest release)")
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
		// gun serve command by default
		args = []string{"serve"}
	}

	switch args[0] {
	case "serve":
		if err := runServer(args[1:], version, *flagID); err != nil {
			log.Println("Simple IoT stopped, reason:", err)
			// exit non-zero so service managers that only restart on failure
			// bring us back up
			os.Exit(1)
		}
	case "log":
		runLog(args[1:])
	case "store":
		runStore(args[1:])
	case "install":
		runInstall(args[1:])
	case "import":
		runImport(args[1:])
	case "export":
		runExport(args[1:])
	case "dump":
		runDump(args[1:])
	case "provision":
		runProvision(args[1:])
	case "update":
		runUpdate(args[1:], version)
	default:
		log.Fatal("Unknown command; options: serve, log, store")
	}
}

func runServer(args []string, version string, id string) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	options, err := server.Args(args, flags)
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
		return fmt.Errorf("error starting server: %v", err)
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
			return errors.New("timeout waiting for SIOT to start")
		}
		log.Println("SIOT started")

		<-chStartCheck
		return nil
	}, func(_ error) {
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
		log.Println("Error connecting to NATS server:", err)
		os.Exit(-1)
	}

	switch {
	case *flagCheck:
		err := client.AdminStoreVerify(nc)
		if err != nil {
			log.Println("DB verify failed:", err)
		} else {
			log.Println("DB verified :-)")
		}

	case *flagFix:
		err := client.AdminStoreMaint(nc)
		if err != nil {
			log.Println("DB maint failed:", err)
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

	isRoot := currentUser.Username == "root"

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

	log.Println("Installing service file:", servicePath)
	log.Println("SIOT executable location:", siotPath)
	log.Println("SIOT data location:", dataDir)

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
			log.Fatal("Exiting install")
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
	log.Println("Please update ports in service file if you want something other than defaults")
}

func runImport(args []string) {
	flags := flag.NewFlagSet("import", flag.ExitOnError)

	flagNatsServer := flags.String("natsServer", defaultNatsServer, "NATS Server")
	flagAuthToken := flags.String("token", "", "Auth token")
	flagDryRun := flags.Bool("dryRun", false, "Print what the file would do without applying it")

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

	plan, err := client.ImportNodes(nc, yaml, "import", *flagDryRun)
	if err != nil {
		log.Fatal("Error importing nodes: ", err)
	}

	fmt.Print(plan.String())

	if len(plan.Errors) > 0 {
		log.Fatalf("Import finished with %v error(s)", len(plan.Errors))
	}

	if *flagDryRun {
		log.Println("Dry run, nothing was applied")
		return
	}

	log.Println("Import success!")
}

func runExport(args []string) {
	flags := flag.NewFlagSet("import", flag.ExitOnError)

	flagNodeID := flags.String("nodeID", "", "node ID to export. Default is root device")
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

	yaml, err := client.ExportNodes(nc, *flagNodeID)
	if err != nil {
		log.Fatal("Error export nodes: ", err)
	}

	_, err = os.Stdout.Write(yaml)

	if err != nil {
		log.Fatal("Error writing YAML to STDOUT: ", err)
	}

}

func runDump(args []string) {
	flags := flag.NewFlagSet("dump", flag.ExitOnError)

	flagNodeID := flags.String("nodeID", "", "node ID to dump. Default is the instance root")
	flagPoints := flags.Bool("points", false, "include every point, with its origin and time")
	flagStreams := flags.Bool("streams", false, "include the replication stream inventory")
	flagAll := flags.Bool("all", false, "same as -points -streams")
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

	out, err := client.DumpInstance(nc, client.DumpOptions{
		NodeID:  *flagNodeID,
		Points:  *flagPoints || *flagAll,
		Streams: *flagStreams || *flagAll,
	})
	if err != nil {
		log.Fatal("Error dumping instance: ", err)
	}

	if _, err := os.Stdout.WriteString(out); err != nil {
		log.Fatal("Error writing dump to STDOUT: ", err)
	}
}

func runProvision(args []string) {
	flags := flag.NewFlagSet("provision", flag.ExitOnError)

	flagDir := flags.String("dir", "", "directory of provisioning files")
	flagCheck := flags.Bool("check", false, "only parse the files, which needs no running instance")
	flagNatsServer := flags.String("natsServer", defaultNatsServer, "NATS Server")
	flagAuthToken := flags.String("token", "", "Auth token")

	if err := flags.Parse(args); err != nil {
		log.Fatal("error: ", err)
	}

	dir := *flagDir
	if dir == "" {
		dir = os.Getenv("SIOT_PROVISIONING_DIR")
	}

	if dir == "" {
		log.Fatal("Error: no provisioning directory given; pass -dir or set SIOT_PROVISIONING_DIR")
	}

	files, err := provisioningFiles(dir)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	if len(files) < 1 {
		log.Fatalf("No provisioning files found in %v", dir)
	}

	// parsing needs nothing but the files, which is what makes -check usable
	// in CI where there is no instance to apply against
	parsed := make([]data.NodeFile, len(files))
	failed := 0

	for i, f := range files {
		contents, err := os.ReadFile(f)
		if err != nil {
			log.Printf("%v: %v\n", filepath.Base(f), err)
			failed++
			continue
		}

		if err := yaml.Unmarshal(contents, &parsed[i]); err != nil {
			log.Printf("%v: %v\n", filepath.Base(f), err)
			failed++
			continue
		}

		if *flagCheck {
			log.Printf("%v: ok\n", filepath.Base(f))
		}
	}

	if failed > 0 {
		log.Fatalf("%v of %v file(s) could not be read", failed, len(files))
	}

	if *flagCheck {
		log.Printf("%v file(s) parsed\n", len(files))
		return
	}

	// without -check, say what applying these files would do to the instance
	natsServer := *flagNatsServer
	if natsServer == defaultNatsServer {
		if e := os.Getenv("SIOT_NATS_SERVER"); e != "" {
			natsServer = e
		}
	}

	authToken := *flagAuthToken
	if authToken == "" {
		authToken = os.Getenv("SIOT_AUTH_TOKEN")
	}

	nc, err := client.EdgeConnect(client.EdgeOptions{
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
	})

	if err != nil {
		log.Fatal("Error connecting to NATS server: ", err)
	}

	errors := 0

	for i, f := range files {
		fmt.Printf("--- %v ---\n", filepath.Base(f))

		plan, err := client.Apply(nc, parsed[i], client.ApplyOptions{DryRun: true})
		if err != nil {
			log.Printf("%v: %v\n", filepath.Base(f), err)
			errors++
			continue
		}

		fmt.Print(plan.String())
		errors += len(plan.Errors)
	}

	if errors > 0 {
		log.Fatalf("Finished with %v error(s)", errors)
	}

	log.Println("Dry run, nothing was applied")
}

// provisioningFiles lists the YAML files in a directory, in the order
// provisioning applies them.
func provisioningFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading %v: %w", dir, err)
	}

	var out []string

	for _, e := range entries {
		name := e.Name()

		if e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}

		switch filepath.Ext(name) {
		case ".yaml", ".yml":
			out = append(out, filepath.Join(dir, name))
		}
	}

	sort.Strings(out)

	return out, nil
}
