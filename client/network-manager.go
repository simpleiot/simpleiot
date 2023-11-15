//go:build linux
// +build linux

package client

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	nm "github.com/Wifx/gonetworkmanager"
	"github.com/godbus/dbus/v5"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

/*
	# NetworkManagerClient

	The NetworkManagerClient is responsible for synchronizing data between
	NetworkManager and the SimpleIoT node tree.

	```
	=========================       ======================== ---    device state     --> ==================
	| NetworkManager (DBus) | <---> | NetworkManagerClient |                             | SimpleIoT Tree |
	=========================       ======================== <-- connection settings --- ==================
	```

	The NetworkManagerClient only controls "SimpleIoT managed" connections
	within NetworkManager. In this way, other NetworkManager connections can be
	created and will not be updated or deleted by SimpleIoT. We designate a
	NetworkManager connection as "managed" by simply adding the "SimpleIoT:"
	prefix to the connection's ID. Managed connections will have a connection
	priority of 10.

	References:
	- https://developer-old.gnome.org/NetworkManager/stable/spec.html
*/

// NetworkManagerClient is a SimpleIoT client that manages network interfaces
// and their connections using NetworkManager via DBus
type NetworkManagerClient struct {
	log        *log.Logger
	nc         *nats.Conn
	config     NetworkManager
	stopCh     chan struct{}
	pointsCh   chan NewPoints
	nmSettings nm.Settings       // initialized on Run()
	nmObj      nm.NetworkManager // initialized on Run()
}

// NetworkManager client configuration
type NetworkManager struct {
	ID                      string                 `node:"id"`
	Parent                  string                 `node:"parent"`
	Disable                 bool                   `point:"disable"`
	Hostname                string                 `point:"hostname"`
	RequestWiFiScan         bool                   `point:"requestWiFiScan"`
	NetworkingEnabled       *bool                  `point:"networkingEnabled"`
	WirelessEnabled         *bool                  `point:"wirelessEnabled"`
	WirelessHardwareEnabled *bool                  `point:"wirelessHardwareEnabled"`
	Devices                 []NetworkManagerDevice `child:"networkManagerDevice"`
	Connections             []NetworkManagerConn   `child:"networkManagerConn"`
}

const dbusSyncInterval = time.Duration(60) * time.Second
const dBusPropertiesChanged = "org.freedesktop.DBus.Properties.PropertiesChanged"

// Print first error to logger
func logFirstError(log *log.Logger, errors []error) {
	if len(errors) > 0 {
		plural := ""
		if len(errors) != 1 {
			plural = "s; the first is"
		}
		log.Printf(
			"SyncConnections had %v error%v: %v",
			len(errors), plural, errors[0],
		)
	}
}

// NewNetworkManagerClient returns a new NetworkManagerClient using its
// configuration read from the Client Manager
func NewNetworkManagerClient(nc *nats.Conn, config NetworkManager) Client {
	// TODO: Ensure only one NetworkManager client exists
	return &NetworkManagerClient{
		log:      log.New(os.Stderr, "networkManager: ", log.LstdFlags|log.Lmsgprefix),
		nc:       nc,
		config:   config,
		stopCh:   make(chan struct{}),
		pointsCh: make(chan NewPoints),
	}
}

