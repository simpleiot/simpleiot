package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

const (
	browserConfigPath = "/etc/default/yoe-kiosk-browser"
	eglfsConfigPath   = "/etc/default/eglfs.json"
	serviceName       = "yoe-kiosk-browser"
)

// BrowserClient is a SimpleIoT client that advertises a service via Browser and also
// queries for neighbouring devices running the same service
type BrowserClient struct {
	log          *log.Logger
	nc           *nats.Conn
	config       Browser
	stopCh       chan struct{}
	pointsCh     chan NewPoints
	edgePointsCh chan NewPoints
}

// Browser client configuration
type Browser struct {
	ID               string `node:"id"`
	Parent           string `node:"parent"`
	Description      string `point:"description"`
	URL              string `point:"url"`
	Disabled         bool   `point:"disabled"`
	Rotate           int    `point:"rotate"`
	KeyboardScale    bool   `point:"keyboardscale"`
	Fullscreen       bool   `point:"fullscreen"`
	DefaultDialogs   bool   `point:"defaultdialogs"`
	DialogColor      string `point:"dialogcolor"`
	TouchQuirk       bool   `point:"touchquirk"`
	RetryInterval    int    `point:"retryinterval"`
	ExceptionURL     string `point:"exceptionurl"`
	IgnoreCertErr    bool   `point:"ignorecerterr"`
	DisableSandbox   bool   `point:"disablesandbox"`
	ScreenResolution string `point:"screenresolution"`
	DisplayCard      string `point:"displaycard"`
	DebugPort        string `point:"debugport"`
}

// BrowserConfigFile outlines the config file for Yoe Kiosk Browser
type BrowserConfigFile struct {
	URL            string
	Rotate         int
	KeyboardScale  bool
	Fullscreen     bool
	DefaultDialogs bool
	DialogColor    string
	TouchQuirk     bool
	RetryInterval  int
	ExceptionURL   string
	IgnoreCertErr  bool
	DisableSandbox bool
	DebugPort      string
	XAuthority     string
	QtQpaPlatform  string
}

// EGLSFSOutputs outlintes the outputs array found in the EGLFS config file JSON
type EGLSFSOutputs struct {
	Name *string `json:"name"`
	Mode *string `json:"mode"`
}

// EGLFSConfigFile outlines the JSON file containing EGLFS configuration
type EGLFSConfigFile struct {
	Device   *string          `json:"device"`
	HwCursor *bool            `json:"hwcursor"`
	Pbuffers *bool            `json:"pbuffers"`
	Outputs  *[]EGLSFSOutputs `json:"outputs"`
}

// NewBrowserClient returns a new BrowserClient using its
// configuration read from the Client Manager
func NewBrowserClient(nc *nats.Conn, config Browser) Client {
	// TODO: Ensure only one Browser client exists
	return &BrowserClient{
		log:      log.New(os.Stderr, "Browser: ", log.LstdFlags|log.Lmsgprefix),
		nc:       nc,
		config:   config,
		stopCh:   make(chan struct{}),
		pointsCh: make(chan NewPoints),
	}
}

// reads the configuration from disk
func readEGLFSConfig() (*EGLFSConfigFile, error) {
	file, err := os.Open(eglfsConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s with error: %w", eglfsConfigPath, err)
	}
	defer file.Close()

	var config EGLFSConfigFile
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode %s with error: %w", eglfsConfigPath, err)
	}

	return &config, nil
}

// update the json file on disk with provided key/value pair
func updateEGLFSConfigInPlace(key, value string) error {
	config, err := readEGLFSConfig()
	if err != nil {
		return fmt.Errorf("failed to read EGLFS config: %w", err)
	}

	switch key {
	case "display-card":
		config.Device = &value
	case "resolution":
		// Update the first output's mode (resolution)
		if len(*config.Outputs) > 0 {
			(*config.Outputs)[0].Mode = &value
		} else {
			// If no outputs exist, create one with default name
			name := "LVDS-1"
			config.Outputs = &[]EGLSFSOutputs{
				{Name: &name, Mode: &value},
			}
		}
	default:
		return fmt.Errorf("unknown EGLFS configuration key: %s", key)
	}

	file, err := os.Create(eglfsConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create EGLFS config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print with 2-space indentation
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode EGLFS JSON: %w", err)
	}

	return nil
}

