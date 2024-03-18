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

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

/*
NetworkManagerClient is a SimpleIoT client that manages network interfaces
and their connections using NetworkManager via D-Bus.
Network connections and device states are synchronized between
NetworkManager and the SimpleIoT node tree.

	==========================      ======================== ---    device state     --> ==================
	| NetworkManager (D-Bus) | <--> | NetworkManagerClient |                             | SimpleIoT Tree |
	==========================      ======================== <-- connection settings --> ==================

The NetworkManagerClient only controls SimpleIoT "managed" connections within
NetworkManager. Although all connections will be added to the SIOT tree,
unmanaged NetworkManager connections will not be updated or deleted by
SimpleIoT.

[NetworkManager Reference Manual]: https://networkmanager.dev/docs/api/latest/
[gonetworkmanager Go Reference]: https://pkg.go.dev/github.com/Wifx/gonetworkmanager/v2
*/
type NetworkManagerClient struct {
	log        *log.Logger
	nc         *nats.Conn
	config     NetworkManager
	stopCh     chan struct{}
	pointsCh   chan NewPoints
	nmSettings nm.Settings       // initialized on Run()
	nmObj      nm.NetworkManager // initialized on Run()
	// deletedConns are managed connections previously deleted from the SIOT
	// tree; map is keyed by NetworkManager's connection UUID and initialized
	// on Run()
	deletedConns map[string]NetworkManagerConn
}