// Run starts the NetworkManager Client
func (c *NetworkManagerClient) Run() error {
	str := "Starting NetworkManager client"
	if c.config.Disable {
		str += " (currently disabled)"
	}
	c.log.Println(str)
	// c.log.Printf("config %+v", c.config)

	/*
		When starting this client, a few things will happen:

		1. We compare the list of SIOT "managed" connections to the SIOT tree
			(the tree has already been loaded into `c.config.Connections`)
			and perform a one-way synchronization to NetworkManager by
			creating, updating, and deleting connections.
		2. Perform a one-way synchronization **from** NetworkManager for
			NetworkManagerDevices in the SIOT tree. Start polling NetworkManager
			to continue syncing.
	*/

	// Note: Writes to `doSync` channel causes a sync operation to occur as soon
	// as possible. Generally, calling `queueSync` is preferred to leverage the
	// syncDelayTimer and rate limit sync operations.
	var syncDelayTimer *time.Timer
	syncDelayTimerLock := new(sync.Mutex)
	doSync := make(chan struct{}, 1)
	var syncTick time.Ticker
	var dbusSub <-chan *dbus.Signal

	init := func() error {
		var err error
		// Initialize NetworkManager settings object
		c.nmSettings, err = nm.NewSettings()
		if err != nil {
			return fmt.Errorf("error getting settings: %w", err)
		}

		// Initialize NetworkManager
		c.nmObj, err = nm.NewNetworkManager()
		if err != nil {
			return fmt.Errorf("error getting NetworkManager: %w", err)
		}
		dbusSub = c.nmObj.Subscribe()

		// Initialize NetworkManager sync ticker
		syncTick = *time.NewTicker(dbusSyncInterval)

		// Queue immediate sync operation
		if len(doSync) == 0 {
			doSync <- struct{}{}
		}

		return nil
	}

	cleanup := func() {
		// Stop tickers and nullify channels to ignore any unprocessed ticks
		syncTick.Stop()
		syncTick.C = nil

		// It's now safe to finish cleanup
		c.nmSettings = nil
		c.nmObj.Unsubscribe()
		c.nmObj = nil
		dbusSub = nil
		c.log.Println("Cleaned up")
	}

	queueSync := func() {
		syncDelayTimerLock.Lock()
		defer syncDelayTimerLock.Unlock()
		if syncDelayTimer == nil {
			syncDelayTimer = time.AfterFunc(5*time.Second, func() {
				syncDelayTimerLock.Lock()
				defer syncDelayTimerLock.Unlock()
				// Queue immediate sync operation
				if len(doSync) == 0 {
					doSync <- struct{}{}
				}
				syncDelayTimer = nil
			})
		}
		// else timer already running and will trigger sync soon
	}

	if !c.config.Disable {
		err := init()
		if err != nil {
			return err
		}
	}

loop:
	for {
		select {
		case <-c.stopCh:
			break loop
		case nodePoints := <-c.pointsCh:
			// log.Print(nodePoints)

			// Handle PointTypeDisable
			if nodePoints.ID == c.config.ID {
				for _, p := range nodePoints.Points {
					if p.Type == data.PointTypeDisable {
						if p.Value != 0 && !c.config.Disable {
							// Disable
							cleanup()
						} else if p.Value == 0 && c.config.Disable {
							// Re-initialize
							err := init()
							if err != nil {
								return err
							}
						}
					}
				}
			}
			// Update config
			err := data.MergePoints(nodePoints.ID, nodePoints.Points, &c.config)
			if err != nil {
				log.Println("Error merging points: ", err)
			}

			// Queue sync operation
			queueSync()
		case <-doSync:
			// Perform sync operations; abort on fatal error
			c.log.Println("Syncing with NetworkManager over D-Bus")
			errs, fatalErr := c.SyncConnections()
			// Abort on fatal error
			if fatalErr != nil {
				return fmt.Errorf("connection sync error: %w", fatalErr)
			}
			logFirstError(c.log, errs)

			// Synchronize devices with NetworkManager
			errs, fatalErr = c.SyncDevices()
			// Abort on fatal error
			if fatalErr != nil {
				return fmt.Errorf("device sync error: %w", fatalErr)
			}
			logFirstError(c.log, errs)

			// Synchronize hostname
			errs, fatalErr = c.SyncHostname()
			// Abort on fatal error
			if fatalErr != nil {
				return fmt.Errorf("connection sync error: %w", fatalErr)
			}
			logFirstError(c.log, errs)
		case <-syncTick.C:
			// Queue sync operation
			if len(doSync) == 0 {
				doSync <- struct{}{}
			}
		case sig, ok := <-dbusSub:
			if !ok {
				// D-Bus subscription closed
				dbusSub = nil
				break // select
			}
			if len(doSync) == 0 && (sig.Name == dBusPropertiesChanged ||
				strings.HasPrefix(sig.Name, "org.freedesktop.NetworkManager.Device")) {
				queueSync()
			} else {
				c.log.Printf("not triggering sync %v for %+v", sig.Name, sig)
			}
		}

		// Scan Wi-Fi networks if needed
		if c.config.RequestWiFiScan {
			// Create point to clear flag
			p := data.Point{
				Type:   "requestWiFiScan",
				Value:  0,
				Origin: c.config.ID,
			}

			// Trigger scan (error stored in Point)
			err := c.WifiScan()
			if err != nil {
				c.log.Printf("Error scanning for wireless APs: %v", err)
				p.Text = err.Error()
			}

			// Clear RequestWiFiScan
			err = SendNodePoint(c.nc, c.config.ID, p, true)
			// Log error only
			if err != nil {
				c.log.Printf("Error clearing requestWiFiScan: %v", err)
			}
			c.config.RequestWiFiScan = false
		}
	}
	cleanup()
	return nil
}