// read the current configuration from disk
func readBrowserConfig(file io.Reader) (*BrowserConfigFile, error) {
	config := &BrowserConfigFile{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "YOE_KIOSK_BROWSER_URL":
			config.URL = value
		case "YOE_KIOSK_BROWSER_ROTATE":
			val, _ := strconv.Atoi(value)
			config.Rotate = val
		case "YOE_KIOSK_BROWSER_KEYBOARD_SCALE":
			val, _ := strconv.ParseBool(value)
			config.KeyboardScale = val
		case "YOE_KIOSK_BROWSER_FULLSCREEN":
			val, _ := strconv.ParseBool(value)
			config.Fullscreen = val
		case "YOE_KIOSK_BROWSER_DEFAULT_DIALOGS":
			val, _ := strconv.ParseBool(value)
			config.DefaultDialogs = val
		case "YOE_KIOSK_BROWSER_DIALOG_COLOR":
			config.DialogColor = value
		case "YOE_KIOSK_BROWSER_TOUCH_QUIRK":
			val, _ := strconv.ParseBool(value)
			config.TouchQuirk = val
		case "YOE_KIOSK_BROWSER_RETRY_INTERVAL":
			val, _ := strconv.Atoi(value)
			config.RetryInterval = val
		case "YOE_KIOSK_BROWSER_EXCEPTION_URL":
			config.ExceptionURL = value
		case "YOE_KIOSK_BROWSER_IGNORE_CERT_ERR":
			val, _ := strconv.ParseBool(value)
			config.IgnoreCertErr = val
		case "QTWEBENGINE_DISABLE_SANDBOX":
			val, _ := strconv.ParseBool(value)
			config.DisableSandbox = val
		case "QTWEBENGINE_REMOTE_DEBUGGING":
			config.DebugPort = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s file: %w", browserConfigPath, err)
	}

	return config, nil
}

