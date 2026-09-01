package server

import (
	"flag"
	"log"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/store"
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
	flagDeviceAuth := flags.String("deviceAuth", "",
		"'optional' (default) accepts the shared token from anywhere; 'required' accepts it only from this host, so remote devices need a credential")
	flagSyslog := flags.Bool("syslog", false, "log to syslog instead of stdout")
	flagDev := flags.Bool("dev", false, "run server in development mode")
	flagCustomUIDir := flags.String("customUIDir", "", "pass custom UI directory")
	flagUIAssetsDebug := flags.Bool("UIAssetsDebug", false, "Dump asset files for debugging")
	flagProvisioningDir := flags.String("provisioningDir", "",
		"directory of YAML files to apply at start-up and when they change (default <SIOT_DATA>/provisioning if it exists)")
	flagStoreMaxMsgsPerSubject := flags.Int64("storeMaxMsgsPerSubject", 0,
		"per-subject history retained in store streams (0 = default of 20000, -1 = unlimited); current state is always preserved")
	flagStoreCompression := flags.String("storeCompression", "",
		"store file compression ('s2' or 'none'); empty uses the default of s2")
	flagStoreSyncInterval := flags.String("storeSyncInterval", "",
		"JetStream file sync interval (Go duration, or 'always' to fsync every write); empty uses the NATS default of 2m")

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

	natsMQTTPort := 0
	natsMQTTPortE := os.Getenv("SIOT_NATS_MQTT_PORT")
	if natsMQTTPortE != "" {
		n, err := strconv.Atoi(natsMQTTPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_MQTT_PORT:", err)
			os.Exit(-1)
		}
		natsMQTTPort = n
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

	deviceAuth := *flagDeviceAuth
	if deviceAuth == "" {
		deviceAuth = os.Getenv("SIOT_DEVICE_AUTH")
	}

	switch deviceAuth {
	case "", DeviceAuthOptional, DeviceAuthRequired:
	default:
		log.Printf("Error parsing device auth %q: expected %q or %q",
			deviceAuth, DeviceAuthOptional, DeviceAuthRequired)
		os.Exit(-1)
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

	// =============================================
	// Provisioning
	// =============================================

	provisioningDir := *flagProvisioningDir

	if provisioningDir == "" {
		provisioningDir = os.Getenv("SIOT_PROVISIONING_DIR")
	}

	if provisioningDir == "" {
		// an image that ships the directory is provisioned without having to
		// say so; one that does not is unaffected
		d := path.Join(dataDir, "provisioning")
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			provisioningDir = d
		}
	}

	if provisioningDir != "" {
		log.Println("Provisioning from:", provisioningDir)
	}

	var provisioningInterval time.Duration

	if v := os.Getenv("SIOT_PROVISIONING_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Println("Error parsing SIOT_PROVISIONING_INTERVAL:", err)
			os.Exit(-1)
		}

		provisioningInterval = d
	}

	// =============================================
	// Store retention and durability
	// =============================================

	storeMaxMsgsPerSubject := *flagStoreMaxMsgsPerSubject
	if storeMaxMsgsPerSubject == 0 {
		if v := os.Getenv("SIOT_STORE_MAX_MSGS_PER_SUBJECT"); v != "" {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				log.Println("Error parsing SIOT_STORE_MAX_MSGS_PER_SUBJECT:", err)
				os.Exit(-1)
			}
			storeMaxMsgsPerSubject = n
		}
	}

	storeCompression := *flagStoreCompression
	if storeCompression == "" {
		storeCompression = os.Getenv("SIOT_STORE_COMPRESSION")
	}

	switch storeCompression {
	case "", store.CompressionS2, store.CompressionNone:
	default:
		log.Printf("Error parsing store compression %q: expected %q or %q",
			storeCompression, store.CompressionS2, store.CompressionNone)
		os.Exit(-1)
	}

	storeSyncIntervalS := *flagStoreSyncInterval
	if storeSyncIntervalS == "" {
		storeSyncIntervalS = os.Getenv("SIOT_STORE_SYNC_INTERVAL")
	}

	var storeSyncInterval time.Duration
	storeSyncAlways := false

	switch storeSyncIntervalS {
	case "":
	case "always":
		storeSyncAlways = true
	default:
		storeSyncInterval, err = time.ParseDuration(storeSyncIntervalS)
		if err != nil {
			log.Println("Error parsing store sync interval:", err)
			os.Exit(-1)
		}
	}

	// TODO, convert this to builder pattern
	o := Options{
		StoreFile:         storeFilePath,
		DataDir:           dataDir,
		ResetStore:        *flagResetStore,
		HTTPPort:          port,
		DebugHTTP:         *flagDebugHTTP,
		DebugLifecycle:    *flagDebugLifecycle,
		NatsServer:        natsServer,
		NatsDisableServer: *flagNatsDisableServer,
		NatsPort:          natsPort,
		NatsHTTPPort:      natsHTTPPort,
		NatsWSPort:        natsWSPort,
		NatsMQTTPort:      natsMQTTPort,
		NatsTLSCert:       natsTLSCert,
		NatsTLSKey:        natsTLSKey,
		NatsTLSTimeout:    natsTLSTimeout,
		AuthToken:         authToken,
		DeviceAuth:        deviceAuth,
		ParticleAPIKey:    particleAPIKey,
		OSVersionField:    osVersionField,
		Dev:               *flagDev,
		CustomUIDir:       *flagCustomUIDir,
		UIAssetsDebug:     *flagUIAssetsDebug,

		ProvisioningDir:      provisioningDir,
		ProvisioningInterval: provisioningInterval,

		StoreMaxMsgsPerSubject: storeMaxMsgsPerSubject,
		StoreCompression:       storeCompression,
		StoreSyncInterval:      storeSyncInterval,
		StoreSyncAlways:        storeSyncAlways,
	}

	return o, nil

}