// SyncConnections performs a one-way synchronization of the NetworkManagerConn
// nodes in the SIOT tree with connections in NetworkManager via D-Bus.
// Returns a list of errors in the order in which they are encountered. If a
// fatal error occurs that aborts the sync operation, that will be included
// in `errs` and returned as `fatal`.
func (c *NetworkManagerClient) SyncConnections() (errs []error, fatal error) {
	// Build map of connection IDs from SIOT tree; value is Disabled flag
	connIDs := make(map[string]bool, len(c.config.Connections))
	for _, nmc := range c.config.Connections {
		if !nmc.Managed() {
			// TODO: Delete this node from the tree?
			errs = append(errs,
				fmt.Errorf("connection has invalid ID: %v", nmc.ID),
			)
		} else {
			connIDs[nmc.ID] = nmc.Disabled // add to the map
		}
	}

	// Get NetworkManagerConn from NetworkManager via D-Bus
	connections, err := c.nmSettings.ListConnections()
	if err != nil {
		fatal = fmt.Errorf("error listing connections: %w", err)
		errs = append(errs, fatal)
		return
	}

	// Build map of NetworkManager connections by ID
	type ConnectionResolved struct {
		Connection nm.Connection
		Resolved   NetworkManagerConn
	}
	nmConns := make(map[string]ConnectionResolved, len(connections))
	for _, conn := range connections {
		settings, err := conn.GetSettings()
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"error getting connection settings: %w", err,
			))
		}
		nmc := ResolveNetworkManagerConn(settings)
		nmc.Parent = c.config.ID

		// Ignore unmanaged connections
		if !nmc.Managed() {
			continue
		}

		// Delete if connection is missing from SIOT tree or disabled
		if disabled, ok := connIDs[nmc.ID]; !ok || disabled {
			c.log.Printf("Deleting connection %v", nmc.ID)
			if err := conn.Delete(); err != nil {
				errs = append(errs, fmt.Errorf(
					"error deleting connection %v: %w", nmc.ID, err,
				))
			}
		} else {
			nmConns[nmc.ID] = ConnectionResolved{conn, nmc}
		}
	}

	// Update NetworkManager from SIOT tree
	for _, treeConn := range c.config.Connections {
		if treeConn.Disabled {
			// this connection was already deleted above
			continue
		}
		conn, found := nmConns[treeConn.ID]
		var nmConnectionUUID string
		if found {
			nmConnectionUUID = conn.Resolved.UUID
			// Update existing connection
			if !treeConn.Equal(conn.Resolved) {
				c.log.Printf("Updating connection %v", treeConn.ID)
				err = conn.Connection.Update(treeConn.DBus())
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
		} else {
			c.log.Printf("Adding connection %v", treeConn.ID)
			// Create new connection profile in NetworkManager
			newConn, err := c.nmSettings.AddConnection(treeConn.DBus())
			if err != nil {
				errs = append(errs, err)
				continue
			}
			settings, err := newConn.GetSettings()
			if err != nil {
				c.log.Printf("Error getting new connection UUID: %v", err)
				continue
			}
			nmConnectionUUID = settings["connection"]["uuid"].(string)
		}

		// Update UUID in SIOT tree, if needed
		if treeConn.UUID != nmConnectionUUID {
			treeConn.UUID = nmConnectionUUID
			err = SendNodePoint(c.nc, treeConn.ID, data.Point{
				Type:   "uuid",
				Text:   nmConnectionUUID,
				Origin: c.config.ID,
			}, true)
			// Log error only
			if err != nil {
				c.log.Printf("Error setting new connection UUID: %v", err)
			}
		}
	}

	return
}