// update the file on disk with provided key/value pair
func updateConfigInPlace(key, value string) error {
	// Map user-friendly keys to actual environment variable names
	keyMap := map[string]string{
		"url":             "YOE_KIOSK_BROWSER_URL",
		"rotate":          "YOE_KIOSK_BROWSER_ROTATE",
		"keyboard-scale":  "YOE_KIOSK_BROWSER_KEYBOARD_SCALE",
		"fullscreen":      "YOE_KIOSK_BROWSER_FULLSCREEN",
		"default-dialogs": "YOE_KIOSK_BROWSER_DEFAULT_DIALOGS",
		"dialog-color":    "YOE_KIOSK_BROWSER_DIALOG_COLOR",
		"touch-quirk":     "YOE_KIOSK_BROWSER_TOUCH_QUIRK",
		"retry-interval":  "YOE_KIOSK_BROWSER_RETRY_INTERVAL",
		"exception-url":   "YOE_KIOSK_BROWSER_EXCEPTION_URL",
		"ignore-cert-err": "YOE_KIOSK_BROWSER_IGNORE_CERT_ERR",
		"disable-sandbox": "QTWEBENGINE_DISABLE_SANDBOX",
		"debug-port":      "QTWEBENGINE_REMOTE_DEBUGGING",
		"xauthority":      "XAUTHORITY",
		"qt-qpa-platform": "QT_QPA_PLATFORM",
	}

	actualKey, exists := keyMap[key]
	if !exists {
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Read the entire file
	file, err := os.Open(browserConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	keyFound := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line contains the key we want to modify
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				lineKey := strings.TrimSpace(parts[0])
				if lineKey == actualKey {
					// Replace the value while preserving any formatting
					lines = append(lines, fmt.Sprintf("%s=%s", actualKey, value))
					keyFound = true
					continue
				}
			}
		}

		// Keep the original line if it's not the key we're modifying
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// If the key wasn't found, append it to the end
	if !keyFound {
		lines = append(lines, fmt.Sprintf("%s=%s", actualKey, value))
	}

	file, err = os.Create(browserConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return nil
}

func restartService(serviceName string) error {
	return exec.Command("/usr/bin/systemctl", "restart", serviceName).Run()
}

func disableService(serviceName string) error {
	return exec.Command("/usr/bin/systemctl", "disable", serviceName).Run()
}

func enableService(serviceName string) error {
	return exec.Command("/usr/bin/systemctl", "enable", serviceName).Run()
}

// Run starts the Browser Client
func (c *BrowserClient) Run() error {
	str := "Starting Browser client"
	if c.config.Disabled {
		str += " (currently disabled)"
	}
	c.log.Println(str)

	init := func() error {

		file, err := os.Open(browserConfigPath)
		if err != nil {
			return fmt.Errorf("failed to open %s with error: %w", browserConfigPath, err)
		}
		defer file.Close()

		browserConfig, err := readBrowserConfig(file)
		if err != nil {
			return err
		}

		c.config.URL = browserConfig.URL
		c.config.Rotate = browserConfig.Rotate
		c.config.KeyboardScale = browserConfig.KeyboardScale
		c.config.Fullscreen = browserConfig.Fullscreen
		c.config.DefaultDialogs = browserConfig.DefaultDialogs
		c.config.DialogColor = browserConfig.DialogColor
		c.config.TouchQuirk = browserConfig.TouchQuirk
		c.config.RetryInterval = browserConfig.RetryInterval
		c.config.ExceptionURL = browserConfig.ExceptionURL
		c.config.IgnoreCertErr = browserConfig.IgnoreCertErr
		c.config.DisableSandbox = browserConfig.DisableSandbox

		eglfsConfig, err := readEGLFSConfig()
		if err != nil {
			return err
		}

		if eglfsConfig.Device != nil {
			c.config.DisplayCard = *eglfsConfig.Device
		}
		if eglfsConfig.Outputs != nil && (*eglfsConfig.Outputs)[0].Mode != nil {
			c.config.ScreenResolution = *(*eglfsConfig.Outputs)[0].Mode
		}
		c.ValidatePoints()

		return nil
	}

	err := init()
	if err != nil {
		c.log.Println("init returned with", err)
		return err
	}

loop:
	for {
		select {
		case <-c.stopCh:
			log.Println("Stopping Browser client:", c.config.Description)
			break loop

		case pts := <-c.pointsCh:
			log.Println("Received point - ", pts)

			err := data.MergePoints(pts.ID, pts.Points, &c.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}

			c.ValidatePoints()

			// Update the configuration in place

			if err := updateConfigInPlace("url", c.config.URL); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("rotate", strconv.Itoa(c.config.Rotate)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("keyboard-scale", boolToIntString(c.config.KeyboardScale)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("fullscreen", boolToIntString(c.config.Fullscreen)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("default-dialogs", boolToIntString(c.config.DefaultDialogs)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("dialog-color", c.config.DialogColor); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("touch-quirk", boolToIntString(c.config.TouchQuirk)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("retry-interval", strconv.Itoa(c.config.RetryInterval)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("exception-url", c.config.ExceptionURL); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("ignore-cert-err", boolToIntString(c.config.IgnoreCertErr)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("disable-sandbox", boolToIntString(c.config.DisableSandbox)); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateConfigInPlace("debug-port", c.config.DebugPort); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateEGLFSConfigInPlace("display-card", c.config.DisplayCard); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}
			if err := updateEGLFSConfigInPlace("resolution", c.config.ScreenResolution); err != nil {
				c.log.Printf("Error updating config: %v", err)
			}

			if c.config.Disabled {
				err = disableService(serviceName)
				if err != nil {
					c.log.Println("Error disabling service ", serviceName)
				}
			} else {
				err = enableService(serviceName)
				if err != nil {
					c.log.Println("Error enabling service ", serviceName)
				}
				err = restartService(serviceName)
				if err != nil {
					c.log.Println("Error restarting service ", serviceName)
				}
			}

		case pts := <-c.edgePointsCh:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &c.config)
			if err != nil {
				log.Println("error merging new points:", err)
			}
		}
	}
	return nil
}

// ValidatePoints sets points to their default state if unset.
func (c *BrowserClient) ValidatePoints() {
	validateString := func(value string, fallback string, nc *nats.Conn, ID string, pointType string) (string, error) {
		val := value
		if val == "" {
			val = fallback
		}
		c.log.Printf("Setting %s to: %s", pointType, val)
		err := SendNodePoint(nc, ID, data.Point{
			Time: time.Now(),
			Type: pointType,
			Key:  "0",
			Text: val}, false)
		if err != nil {
			c.log.Println("Error sending point: ", err)
			return val, err
		}
		return val, nil
	}

	validateInt := func(value int, fallback int, nc *nats.Conn, ID string, pointType string) (int, error) {
		val := value
		if val < 0 {
			val = fallback
		}
		c.log.Printf("Setting %s to: %d", pointType, val)
		err := SendNodePoint(nc, ID, data.Point{
			Time:  time.Now(),
			Type:  pointType,
			Key:   "0",
			Value: float64(val)}, false)
		if err != nil {
			c.log.Println("Error sending point: ", err)
			return val, err
		}
		return val, nil
	}

	validateBool := func(value bool, nc *nats.Conn, ID string, pointType string) (bool, error) {
		c.log.Printf("Setting %s to: %t", pointType, value)
		err := SendNodePoint(nc, ID, data.Point{
			Time:  time.Now(),
			Type:  pointType,
			Key:   "0",
			Value: data.BoolToFloat(value)}, false)
		if err != nil {
			c.log.Println("Error sending point: ", err)
			return value, err
		}
		return value, nil
	}

	defaults := struct {
		URL              string
		Disabled         bool
		Rotate           int
		DefaultDialogs   bool
		DialogColor      string
		TouchQuirk       bool
		RetryInterval    int
		ExceptionURL     string
		IgnoreCertErr    bool
		DisableSandbox   bool
		DebugPort        string
		ScreenResolution string
		DisplayCard      string
	}{
		"http://localhost:8080",
		false,
		0,
		false,
		"#D91824",
		true,
		10,
		"",
		true,
		true,
		"",
		"1024x600",
		"/dev/dri/card0",
	}

	c.config.URL, _ = validateString(c.config.URL, defaults.URL, c.nc, c.config.ID, data.PointTypeURL)
	c.config.Disabled, _ = validateBool(c.config.Disabled, c.nc, c.config.ID, data.PointTypeDisabled)
	c.config.Rotate, _ = validateInt(c.config.Rotate, defaults.Rotate, c.nc, c.config.ID, data.PointTypeRotate)
	c.config.KeyboardScale, _ = validateBool(c.config.KeyboardScale, c.nc, c.config.ID, data.PointTypeKeyboardScale)
	c.config.Fullscreen, _ = validateBool(c.config.Fullscreen, c.nc, c.config.ID, data.PointTypeFullscreen)
	c.config.DefaultDialogs, _ = validateBool(c.config.DefaultDialogs, c.nc, c.config.ID, data.PointTypeDefaultDialogs)
	c.config.TouchQuirk, _ = validateBool(c.config.TouchQuirk, c.nc, c.config.ID, data.PointTypeTouchQuirk)
	c.config.IgnoreCertErr, _ = validateBool(c.config.IgnoreCertErr, c.nc, c.config.ID, data.PointTypeIgnoreCertErr)
	c.config.DisableSandbox, _ = validateBool(c.config.DisableSandbox, c.nc, c.config.ID, data.PointTypeDisableSandbox)
	c.config.URL, _ = validateString(c.config.URL, defaults.URL, c.nc, c.config.ID, data.PointTypeURL)
	c.config.DialogColor, _ = validateString(c.config.DialogColor, defaults.DialogColor, c.nc, c.config.ID, data.PointTypeDialogColor)
	c.config.RetryInterval, _ = validateInt(c.config.RetryInterval, defaults.RetryInterval, c.nc, c.config.ID, data.PointTypeRetryInterval)
	c.config.ExceptionURL, _ = validateString(c.config.ExceptionURL, defaults.ExceptionURL, c.nc, c.config.ID, data.PointTypeExceptionURL)
	c.config.DebugPort, _ = validateString(c.config.DebugPort, defaults.DebugPort, c.nc, c.config.ID, data.PointTypeDebugPort)
	c.config.ScreenResolution, _ = validateString(c.config.ScreenResolution, defaults.ScreenResolution, c.nc, c.config.ID, data.PointTypeScreenResolution)
	c.config.DisplayCard, _ = validateString(c.config.DisplayCard, defaults.DisplayCard, c.nc, c.config.ID, data.PointTypeDisplayCard)
}

func boolToIntString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Stop stops the Browser Client
func (c *BrowserClient) Stop(error) {
	close(c.stopCh)
}

// Points is called when the client's node points are updated
func (c *BrowserClient) Points(nodeID string, points []data.Point) {
	c.pointsCh <- NewPoints{
		ID:     nodeID,
		Points: points,
	}
}

// EdgePoints is called by the Manager when new edge points for this
// node are received.
func (c *BrowserClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	c.edgePointsCh <- NewPoints{nodeID, parentID, points}
}
