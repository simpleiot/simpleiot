package server

import (
	"flag"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/system"
)

// Args parses common SIOT command line options
func Args(args []string, flags *flag.FlagSet) (Options, error) {
	defaultNatsServer := "nats://127.0.0.1:4222"

	// =============================================
	// Command line options
	// =============================================
	if flags == nil {
		flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}

	// configuration options
	flagDebugHTTP := flags.Bool("debugHttp", false, "dump http requests")
	flagDebugLifecycle := flags.Bool("debugLifecycle", false, "debug program lifecycle")
	flagNatsServer := flags.String("natsServer", defaultNatsServer, "NATS Server")
	flagNatsDisableServer := flags.Bool("natsDisableServer", false, "disable NATS server (if you want to run NATS separately)")
	flagStore := flags.String("store", "siot.sqlite", "store file, default siot.sqlite")
	flagResetStore := flags.Bool("resetStore", false, "permanently wipe data in store at start-up")
	flagAuthToken := flags.String("token", "", "auth token")
	flagSyslog := flags.Bool("syslog", false, "log to syslog instead of stdout")
	flagDev := flags.Bool("dev", false, "run server in development mode")
	flagCustomUIDir := flags.String("customUIDir", "", "pass custom UI directory")
	flagUIAssetsDebug := flags.Bool("UIAssetsDebug", false, "Dump asset files for debugging")

	if err := flags.Parse(args); err != nil {
		return Options{}, err
	}

	// =============================================
	// General Setup
	// =============================================

	// set up local database
	dataDir := os.Getenv("SIOT_DATA")
	if dataDir == "" {
		dataDir = "./"
	}

	// populate files in file system
	err := files.UpdateFiles(dataDir)

	if err != nil {
		log.Println("Error updating files:", err)
		os.Exit(-1)
	}

	storeFilePath := path.Join(dataDir, *flagStore)

	// =============================================
	// NATS stuff
	// =============================================

	// populate general args
	natsPort := 4222

	natsPortE := os.Getenv("SIOT_NATS_PORT")
	if natsPortE != "" {
		n, err := strconv.Atoi(natsPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_PORT:", err)
			os.Exit(-1)
		}
		natsPort = n
	}

	natsHTTPPort := 8222

	natsHTTPPortE := os.Getenv("SIOT_NATS_HTTP_PORT")
	if natsHTTPPortE != "" {
		n, err := strconv.Atoi(natsHTTPPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_HTTP_PORT:", err)
			os.Exit(-1)
		}
		natsHTTPPort = n
	}

	natsWSPort := 9222
	natsWSPortE := os.Getenv("SIOT_NATS_WS_PORT")
	if natsWSPortE != "" {
		n, err := strconv.Atoi(natsWSPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_WS_PORT:", err)
			os.Exit(-1)
		}
		natsWSPort = n
	}

	natsServer := *flagNatsServer
	// only consider env if command line option is something different
	// that default
	if natsServer == defaultNatsServer {
		natsServerE := os.Getenv("SIOT_NATS_SERVER")
		if natsServerE != "" {
			natsServer = natsServerE
		}
	}

	natsTLSCert := os.Getenv("SIOT_NATS_TLS_CERT")
	natsTLSKey := os.Getenv("SIOT_NATS_TLS_KEY")
	natsTLSTimeoutS := os.Getenv("SIOT_NATS_TLS_TIMEOUT")

	natsTLSTimeout := 0.5

	if natsTLSTimeoutS != "" {
		natsTLSTimeout, err = strconv.ParseFloat(natsTLSTimeoutS, 64)
		if err != nil {
			log.Println("Error parsing nats TLS timeout:", err)
			os.Exit(-1)
		}
	}

	authToken := os.Getenv("SIOT_AUTH_TOKEN")
	if *flagAuthToken != "" {
		authToken = *flagAuthToken
	}

	if *flagSyslog {
		err := system.EnableSyslog()
		if err != nil {
			log.Println("Error enabling syslog:", err)
		}
	}

	// finally, start web server
	port := os.Getenv("SIOT_HTTP_PORT")
	if port == "" {
		port = "8118"
	}

	osVersionField := os.Getenv("OS_VERSION_FIELD")
	if osVersionField == "" {
		osVersionField = "VERSION"
	}

	// set up particle connection if configured
	// todo -- move this to a node
	particleAPIKey := os.Getenv("SIOT_PARTICLE_API_KEY")

	// TODO, convert this to builder pattern
	o := Options{
		StoreFile:         storeFilePath,
		ResetStore:        *flagResetStore,
		HTTPPort:          port,
		DebugHTTP:         *flagDebugHTTP,
		DebugLifecycle:    *flagDebugLifecycle,
		NatsServer:        natsServer,
		NatsDisableServer: *flagNatsDisableServer,
		NatsPort:          natsPort,
		NatsHTTPPort:      natsHTTPPort,
		NatsWSPort:        natsWSPort,
		NatsTLSCert:       natsTLSCert,
		NatsTLSKey:        natsTLSKey,
		NatsTLSTimeout:    natsTLSTimeout,
		AuthToken:         authToken,
		ParticleAPIKey:    particleAPIKey,
		OSVersionField:    osVersionField,
		Dev:               *flagDev,
		CustomUIDir:       *flagCustomUIDir,
		UIAssetsDebug:     *flagUIAssetsDebug,
	}

	return o, nil

}