// SyncDevices performs a one-way synchronization of the devices in
// NetworkManager with the NetworkManagerDevices nodes in the SIOT tree via
// D-Bus. Additionally, the NetworkingEnabled and WirelessHardwareEnabled flags
// are copied to the SIOT tree; the WirelessEnabled flag is copied to
// NetworkManager if it is non-nil and copied to the SIOT tree if it is nil.
// Returns a list of errors in the order in which they are encountered.
// If a fatal error occurs that aborts the sync operation, that will be included
// in `errs` and returned as `fatal`.
func (c *NetworkManagerClient) SyncDevices() (errs []error, fatal error) {
	networkingEnabled, err := c.nmObj.GetPropertyNetworkingEnabled()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting network property: %w", err))
	}
	wirelessEnabled, err := c.nmObj.GetPropertyWirelessEnabled()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting network property: %w", err))
	}
	wirelessHwEnabled, err := c.nmObj.GetPropertyWirelessHardwareEnabled()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting network property: %w", err))
	}
	nmDevices, err := c.nmObj.GetAllDevices()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting devices: %w", err))
	}
	// Abort on any error received above
	if len(errs) > 0 {
		fatal = errs[0]
		return
	}

	// Sync Networking / Wireless enabled flags
	pts := data.Points{}
	if c.config.NetworkingEnabled == nil ||
		*c.config.NetworkingEnabled != networkingEnabled {
		c.config.NetworkingEnabled = &networkingEnabled
		p := data.Point{
			Type:   "networkingEnabled",
			Value:  0,
			Origin: c.config.ID,
		}
		if networkingEnabled {
			p.Value = 1
		}
		pts.Add(p)
	}
	if c.config.WirelessHardwareEnabled == nil ||
		*c.config.WirelessHardwareEnabled != wirelessHwEnabled {
		c.config.WirelessHardwareEnabled = &wirelessHwEnabled
		p := data.Point{
			Type:   "wirelessHardwareEnabled",
			Value:  0,
			Origin: c.config.ID,
		}
		if wirelessHwEnabled {
			p.Value = 1
		}
		pts.Add(p)
	}
	if c.config.WirelessEnabled == nil {
		// Copy to SIOT tree
		c.config.WirelessEnabled = &wirelessEnabled
		p := data.Point{
			Type:   "wirelessEnabled",
			Value:  0,
			Origin: c.config.ID,
		}
		if wirelessEnabled {
			p.Value = 1
		}
		pts.Add(p)
	} else if wirelessEnabled != *c.config.WirelessEnabled {
		// Copy to NetworkManager
		err = c.nmObj.SetPropertyWirelessEnabled(*c.config.WirelessEnabled)
		if err != nil {
			errs = append(errs,
				fmt.Errorf("error setting WirelessEnabled: %w", err),
			)
		}
	}

	// Send points
	if pts.Len() > 0 {
		err = SendNodePoints(
			c.nc,
			c.config.ID,
			pts,
			false,
		)
		if err != nil {
			errs = append(errs, fmt.Errorf("error updating enabled flags: %w", err))
		}
	}

	// Populate NetworkManager device info; keyed by their UUID
	deviceInfo := make(map[string]NetworkManagerDevice)
	for _, nmDevice := range nmDevices {
		dev, err := ResolveDevice(c.config.ID, nmDevice)
		if err != nil {
			errs = append(errs, fmt.Errorf("error resolving device: %w", err))
		}
		if !dev.Managed {
			continue // ignore devices not managed by NetworkManager
		}
		deviceInfo[dev.ID] = dev

		// data, _ := json.MarshalIndent(deviceInfo[dev.ID], "", "\t")
		// c.log.Println(string(data))
	}

	// Update devices already in SIOT tree
	for _, device := range c.config.Devices {
		nmDevice, ok := deviceInfo[device.ID]
		if ok {
			// Preserve AccessPoints
			nmDevice.AccessPoints = device.AccessPoints
			// Update device
			pts, err := data.DiffPoints(device, nmDevice)
			// Set origin on all points
			for _, p := range pts {
				p.Origin = c.config.ID
			}
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"error updating device %v: %w", device.ID, err,
				))
			}
			if len(pts) > 0 {
				c.log.Printf("Updating device %v\n%v", device.ID, pts)
				err := SendNodePoints(
					c.nc,
					device.ID,
					pts,
					false,
				)
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"error updating device %v: %w", device.ID, err,
					))
				}

				err = data.Decode(data.NodeEdgeChildren{
					NodeEdge: data.NodeEdge{
						ID:     device.ID,
						Parent: device.Parent,
						Points: pts,
					},
				}, &device)

				if err != nil {
					errs = append(errs, fmt.Errorf("error decoding data %v: %w", device.ID, err))
				}
			}
			// Remove from deviceInfo to avoid duplicating it later
			delete(deviceInfo, device.ID)
		} else {
			// Remove device
			c.log.Printf("Deleting device %v", device.ID)
			err := SendEdgePoint(
				c.nc,
				device.ID,
				device.Parent,
				data.Point{
					Type:   "tombstone",
					Value:  1,
					Origin: c.config.ID,
				},
				true,
			)
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"error deleting device %v: %w", device.ID, err,
				))
			}
		}
	}

	// Add devices not in SIOT tree
	// Note: updated devices are removed from deviceInfo above
	for _, nmDevice := range deviceInfo {
		c.log.Printf("Adding device %v", nmDevice.ID)
		err := SendNodeType(c.nc, nmDevice, c.config.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"error adding device %v: %w", nmDevice.ID, err,
			))
		}
	}

	return
}