// NetworkManager client configuration
type NetworkManager struct {
	ID                      string                 `node:"id"`
	Parent                  string                 `node:"parent"`
	Description             string                 `point:"description"`
	Disabled                bool                   `point:"disabled"`
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
func logFirstError(method string, log *log.Logger, errors []error) {
	if len(errors) > 0 {
		plural := ""
		if len(errors) != 1 {
			plural = "s; the first is"
		}
		log.Printf(
			"%v had %v error%v: %v",
			method, len(errors), plural, errors[0],
		)
	}
}

// NewNetworkManagerClient returns a new NetworkManagerClient using its
// configuration read from the Client Manager
func NewNetworkManagerClient(nc *nats.Conn, config NetworkManager) Client {
	// TODO: Ensure only one NetworkManager client exists
	return &NetworkManagerClient{
		log: log.New(
			os.Stderr,
			"networkManager: ",
			log.LstdFlags|log.Lmsgprefix,
		),
		nc:       nc,
		config:   config,
		stopCh:   make(chan struct{}),
		pointsCh: make(chan NewPoints),
	}
}

// Run starts the NetworkManager Client. Restarts if `networkManager` nodes or
// their descendants are added / removed.
func (c *NetworkManagerClient) Run() error {
	str := "Starting NetworkManager client"
	if c.config.Disabled {
		str += " (currently disabled)"
	}
	c.log.Println(str)
	// c.log.Printf("config %+v", c.config)

	/*
		When starting this client, a few things will happen:

		1. We load all previously deleted connections from the SIOT tree for
			future sync operations.
		2. We compare the list of SIOT "managed" connections to the SIOT tree
			(the tree has already been loaded into `c.config.Connections`)
			and perform a one-way synchronization to NetworkManager by
			creating, updating, and deleting connections. Unmanaged connections
			are copied to the SIOT tree.
		3. Perform a one-way synchronization **from** NetworkManager for
			NetworkManagerDevices in the SIOT tree.
		4. Start polling NetworkManager and continue syncing.
	*/

	// Note: Writes to `doSync` channel causes a sync operation to occur as soon
	// as possible. Generally, calling `queueSync` is preferred to leverage the
	// syncDelayTimer and rate limit sync operations.
	var syncDelayTimer *time.Timer
	syncDelayTimerLock := &sync.Mutex{}
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

		// Get deleted previously managed connections in SIOT tree
		var allConnNodes []data.NodeEdge
		allConnNodes, err = GetNodes(
			c.nc, c.config.ID, "all", "networkManagerConn", true,
		)
		if err != nil {
			return fmt.Errorf("error getting deleted connection nodes: %w", err)
		}
		c.deletedConns = make(map[string]NetworkManagerConn)
		for _, ne := range allConnNodes {
			// Check tombstone to see if this node was deleted
			deleted := false
			for _, p := range ne.EdgePoints {
				if p.Type == data.PointTypeTombstone {
					deleted = int(p.Value)%2 == 1
					break
				}
			}
			if deleted {
				// Decode ne to NetworkManagerConn and add to deletedConns map
				var deletedConn NetworkManagerConn
				err = data.Decode(
					data.NodeEdgeChildren{NodeEdge: ne, Children: nil},
					&deletedConn,
				)
				if err != nil {
					return fmt.Errorf(
						"error decoding deleted connection node: %w", err,
					)
				}
				if deletedConn.Managed {
					c.deletedConns[deletedConn.ID] = deletedConn
				}
			}
		}

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
		if len(doSync) > 0 {
			return // sync already queued to run immediately
		}
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

	if !c.config.Disabled {
		err := init()
		if err != nil {
			return err
		}
	}

	// Flag to mute logging SyncConnections() errors when no connection nodes
	// have been updated
	muteSyncConnectionsError := false

loop:
	for {
		select {
		case <-c.stopCh:
			break loop
		case nodePoints := <-c.pointsCh:
			// c.log.Print(nodePoints)

			disabled := c.config.Disabled

			// Update config
			err := data.MergePoints(nodePoints.ID, nodePoints.Points, &c.config)
			if err != nil {
				log.Println("Error merging points:", err)
			}

			// Handle Disable flag
			if c.config.Disabled {
				if !disabled {
					cleanup()
				}
			} else if disabled {
				// Re-initialize
				err := init()
				if err != nil {
					return err
				}
			} else {
				// If this is a connection node point, unmute SyncConnections()
				// errors.
				for _, conn := range c.config.Connections {
					if conn.ID == nodePoints.ID {
						muteSyncConnectionsError = false
						break
					}
				}
				// Queue sync operation
				queueSync()
			}
		case <-doSync:
			// Perform sync operations; abort on fatal error
			c.log.Println("Syncing with NetworkManager over D-Bus")
			errs, fatalErr := c.SyncConnections()
			// Abort on fatal error
			if fatalErr != nil {
				return fmt.Errorf("connection sync error: %w", fatalErr)
			}
			if !muteSyncConnectionsError {
				logFirstError("SyncConnections", c.log, errs)
				muteSyncConnectionsError = true
			}

			// Synchronize devices with NetworkManager
			errs, fatalErr = c.SyncDevices()
			// Abort on fatal error
			if fatalErr != nil {
				return fmt.Errorf("device sync error: %w", fatalErr)
			}
			logFirstError("SyncDevices", c.log, errs)

			// Synchronize hostname
			errs, fatalErr = c.SyncHostname()
			// Abort on fatal error
			if fatalErr != nil {
				return fmt.Errorf("hostname sync error: %w", fatalErr)
			}
			logFirstError("SyncHostname", c.log, errs)
		case <-syncTick.C:
			muteSyncConnectionsError = false
			// Queue sync operation
			queueSync()
		case sig, ok := <-dbusSub:
			if !ok {
				// D-Bus subscription closed
				dbusSub = nil
				break // select
			}
			if sig.Name == dBusPropertiesChanged ||
				// TODO: Confirm these strings are sufficient
				strings.HasPrefix(sig.Name, "org.freedesktop.NetworkManager.Device") ||
				strings.HasPrefix(sig.Name, "org.freedesktop.NetworkManager.Connection") ||
				strings.HasPrefix(sig.Name, "org.freedesktop.NetworkManager.Settings.Connection") {
				queueSync()
			} else {
				c.log.Printf("not triggering sync %v for %+v", sig.Name, sig)
			}
		}

		// Scan Wi-Fi networks if needed
		if !c.config.Disabled && c.config.RequestWiFiScan {
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
			c.config.RequestWiFiScan = false
			err = SendNodePoint(c.nc, c.config.ID, p, true)
			// Log error only
			if err != nil {
				c.log.Printf("Error clearing requestWiFiScan: %v", err)
			}
		}
	}
	cleanup()
	return nil
}

// Helper function to emit point for connection error and update
// the NetworkManagerConn.Error field
func (c *NetworkManagerClient) emitConnectionError(
	conn *NetworkManagerConn, err error,
) error {
	if err == nil {
		conn.Error = ""
	} else {
		conn.Error = err.Error()
	}
	emitErr := SendNodePoint(c.nc, conn.ID, data.Point{
		Type:   "error",
		Text:   conn.Error,
		Origin: c.config.ID,
	}, true)
	if emitErr != nil {
		return fmt.Errorf(
			"error emitting error for connection %v: %w", conn.ID, err,
		)
	}
	return nil
}

// SyncConnections performs a one-way synchronization of the NetworkManagerConn
// nodes in the SIOT tree with connections in NetworkManager via D-Bus. The
// sync direction is determined by the connection's Managed flag. If set, the
// connection in NetworkManager is updated with the data in the SIOT tree;
// otherwise, the SIOT tree is updated with the data in NetworkManager.
// Returns a list of errors in the order in which they are encountered. If a
// fatal error occurs that aborts the sync operation, that will be included
// in `errs` and returned as `fatal`.
func (c *NetworkManagerClient) SyncConnections() (errs []error, fatal error) {
	// Build set of connection IDs from SIOT tree
	treeConnIDs := make(map[string]struct{}, len(c.config.Connections))
	for _, treeConn := range c.config.Connections {
		treeConnIDs[treeConn.ID] = struct{}{} // add to set
	}

	// Get NetworkManagerConn from NetworkManager via D-Bus
	connections, err := c.nmSettings.ListConnections()
	if err != nil {
		fatal = fmt.Errorf("error listing connections: %w", err)
		errs = append(errs, fatal)
		return
	}

	// Build map of NetworkManager connections by ID and handle connections not
	// found in the SIOT tree.
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

		// Handle connections not found in the SIOT tree. If a connection is not
		// in the SIOT tree, we check to see if a previously deleted, managed
		// connection with the same UUID existed in the SIOT tree. If so, we
		// delete it from NetworkManager; otherwise, it must be a new connection
		// to be added to the SIOT tree.
		if _, ok := treeConnIDs[nmc.ID]; !ok {
			// Note: deletedConns is keyed by UUID, not ID
			if _, ok := c.deletedConns[nmc.ID]; ok {
				// Delete connection from NetworkManager
				if err := conn.Delete(); err != nil {
					errs = append(errs, fmt.Errorf(
						"error deleting connection %v: %w", nmc.ID, err,
					))
				}
				c.log.Printf("Deleted connection %v (%v)",
					nmc.ID, nmc.Description,
				)
			} else {
				// Add connection to SIOT tree
				c.log.Printf("Detected connection %v (%v)",
					nmc.ID, nmc.Description,
				)
				err := SendNodeType(c.nc, nmc, c.config.ID)
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"error adding connection node %v: %w", nmc.ID, err,
					))
				}
			}
		} else {
			// Add NetworkManager connection to the map and handle it in the
			// next loop
			nmConns[nmc.ID] = ConnectionResolved{conn, nmc}
		}
	}

	// Now handle each connection already in the SIOT tree
	for i := range c.config.Connections {
		treeConn := &c.config.Connections[i]
		var pts data.Points // points to update this connection in SIOT tree
		nmc, found := nmConns[treeConn.ID]
		if found {
			// Connection also exists in NetworkManager
			if treeConn.Managed {
				// Update connection in NetworkManager, except LastActivated
				if treeConn.LastActivated != nmc.Resolved.LastActivated {
					treeConn.LastActivated = nmc.Resolved.LastActivated
					pts.Add(data.Point{
						Type:   "lastActivated",
						Value:  float64(treeConn.LastActivated),
						Origin: c.config.ID,
					})
				}

				// Sync properties not populated by ResolveNetworkManagerConn
				nmc.Resolved.Managed = true
				if nmc.Resolved.Type == "802-11-wireless" &&
					nmc.Resolved.WiFiConfig.KeyManagement == "wpa-psk" {
					secrets, err := nmc.Connection.GetSecrets(
						"802-11-wireless-security",
					)
					if err != nil {
						// Wrap error, append to errs, and emit on connection
						err = fmt.Errorf(
							"error getting secrets for connection %v: %w",
							nmc.Resolved.ID, err,
						)
						errs = append(errs, err)
						err = c.emitConnectionError(treeConn, err)
						if err != nil {
							errs = append(errs, err)
						}
						continue
					}
					if psk, ok := secrets["802-11-wireless-security"]["psk"].(string); ok {
						nmc.Resolved.WiFiConfig.PSK = psk
					}
				}
				// Update existing connection
				if !treeConn.Equal(nmc.Resolved) {
					// diff, err := data.DiffPoints(nmc.Resolved, treeConn)
					// c.log.Printf("DEBUG: %v %v", err, diff)

					err = nmc.Connection.Update(treeConn.DBus())
					if err != nil {
						err = fmt.Errorf(
							"error updating connection %v: %w", treeConn.ID, err,
						)
						errs = append(errs, err)
						err = c.emitConnectionError(treeConn, err)
						if err != nil {
							errs = append(errs, err)
						}
						// Delete connection because update failed
						err = nmc.Connection.Delete()
						if err != nil {
							errs = append(errs, fmt.Errorf(
								"error deleting connection %v: %w",
								treeConn.ID, err,
							))
						}
						continue
					}
					c.log.Printf("Updated connection %v (%v)",
						treeConn.ID, treeConn.Description,
					)
					// If this connection is currently active, reactivate it
					acs, err := c.nmObj.GetPropertyActiveConnections()
					if err != nil {
						errs = append(errs, fmt.Errorf(
							"error getting active connections: %w", err,
						))
						continue
					}
					for _, ac := range acs {
						acID, err := ac.GetPropertyUUID()
						if err != nil {
							errs = append(errs, fmt.Errorf(
								"error getting active connection UUID: %w", err,
							))
							break
						}
						if acID == treeConn.ID {
							// Reactivate connection
							err = c.nmObj.DeactivateConnection(ac)
							if err != nil {
								err = fmt.Errorf(
									"error deactivating connection %v: %w",
									treeConn.ID, err,
								)
								errs = append(errs, err)
								err = c.emitConnectionError(treeConn, err)
								if err != nil {
									errs = append(errs, err)
								}
							}
							// Note: Device not specified
							_, err = c.nmObj.ActivateConnection(
								nmc.Connection, nil, nil,
							)
							if err != nil {
								err = fmt.Errorf(
									"error activating connection %v: %w",
									treeConn.ID, err,
								)
								errs = append(errs, err)
								err = c.emitConnectionError(treeConn, err)
								if err != nil {
									errs = append(errs, err)
								}
							}

							// Reapply connection settings to all devices
							// devs, err := ac.GetPropertyDevices()
							// if err != nil {
							// 	errs = append(errs, fmt.Errorf(
							// 		"error reactivating %v: %w",
							// 		treeConn.ID, err,
							// 	))
							// 	break
							// }
							// for _, dev := range devs {
							// 	err = dev.Reapply(treeConn.DBus(), 0, 0)
							// 	if err != nil {
							// 		c.log.Printf("warning: could not reapply connection %v for device %v: %v",
							// 			treeConn.ID, dev.GetPath(), err,
							// 		)
							// 	} else {
							// 		c.log.Printf("Reapplied connection %v to device %v",
							// 			treeConn.ID, dev.GetPath(),
							// 		)
							// 	}
							// }
							// break
						}
					}
				}
			} else {
				// Update connection in SIOT tree
				diffPts, err := data.DiffPoints(treeConn, &nmc.Resolved)
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"error updating connection node %v: %w", treeConn.ID, err,
					))
					continue
				}
				if diffPts.Len() > 0 {
					c.log.Printf("Updating connection node %v (%v)",
						treeConn.ID, treeConn.Description,
					)
					// c.log.Println("DEBUG", diffPts)
					pts = append(pts, diffPts...)
				}
			}
		} else {
			// Connection does not exist in NetworkManager
			if treeConn.Managed {
				// Handle case where node ID is not a valid UUID
				_, err = uuid.Parse(treeConn.ID)
				if err != nil {
					err = fmt.Errorf("invalid UUID: %w", err)
				} else {
					// Add connection to NetworkManager
					_, err = c.nmSettings.AddConnection(treeConn.DBus())
				}
				if err != nil {
					err = fmt.Errorf(
						"error adding connection %v: %w", treeConn.ID, err,
					)
					errs = append(errs, err)
					err = c.emitConnectionError(treeConn, err)
					if err != nil {
						errs = append(errs, err)
					}
					continue
				}
				c.log.Printf("Added connection %v (%v)",
					treeConn.ID, treeConn.Description,
				)
			} else {
				// Delete connection from SIOT tree
				err = SendEdgePoint(c.nc, treeConn.ID, treeConn.Parent, data.Point{
					Type:  data.PointTypeTombstone,
					Value: 1,
				}, true)
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"error removing connection node %v: %w", treeConn.ID, err,
					))
					continue
				}
				c.log.Printf("Deleted connection node %v (%v)",
					treeConn.ID, treeConn.Description,
				)
			}
		}

		// Update points in SIOT tree, if needed
		if pts.Len() > 0 {
			// Set origin on all points
			for _, p := range pts {
				p.Origin = c.config.ID
			}
			err = SendNodePoints(c.nc, treeConn.ID, pts, true)
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
	for i := range c.config.Devices {
		device := &c.config.Devices[i]
		nmDevice, ok := deviceInfo[device.ID]
		if ok {
			// Preserve AccessPoints
			nmDevice.AccessPoints = device.AccessPoints
			// Update device
			pts, err := data.DiffPoints(device, &nmDevice)
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
				}, device)

				if err != nil {
					errs = append(errs, fmt.Errorf(
						"error decoding data %v: %w", device.ID, err,
					))
				}
			}
			// Delete from deviceInfo to avoid duplicating it later
			delete(deviceInfo, device.ID)
		} else {
			// Delete device
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
	// Note: updated devices are deleted from deviceInfo above
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
		if err != nil {
			errs = append(errs, err)
		}
	} else if hostname != c.config.Hostname {
		// Write hostname to NetworkManager
		err = c.nmSettings.SaveHostname(c.config.Hostname)
		if err != nil {
			errs = append(errs, err)
		}
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
	for devIndex := range c.config.Devices {
		device := &c.config.Devices[devIndex]
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
			}, device)
			if err != nil {
				return fmt.Errorf("error decoding data: %w", err)
			}

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
	// c.edgePointsCh <- NewPoints{
	// 	ID:     nodeID,
	// 	Parent: parentID,
	// 	Points: points,
	// }
}