// SyncHostname writes the hostname from the SimpleIoT tree to NetworkManager;
// however, if SimpleIoT does not have a hostname set, the current hostname
// will be stored in the tree instead.
func (c *NetworkManagerClient) SyncHostname() (errs []error, fatal error) {
	hostname, err := c.nmSettings.GetPropertyHostname()
	if err != nil {
		fatal = fmt.Errorf("error getting hostname: %w", err)
		errs = append(errs, fatal)
	}
	if c.config.Hostname == "" {
		// Write hostname to tree
		c.config.Hostname = hostname
		err = SendNodePoint(c.nc, c.config.ID, data.Point{
			Type:   "hostname",
			Text:   hostname,
			Origin: c.config.ID,
		}, true)
		errs = append(errs, err)
	} else if hostname != c.config.Hostname {
		// Write hostname to NetworkManager
		err = c.nmSettings.SaveHostname(c.config.Hostname)
		errs = append(errs, err)
	}
	return
}

// WifiScan scans for Wi-Fi access points using available Wi-Fi devices.
// When scanning is complete, the access points are saved as points on the
// NetworkManagerDevice node.
func (c *NetworkManagerClient) WifiScan() error {
	c.log.Println("Scanning for wireless APs...")

	nmDevices, err := c.nmObj.GetAllDevices()
	if err != nil {
		return fmt.Errorf("error getting devices: %w", err)
	}

	// Populate NetworkManager device info; keyed by their UUID
	nmDeviceMap := make(map[string]nm.DeviceWireless)
	for _, nmDevice := range nmDevices {
		dev, err := ResolveDevice(c.config.ID, nmDevice)
		if err != nil {
			return fmt.Errorf("error resolving device: %w", err)
		}
		if !dev.Managed {
			continue // ignore devices not managed by NetworkManager
		}
		if nmWifiDevice, ok := nmDevice.(nm.DeviceWireless); ok {
			nmDeviceMap[dev.ID] = nmWifiDevice
		}
	}

	// For each Wi-Fi device in SIOT tree
	found := false
	for devIndex, device := range c.config.Devices {
		if device.DeviceType != nm.NmDeviceTypeWifi.String() ||
			device.State == nm.NmDeviceStateUnmanaged.String() ||
			device.State == nm.NmDeviceStateUnavailable.String() {
			continue
		}
		found = true
		nmDevice, ok := nmDeviceMap[device.ID]
		if !ok {
			continue // no longer available
		}
		lastScan, _ := nmDevice.GetPropertyLastScan() // Ignore error

		// Get system uptime because LastScan property is milliseconds since
		// CLOCK_BOOTTIME
		var sysInfo syscall.Sysinfo_t
		err = syscall.Sysinfo(&sysInfo)
		if err != nil {
			return fmt.Errorf("error getting system uptime: %v", err)
		}

		// If last scan was more than RescanTimeoutSeconds ago, re-scan APs
		if int64(sysInfo.Uptime)-lastScan/1000 > RescanTimeoutSeconds {
			sigChan := c.nmObj.Subscribe()
			err = nmDevice.RequestScan()

			// Wait for "LastScan" property of device to be updated.
			// This indicates that the scan is complete
			timeout := time.After(5 * time.Second)
		scanLoop:
			for {
				select {
				case sig, ok := <-sigChan:
					if !ok {
						return fmt.Errorf(
							"D-Bus subscription closed while scanning for access points",
						)
					} else if sig.Path == nmDevice.GetPath() &&
						sig.Name == dBusPropertiesChanged &&
						len(sig.Body) >= 2 {

						/* Note: sig.Body should be
						[
							interface_name: string,
							changed_properties: map[string]dbus.Variant,
							invalidated_properties: []string
						]
						*/
						changed, ok := sig.Body[1].(map[string]dbus.Variant)
						if !ok {
							return fmt.Errorf(
								"D-Bus signal body had unexpected format",
							)
						}

						if _, ok := changed["LastScan"]; ok {
							break scanLoop
						}
					}
				case <-timeout:
					// On timeout, just exit loop and return APs that have
					// already been found
					break scanLoop
				}
			}
			c.nmObj.Unsubscribe()
		}

		// device.GetPropertyAccessPoints() can return dbus object paths
		// that become invalidated when `ResolveAccessPoint` tries to read from
		// them.  Rather than silently ignoring these errors and excluding
		// the access point from the slice, we instead just try calling the
		// function up to 3 times, assuming it will probably work on the 2nd
		// attempt.
	getPropertyAccessPoints:
		for i := 0; i < 3; i++ {
			var nmAPs []nm.AccessPoint
			nmAPs, err = nmDevice.GetPropertyAccessPoints()
			if err != nil {
				continue getPropertyAccessPoints
			}
			// Convert nm.AccessPoint to AccessPoint
			pts := make(data.Points, 0, len(nmAPs))
			for i, nmAP := range nmAPs {
				var ap AccessPoint
				ap, err = ResolveAccessPoint(nmAP)
				if err != nil {
					continue getPropertyAccessPoints
				}
				apJSON, err := ap.MarshallJSON()
				if err != nil {
					return fmt.Errorf("error encoding: %v", err)
				}
				pts.Add(data.Point{
					Type:   "accessPoints",
					Key:    strconv.Itoa(i),
					Text:   string(apJSON),
					Origin: c.config.ID,
				})
			}
			c.log.Printf("Discovered %v access points", len(pts))

			// Add tombstone points
			currAPLen := len(device.AccessPoints)
			for i := currAPLen - 1; i >= len(nmAPs); i-- {
				pts.Add(data.Point{
					Type:      "accessPoints",
					Key:       strconv.Itoa(i),
					Tombstone: 1,
					Origin:    c.config.ID,
				})
			}

			// Send points to device
			err := SendNodePoints(
				c.nc,
				device.ID,
				pts,
				false,
			)
			if err != nil {
				return fmt.Errorf(
					"error updating device %v: %w", device.ID, err,
				)
			}

			err = data.Decode(data.NodeEdgeChildren{
				NodeEdge: data.NodeEdge{
					ID:     device.ID,
					Parent: device.Parent,
					Points: pts,
				},
			}, &device)
			if err != nil {
				return fmt.Errorf("error decoding data: %w", err)
			}

			c.config.Devices[devIndex] = device
			break getPropertyAccessPoints
		}
		if err != nil {
			return fmt.Errorf("cannot get AccessPoints property from D-Bus")
		}
	}
	if !found {
		return fmt.Errorf("no Wi-Fi devices found")
	}
	return nil
}

// Stop stops the NetworkManager Client
func (c *NetworkManagerClient) Stop(error) {
	close(c.stopCh)
}

// Points is called when the client's node points are updated
func (c *NetworkManagerClient) Points(nodeID string, points []data.Point) {
	c.pointsCh <- NewPoints{
		ID:     nodeID,
		Points: points,
	}
}

// EdgePoints is called when the client's node edge points are updated
func (c *NetworkManagerClient) EdgePoints(
	_ string, _ string, _ []data.Point,
) {
	// Do nothing

	// c.edgePointsCh <- NewPoints{
	// 	ID:     nodeID,
	// 	Parent: parentID,
	// 	Points: points,
	// }
}
