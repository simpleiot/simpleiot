# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For more details or to discuss releases, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## [Unreleased]

## Next

- sync: rewrite upstream sync on boundary-origin stream replication (ADR-7
  Stage 3): durable-consumer replication in both directions with resumable
  offline catch-up, adoption announcement on first connect, and no hash
  tree walks
- **breaking:** deleting a device node on the upstream now detaches it —
  the device no longer re-creates itself; only the upstream can restore it
- store: consume replica streams from other instances, merge them into the
  caches with the ADR-7 tip rule, and re-broadcast changed tips locally
  with a `Siot-Origin` header; remote-origin data is never written to
  local origin streams, so sync cannot echo
- db: the InfluxDB client now reads points from the store streams with
  durable consumers instead of the `up.>` wire subjects, so external
  history is gap-free across sync outages, instance restarts, and the
  client's own downtime (high-rate points still arrive over `phrup.>`)
- test: the db client test starts a throwaway VictoriaMetrics instance
  itself (skipped when the binary is not installed), replacing the
  external-InfluxDB requirement, and verifies the point path end to end
- store: add boundary resolution to the edge cache (`IsBoundary`,
  `OwningBoundary`) for the ADR-7 boundary-origin stream layout
- store: reply once and stop processing when an edge point DB write fails
- store: validate that new edges carry a `nodeType` point before writing to
  JetStream, so the stream and edge cache cannot diverge
- store: move to boundary-origin streams (`inst-<boundaryID>-<originID>`):
  streams are created per sync/AuthZ boundary rather than per node, subjects
  carry both routing tokens, and current state is the merge of subject tips
  with a deterministic origin tie-break (ADR-7 Stage 2 revision)
- store: pre-populate the point cache at startup and load on cache miss,
  fixing config points being lost after the first write following a restart
- store: migrate a node's subjects (and its owned subtree) to the new
  boundary stream when a move or undelete changes its owning boundary,
  preserving original point timestamps
- add `--storeMaxMsgsPerSubject` / `SIOT_STORE_MAX_MSGS_PER_SUBJECT` to bound
  per-subject history in store streams (current state always preserved)
- store: per-subject retention now defaults to 5000 messages (about a month
  of 10-minute data; configuration subjects are effectively unlimited);
  pass `-1` for unlimited. Each instance applies its own retention to
  replica streams it discovers, so hub and device retain independently
- add `--storeSyncInterval` / `SIOT_STORE_SYNC_INTERVAL` to tune the
  JetStream fsync interval (`always` fsyncs every write) for power-loss
  durability on edge devices
- replace SQLite store with JetStream per-node streams (ADR-7 Stage 2)
- remove SQLite dependency and hash tree
- add in-memory point cache and edge cache for fast lookups
- update NATS dependencies (nats.go v1.49.0, nats-server v2.12.5)
- **breaking:** existing SQLite databases must be migrated via
  `siot export`/`siot import`
- replace Point `Value`/`Text` fields with unified `DataType`/`Data` encoding
  (ADR-7 Stage 1) (#742)
- replace protobuf point encoding with compact binary format (#742)
- move edge point NATS subjects to `ep.` prefix and add type/key to node point
  subjects (#742)
- add `dataType` field to Elm frontend Point type (#742)
- add `MarshalYAML` for human-readable point export (#742)
- add OOM protection to `DecodePoints` (#742)
- fix Shelly mDNS data race by creating fresh params per scan (#742)
- fix JSON/YAML unmarshal when `dataType` is set but `data` is empty (#742)
- update docs to reflect new point encoding and NATS subjects (#742)
- import: a node created from a file now keeps a single `nodeType` edge point.
  A file carries the node type, and sending a node added a second one, so an
  export of an imported tree did not match the file it came from.
- an edge now holds one point per type and key. The store appended a repeated
  point in the same message rather than replacing what it had for it.
- fix a data race between a node watcher and its caller when a node has a keyed
  point that decodes into a map. Merging an update wrote into the map the
  caller was already holding, so decoding now replaces the map with a copy.

## [0.22.0] - 2026-08-04

- config files: one YAML format now describes a tree of nodes, and
  `siot export`, `siot import`, and provisioning all use it. The node type is
  the key and each point type is a key of its own, so a file reads as
  configuration rather than as a serialized structure:

  ```yaml
  nodes:
    - group:
        description: Sensors
        children:
          - modbus:
              description: Modbus sensors
              port: /dev/ttyS1
              baud: 9600
  ```

  How a value is written decides what it becomes: `9600` is numeric, `"9600"` is
  text, `1` and `1.5` are an integer and a float, a mapping is a set of keyed
  points, and a sequence is an array. Files in the previous format are not read;
  re-export the tree to get one in the current format. See
  [user/configuration](docs/user/configuration.md).

- config files: a file describes configuration and nothing else. It carries no
  node IDs, no origins, and no points without a value. A `nodeID` point names
  the node it refers to by description, so a rule can point at a variable
  without either of them knowing its UUID, in a file or across files.

- import: applying a file is now idempotent. Nodes are matched by description
  rather than by ID, so importing a file creates what is missing, sends only the
  points that differ, and does nothing when the tree already agrees. Running the
  same import twice does what running it once did. A `delete` list removes
  nodes.

  A description is how a file finds a node, so renaming one in the UI detaches
  it from the file that describes it and the next apply creates a second node
  beside it. Rename in two steps: delete the old description in the same file
  that introduces the new one.

- import: `-parentID` and `-preserveIDs` are gone. A file says where its nodes
  attach with `parent`, naming a node by description, and every node a file
  creates gets a fresh ID. `-dryRun` prints what a file would do without
  applying it. The `(import)` suffix that was appended to descriptions is gone,
  since it would have broken idempotence.

- import: importing at the root no longer replaces the root node. A file never
  describes the root, which is this instance rather than configuration, so the
  restart that an import at `root` used to require is gone along with the root
  watcher that performed it. `siot serve` still exits non-zero on error, so
  supervisors that only restart on failure will bring it back up.

- export: an export is now a provisioning file. The root node is left out, as
  are node IDs, valueless points, and origins. An export refuses to write a tree
  whose siblings share a description, since no file could say which node it
  meant.

- provisioning: an instance can configure itself from files. A directory given
  by `-provisioningDir` or `SIOT_PROVISIONING_DIR`, defaulting to
  `<SIOT_DATA>/provisioning` when it exists, is applied at start-up and whenever
  a file changes, so a unit built from an image comes up configured with no
  import step. `SIOT_PROVISIONING_INTERVAL` sets how often to look for changes
  the watch might have missed.

  Files can also be uploaded through the UI as `file` nodes under the
  provisioning node, which is how a unit whose filesystem you cannot reach gets
  configured. Files on disk apply first and uploads layer on top, oldest first.

  A `provisioning` node under the root records what was applied, with a checksum
  per file and the last error if one failed. Removing a file removes its status;
  the nodes it created stay, since provisioning describes what should exist
  rather than owning what it made.

- provisioning: `siot provision -dir <dir>` prints what a directory of files
  would do to a running instance without applying it, and `-check` only parses
  them, which needs no instance and is what a build can use to fail on a bad
  file.

## [0.21.0] - 2026-08-03

- modbus: move Modbus to the client architecture. Modbus buses are now started
  and stopped by the client manager like every other client, rather than by a
  20-second poll of the store, so a new bus or IO starts collecting data right
  away instead of up to 20 seconds later. Node and point types are unchanged, so
  existing configurations keep working.
- modbus, 1-wire: start a bus that lives inside a group. Discovery looked only
  at the direct children of the root node, so a bus placed anywhere else never
  ran.
- modbus: adding or removing an IO now restarts its bus, which reopens the port.
  A Modbus server drops the TCP connections it holds when this happens. This
  applies when a person edits the configuration, not during normal polling.
- modbus: read `int16` values as signed. An `int16` input or holding register
  was decoded as an unsigned value, so a negative reading appeared as a large
  positive number on both the client and the server.
- modbus: write a `valueSet` to the device as soon as it arrives. The check that
  decided whether a value could be written compared the data format against the
  IO type, which never matched, so a write waited for the next poll.
- modbus: treat a scale of zero as one. An IO created without a scale read every
  register as zero; the old code declined to start such an IO at all, which was
  just as hard to diagnose.
- modbus: publish the corrected response timeout when a bus has none configured
  or has a non-positive one, so what the bus uses is what the configuration
  shows.
- modbus: closing a Modbus TCP server no longer waits on a connection listener
  that has already exited, which could happen when the far end dropped the
  connection at the moment the server was being closed.
- modbus: the first tests for the Modbus subsystem, covering every register type
  and data format end to end through a server bus and a client bus talking over
  a TCP socket.
- 1-wire: **1-Wire bus nodes are no longer created automatically.** Add a 1-Wire
  node where you want it and set its `Index` to the bus controller number, which
  matches the `w1_bus_master<index>` directory in `/sys/bus/w1/devices`. The
  sensors on that bus are still detected and added for you. Existing bus nodes
  keep working unchanged; this affects new setups, and any install that relied
  on a deleted bus node reappearing.
- 1-wire: move 1-Wire to the client architecture. A new bus or sensor is picked
  up as soon as it is configured rather than on the next 20-second scan. This is
  also the first test coverage the subsystem has had.
- 1-wire: detect sensors below the configured bus controller rather than across
  all of them. With two controllers, every bus node claimed every sensor.
- server: retire the `node` package. With Modbus and 1-Wire moved to clients,
  all that remained of the node manager was writing the app and OS versions to
  the root node, which is now a small function that runs once after the store
  starts. The 20-second scan that drove the old bus managers is gone.
- import: restart automatically after importing with `-parentID=root`. Such an
  import replaces the root node, which leaves everything that resolved a node
  relative to the old root working against a tree that no longer exists, so the
  instance had to be restarted by hand before it resumed collecting data. The
  server now exits once the import settles and expects the service manager to
  start it again; the service file installed by `siot install` sets
  `Restart=always` for this reason. If you run Simple IoT some other way, please
  make sure the process is restarted. `siot serve` also exits non-zero on error
  now, so supervisors that only restart on failure will bring it back up.
- client manager: refresh the root node on every scan. The manager resolved the
  root once at startup, so it kept scanning below a root that an import had
  replaced and never started clients for the imported nodes. This also covers
  instances that are not configured to restart.
- client manager: stop clients when the last node of a type is deleted. The scan
  returned early when it found no nodes, which skipped the step that shuts down
  clients whose nodes are gone.

## [0.20.6] - 2026-08-01

- metrics client: collect fan speed and drive level from the hwmon interface,
  reported as `metricSysFanSpeed` in RPM and `metricSysFanPWM` on the 0-255
  scale the kernel uses. Drivers that report a plain `rpm` file rather than the
  usual `fan1_input`, such as the Tegra tachometer, are read as well.
- metrics client: collect the state of each thermal cooling device as
  `metricSysCoolingState`, keyed by device type. A state above zero means the
  thermal governor is limiting the system, so a rising `cpufreq` or `devfreq`
  state shows performance being given up to stay cool, which temperature alone
  does not reveal. The scale each device is measured against is published once
  at startup as `metricSysCoolingStateMax`.
- metrics client: collect rail voltage, current, and power from the hwmon power
  monitors, published as the existing `voltage`, `current`, and `power` types in
  volts, amps, and watts. A channel is published when its driver labels it,
  which is how a board names the rail a channel measures, so a Jetson reports
  `VDD_GPU_SOC` and friends by name. Monitors that do not report power directly,
  including the INA3221, still give voltage and current, and the product stands
  in for the missing reading.
- metrics client: collect the current clock of each CPU as `metricSysCPUFreq`,
  in MHz and keyed by `cpu0`, `cpu1`, and so on. Read alongside the cooling
  device states, it shows where the clocks settled once the thermal governor
  pulled them back.
- documentation: describe the thermal, power, and clock readings the system
  metrics collect, in [metrics](docs/user/metrics.md)

## [0.20.5] - 2026-08-01

- metrics client: read the Linux thermal zones directly so SoC temperatures are
  collected. The sensor library only consults the zones on systems that have no
  hwmon temperature inputs, so boards that expose both, such as the Jetson AGX
  Orin, reported only their board sensors while the CPU, SoC, and junction
  readings went uncollected. Zones whose rail is powered down are skipped
  individually, sensors that fail to read no longer discard the ones that
  succeeded, and repeated sensor names are numbered (`tmp451`, `tmp451_2`) so
  two readings no longer overwrite each other.

## [0.20.4] - 2026-08-01

- serial node UI: keep the remaining configuration fields out of the point and
  value table, so it shows the data the MCU reports rather than the node's own
  settings. `protocol`, `timeout`, `logConsole`, `syncParent`, `download`, and
  `progress` were listed there alongside MCU points. `protocol` and `timeout`
  are Modbus settings as well, so that node is tidier too.

## [0.20.3] - 2026-08-01

- serial client: exit the port reader when the port is closed. It previously
  fell through to its retry path, and because a closed port fails every read
  immediately, the goroutine spun at full speed for the life of the process. One
  was left behind on every close, including each disable/enable of a serial
  node.
- serial client: make the connected state self-correcting. It was published only
  when it changed, so a single update that was lost or overwritten left a node
  reported as not connected even while data kept arriving. The state is now
  recorded locally only after a successful publish, and in shell mode the
  watchdog republishes it on a slow cadence so the reported state converges.

## [0.20.2] - 2026-08-01

- update frontend assets

## [0.20.1] - 2026-07-31

- serial client: don't display log field in UI when in shell mode

## [0.20.0] - 2026-07-31

- serial: add a Zephyr shell protocol mode to the MCU serial client. Points are
  exchanged as lines of ASCII with an MCU's console shell rather than
  COBS-framed binary packets, so the link a developer reads is the link Simple
  IoT uses for data. Select it with the Protocol setting on a serial node; an
  empty value means binary, so existing nodes are unaffected. See the
  [MCU documentation](docs/user/mcu.md) and the
  [serial reference](docs/ref/serial.md).
- serial: add a "Log console output" option that mirrors the MCU console to the
  Simple IoT server log. Shell protocol only.

## [0.19.0] - 2026-07-31

- add GPS client that reads position data from a serial NMEA receiver, the gpsd
  daemon, or an internal simulator, and publishes latitude, longitude, altitude,
  speed, heading, fix status, and satellite information. See the
  [GPS documentation](docs/user/gps.md), which includes instructions for
  plotting a track on a Grafana geomap.
- the simulated GPS source now continues from the node's last published
  position, so a configuration change or an application restart picks the track
  up where it left off. A `Reset location` button, or editing the start
  latitude, longitude, or heading, moves the track back to the start.
- the simulated GPS source now logs the points it generates at debug level 4,
  matching the raw data the serial and gpsd sources log at that level. The
  [GPS documentation](docs/user/gps.md#debug-levels) now describes what each
  debug level covers.
- changing the GPS debug level now takes effect right away. The level was read
  once when a source started, so a change was ignored until something else
  restarted the source.
- the GPS `Rx count` and `Error count` reset checkboxes now clear after the
  counter is zeroed. They stayed set, so a reset only took effect on every
  second click.
- document what storing text points in Victoria Metrics actually does; the value
  is converted to 0 rather than the line being rejected
- (BREAKING) remove the legacy `gps` package, which predated the client
  architecture and is superseded by the GPS client. `data.GpsPos` is unchanged
  and still used by the modem code.
- add `siot update`, which updates Simple IoT to the latest release published on
  GitHub. It downloads the release for the platform it is running on, verifies
  it against the published checksums, and replaces the binary in place.
  `siot update -check` reports what is available without installing it. See the
  [installation documentation](docs/user/installation.md#updating).
- releases now publish the `siot` executable directly instead of wrapping it in
  a `.tar.gz` or `.zip` archive, so downloading and running it takes one less
  step. Asset names are unchanged apart from the archive extension, and Windows
  binaries now end in `.exe`.
- releases are now built and published by a GitHub Actions workflow when a
  version tag is pushed, and the release notes come from the matching
  `CHANGELOG.md` section

## [[0.18.5] - 2025-09-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.5)

- add configurable response timeout parameter for Modbus clients with 100ms
  default

## [[0.18.4] - 2025-08-04](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.4)

- don't return power information for Shelly devices that don't have power
  measurement (like Plus 1)
- support status for Shelly gen1 relays
- update Go packages to
  [latest versions](https://github.com/simpleiot/simpleiot/pull/752/commits/6f6a00ed4b27d0809ef28e96216631a2e8da9559)
- fix bug with importing a list of nodes
- add browser client allowing control and configuration of Yoe Kiosk Browser

## [[0.18.3] - 2025-03-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.3)

- add favicon to frontend so icon displays in browser tabs (#756)

## [[0.18.2] - 2025-03-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.2)

- fix bug where export/import displayed no nodes (#749). Fixed in (#753)

## [[0.18.1] - 2024-11-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.1)

- fix bug where Shelly IO Enable Control option was not working (#739)

## [[0.18.0] - 2024-11-07](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.0)

- change default login to `admin`/`admin` (used to be `admin@admin.com`, but
  there was is reason to have a bogus email address). (#730)
- file client/node
  - option to store binary files
  - display filename, file size, and stored size
  - create file client backend code that runs for file nodes
  - calculate and populate md5sum when file contents change
  - display md5sum in file node UI
- serial client/node
  - add serial file download -- can be used for MCU updates
  - fix issues with populate node ID for high rate data
- db client
  - fix crash if node ID is not populated correctly in data
- client (BREAKING API CHANGE)
  - renamed client.Group -> client.RunGroup. This is so we don't conflict with
    the client that manages group nodes
- fix issue with user login when moving users to different groups (#713)

## [[0.17.0] - 2024-08-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.17.0)

- add rule/condition/action disable flag (#352)
- rule action: add point key field (#714)

## [[0.16.2] - 2024-06-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.16.2)

- db client: Improve Influx history query functionality
  - If history query response fails, try responding again with ErrorMessage
  - TagFilters values can now be empty string or a slice of strings

## [[0.16.1] - 2024-05-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.16.1)

- Modbus API: add an option to validate the input when a client writes to a
  register.
- Update client:
  - improve autodownload logic
  - check for updates when URI is changed
  - improve error handling and reporting
  - fix bug when reducing update list
- expand documentation on
  [creating a client](https://docs.simpleiot.org/docs/ref/client.html#creating-new-clients).

## [[0.16.0] - 2024-05-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.16.0)

- add Update client -- currently supports system updates
  [docs](https://docs.simpleiot.org/docs/user/update.html).
- update elm-tooling
- api: Added `history.<nodeId>` NATS endpoint to send Influx history queries to
  an Influx DB client node.

## [[0.15.3] - 2024-03-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.15.3)

- UI: add tag UI to metrics client UI

## [[0.15.0] - 2024-03-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.15.0)

- NTP client: Do not set configuration if servers are not specified. This allows
  timesyncd to use the default configuration if no servers are specified.
- server: Args now accepts a `*FlagSet` to allow flags to be extended
- Influx client when writing points from a given node also adds additional tags
  based on the node that emitted the point. Previously, `nodeID` tag was added,
  but this has been renamed to `node.id`. Also added is `node.type` and
  `node.description` (populated with the value of a point of type
  "description").
- For each Influx DB client, the user can specify an array of tag point types
  (via point type "tagPointType"). These point types are also copied as tags for
  each point emitted by the node. For example, if node A has two points tag:city
  (i.e. Point.Type is "tag" and Point.Key is "city") and tag:state, then these
  point values are appended to every single point emitted by node A. In Influx,
  each point would have a tag `node.tag.city` and `node.tag.state` with their
  respective Point.Text values.
- BREAKING CHANGE: Influx DB tag `nodeID` is now `node.id`
- update frontend dependencies and fix various build issues
- UI: add tag UI most clients so that custom tags can be added to each node.

## [[0.14.10] - 2024-02-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.10)

- store: Improved performance when loading many nodes and edges
- serial: Fixed bug: do not write points over closed serial port

## [[0.14.9] - 2024-01-18](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.9)

- require custom UI assets to be rooted and not be public directory
- add `-UIAssetsDebug` cmdline flag. This will dump all the UI assets file and
  is useful in debugging to make sure your assets files are correct -- it can
  get a little tricky with embedding, etc.

## [[0.14.8] - 2024-01-16](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.8)

- support passing in a custom UI (fs.FS or directory name) to the SIOT server.
  This allows you to replace the SIOT UI with a custom version.

## [[0.14.7] - 2024-01-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.7)

- add modbus swap words for Int32/Uint32 writes

## [[0.14.6] - 2024-01-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.6)

- verb -> adjective changes in several types. This is more consistent and
  accurate with how things are done in this industry (HTML, etc). This is a
  breaking change in that nodes with disable or control flag set will need to be
  reconfigured.
  - disable -> disabled
  - control -> controlled
- add modbus Float32ToRegsSwapWords()

## [[0.14.5] - 2024-01-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.5)

- simpleiot-js: Fixed bugs and improved README
- Replace deprecated `io/ioutil` functions (#680)
- fixed frontend bug where only custom node types could be added

## [[0.14.4] - 2023-12-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.4)

- UI: in node raw view, you can now edit/add/delete points (#676)
- UI: add custom node types

## [[0.14.3] - 2023-12-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.3)

- UI: display unknown nodes as raw type and points
- UI: add raw view button to node expanded view. This allows us to view the raw
  points in any node which is useful for debugging and development. (see
  [docs](https://docs.simpleiot.org/docs/user/ui.html#raw-node-view) for more
  information)

## [[0.14.2] - 2023-11-29](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.2)

- Signal generator client: replaced "Sync Parent" option with "Destination" to
  indicate the destination node and point type for generated points
- update gonetworkmanager to v2.1.0 and fix sync bugs
- network-manager client: Now supports better connection sync via connection
  `Managed` flag; fixed a few bugs; WiFiConfig sync now works

## [[0.14.1] - 2023-11-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.1)

- update frontend assets (missed that in v0.14.0)

## [[0.14.0] - 2023-11-14](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.0)

- update to nats-server to v2.10.4
- update to nats client package to v1.31.0
- development: `envsetup.sh` sources `local.sh` if it exists
- Go client API for export/import nodes to/from YAML
- `siot` CLI export and import commands
- simpleiot-js improvements
- Network Manager Client (WIP)
- NTP Client
- serial client: allow configuration of HR point destination
- serial client: add "Sync Parent" option
- Signal generator client: add support for square, triangle, and random walk
  patterns
- fix issue with batched points of the same type/key (#658)

## [[0.13.1] - 2023-10-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.13.1)

- update client manager API to include list of parent node types
- fix issue with duplicating nodes where there were two copies of Description
  points
- display decode error count for high-rate serial packets
- display rate for high-rate serial packets

## [[0.13.0] - 2023-09-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.13.0)

- implement `siot install` command (#527)
- update frontend poll rate from 3s to 4s
- fix `siot store` (was crashing due to Opened not being defined)

## [[0.12.7] - 2023-09-14](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.7)

- serial client decoding improvements

## [[0.12.6] - 2023-09-13](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.6)

- fix issue with email in user node UI (#609)

## [[0.12.5] - 2023-08-25](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.5)

- add supported for Linux temp sensors (#607)

## [[0.12.4] - 2023-08-25](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.4)

- Reworked and simplified decode and merge routines (#589). See
  [documentation](https://docs.simpleiot.org/docs/ref/data.html?#converting-nodes-to-other-data-structures)
- UI: fixed issue with with paste node rule condition/actions (#600)
- Can client: fixed various issues (#498)
- Rule client: fix issue with error reporting (#599)
- switch to forked mdns package to get rid of closing messages (#558)
- update nats.go package from v1.20.0 => v1.28.0
- update nats-server package from v2.9.6 => v2.9.21
- default NATS server to 127.0.0.1 instead of localhost

## [[0.12.3] - 2023-08-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.3)

- switch to elm-tooling to enable building on Linux and MacOS ARM machines
- enable riscv builds in release

## [[0.12.2] - 2023-08-01](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.2)

- fix login

## [[0.12.1] - 2023-07-27](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.1)

- fix control of Shelly lights
- required that `Point:Key` field always be set (#580)
- improvements in point decode and merge with arrays (not finished)

## [[0.12.0] - 2023-07-21](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.0)

- support Dates in Rule schedule conditions
- Rules are re-run if any rule configuration changes
- Display error conditions in Rule nodes
- hide schedule weekday entry when dates are active
- hide schedule date entry when weekdays are active
- support deleting (tombstone points) in NodeDecode and NodeMerge functions

## [[0.11.4] - 2023-06-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.4)

- remove index field from Point data structure. See #565
- add support for Shelly Plus2PM
- change Shelly client to use Shelly API
  [Component model](https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Introduction)

## [[0.11.3] - 2023-06-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.3)

- serial client: add high rate rx count for debugging

## [[0.11.2] - 2023-06-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.2)

- fix race condition in Client Manager client startup (#552). This fixes a crash
  when detecting Shelly devices.

## [[0.11.1] - 2023-05-30](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.1)

- update point merge code to handle complex types
- more fixes for rule condition schedule processing (#547)
- fix issue with Shelly device discovery duplicating devices (#552)
- client manager: fix race condition with subscriptions and deleting client
  states

## [[0.11.0] - 2023-05-23](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.0)

- fix rule condition schedule processing (#547)
- support high rate serial MCU data (#517)

## [[0.10.3] - 2023-05-16](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.3)

- use mDNS responses to set shelly IO back online
- Client Manager: improve filtering of points -- see
  [Message echo](https://docs.simpleiot.org/docs/ref/client.html#message-echo)

## [[0.10.2] - 2023-05-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.2)

- default to control being disabled for shelly devices and add UI to enable
  control (#544)

## [[0.10.1] - 2023-05-13](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.1)

- fix issues with Shelly devices appearing offline when first discovered
- disable IPv6 in Shelly mDNS (does not seem to fix all issues on some machines)

## [[0.10.0] - 2023-04-28](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.0)

- support for Shelly Home Automation devices (#189) (see
  [docs](https://docs.simpleiot.org/docs/user/shelly.html))
- switch Linting/CI to use golangci-lint and fix issues in codebase
- point encode/decode functions now support arrays and maps. Thanks @bminer!

## [[0.9.0] - 2023-02-28](https://github.com/simpleiot/simpleiot/releases/tag/v0.9.0)

- change default HTTP port from 8080 to 8118. This should reduce conflicts with
  other apps and require us to configure the HTTP port less often. (#495)
- BREAKING CHANGE: change protobuf point.value encoding from float to double
  (#291) This change introduces a protocol change so all instances in a system
  will need to be updated. If this is a problem, let us know and we can work out
  a migration.
- sqlite schema: change time storage from two fields (time_s, time_ns) to single
  time that contains NS since Unix epoch.
- documentation cleanup (#509)
- move particle code to client and add UI (#503). See
  [Particle client docs](https://docs.simpleiot.org/docs/user/particle.html).
- simplify serial MCU encoding (#517)
- improve serial MCU UI point display
- use Go crypto/rand API instead of /dev/random. May fix windows issues (#517)

## [[0.8.0] - 2023-01-23](https://github.com/simpleiot/simpleiot/releases/tag/v0.8.0)

- update elm-watch to v1.1.2
- add system, application, and process
  [metrics](https://docs.simpleiot.org/docs/user/metrics.html) (#256, #255)

## [[0.7.2] - 2023-01-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.7.2)

- fix race condition with clients that have multi-level nodes (ex Rule client)
  #487

## [[0.7.1] - 2023-01-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.7.1)

(DO NOT USE, THIS VERSION HAS PROBLEMS WITH FRONTEND ASSETS)

- upgrade frontend to elm-spa 6 (#197)
- apply elm-review rules to frontend code and integrate with CI (#222)
- changes so user does not have to log in if backend or browser is restarted
  (#474)
  - frontend: store JWT Auth token in browser storage
  - frontend: store JWT key in db
- use [air](github.com/cosmtrek/air) instead of entr for watching Go files
  during development. This allows `siot_watch` to work on MacOS, and should also
  be useful in a Windows dev setup.

See the [Hot reloading the Simple IoT UI](https://youtu.be/_Nrs2_l62_Q) for a
demo of these changes.

## [[0.7.0] - 2022-12-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.7.0)

- add [CAN bus client](https://docs.simpleiot.org/docs/user/can.html)

## [[0.6.2] - 2022-12-07](https://github.com/simpleiot/simpleiot/releases/tag/v0.6.2)

- moved the node type from node point to edge field. This allows us to index
  this and make queries that search the node tree more efficient.
- support for processing clients in groups. Previously, client nodes had to be a
  child of the root device node.
- fix issue with `siot log` due to previous NATS API change

## [[0.6.1] - 2022-12-01](https://github.com/simpleiot/simpleiot/releases/tag/v0.6.1)

- fix bug in influx db client due to recent API changes
- fix bug in client manager where Stop() hangs if Start() has already exited
- don't allow deleting of root node
- allow configuring of root node ID, otherwise UUID is used
- sync:
  - add option to configure sync period (defaults to 20s).
  - if upstream node is deleted on the upstream, it is restored
  - don't include edge points of root node in hash calculation. This allows node
    to be moved around in the upstream instance and other changes.

## [[0.6.0] - 2022-11-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.6.0)

- improve error handling in serial client cobs decoder
- rename upstream -> sync
  - re-implement node hash using CRC-32 and XOR hash
  - re-implement upstream sync using new hash mechanism
  - write tests for sync
- implement `siot log` subcommand -- this dumps SIOT messages
- implement `siot store` subcommand -- used to check and fix store
- simpleiot-js frontend library changes
  - re-worked to use updated NATS API
  - added `sendEdgePoints` API function
  - added unit tests, linting, etc.

Note, there have been some database changes. To update, do the following:

- `sqlite3 siot.sqlite`
  - `update set up="root" from edges where up="none";`
- start simpleiot
  - in another terminal, run: `siot store -fix`. Do this several times until the
    original siot process does not show any fixes.

## [[0.5.5] - 2022-10-31](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.5)

- fix population of AppVersion in server
- serial client
  - add configuration of max message length
  - improve error handling and port resets

## [[0.5.4] - 2022-10-28](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.4)

- clean up SIOT main to allow callers to have their own set of flags at the top
  level before calling SIOT server.

NOTE, to run siot with flags, you must do something like:

`siot serve -debugHttp`

The server flags are now part of the serve subcommand.

## [[0.5.3] - 2022-10-27](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.3)

- add serial client debug level 9 to dump raw serial data before COBS processing

## [[0.5.2] - 2022-10-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.2)

- **Breaking change**: the node hash type has changed from a string to an int,
  which requires deleting the database and starting over.
- switch from Genesis to go-embed for embedding frontend assets
- add embedded assets FS wrapper to allow embedding compressed assets and we
  decompress them on the fly if requested.
- add `elm.js.gz` to repo. This will allow us to run SIOT without building the
  frontend first. Should enable stuff like
  `go run github.com/simpleiot/simpleiot/cmd/siot` and allow using SIOT server
  as a Go package in other projects.
- add server API to add clients. This will allow customization of what clients
  are used in the system, as well as easily adding custom ones.
- fix version in SIOT app to be Git version (was always printing development)

You can now do things like:
`go run github.com/simpleiot/simpleiot/cmd/siot@latest`

## [[0.5.1] - 2022-10-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.1)

- handle config changes in influx db client
- lifecycle improvements
  - fix race condition in http api shutdown
  - shutdown nats client after the rest of the apps
  - store: close nats subscriptions on shutdown
- Added Signal generator -- can be used to generate arbitrary signals
  (currently, high rate Sine waves only)
- add NATS subjects for high rate data (see [API](docs/ref/api.md))
- add [test app](cmd/point-size/main.go) to determine point protobuf sizes
- fix synchronization problem on shutdown -- need to wait for clients to close
  before closing store, otherwise we can experience delays on node fetch
  timeouts.
- fix issue when updating multiple points in one NATS message (only the first
  got written) (introduced in v0.5.0)
- Serial MCU Client:
  - added debug level for logging points and
    [updated what logging levels mean](https://docs.simpleiot.org/docs/user/mcu.html).
  - don't send rx/tx stats reset points to MCU
  - support high-rate MCU data (set message subject to `phr`).

## [[0.5.0] - 2022-09-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.0)

**NOTE, this is a testing release where we are still in the middle of reworking
the store and various clients. Upstream functionality does not work in this
release. If you need upstream support, use a 0.4.x release.**

The big news for this release is switching the store to SQLite and moving rule
and db functionality out of the store and into clients.

- switch store to sqlite (#320)
- rebroadcast messages at each upstream node (#390)
- extensive work on client manager. It is now much easier to keep your local
  client config synchronized with ongoing point changes. Client manager also now
  supports client configurations with two levels of nodes, such as is used in
  rules where you have a rule node and child condition/action nodes.
- fix bug with fast changes in UI do not always stick (#414)
- move rules engine from store to siot client (#409)
- move influxdb code from store to client package (#410)
- replace all NatsRequest payloads with array of points (#406)

## [[0.4.5] - 2022-09-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.5)

- set time on points received from serial MCU if not set
- display key in points if set

## [[0.4.4] - 2022-09-01](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.4)

- switch serial CRC algorithm to CRC-16/KERMIT

## [[0.4.3] - 2022-08-29](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.3)

- serial MCU: display rx/tx stats and any extra points in UI

## [[0.4.1] - 2022-08-24](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.1)

- docs: add
  [Modbus user documentation](https://docs.simpleiot.org/docs/user/modbus.html).
- docs: add
  [Notification user documentation](https://docs.simpleiot.org/docs/user/notifications.html)
- data/merge.go: fix bug if text and value are both 0
- support Debug levels in serial MCU client: 0=no messages, 1=ascii log, 2=dump
  rx data
- serial MCU client: fix issue where reset error count was not working

## [[0.4.0] - 2022-07-29](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.0)

- serial [MCU client](https://docs.simpleiot.org/docs/ref/serial.html) support
  (#380)
- add
  [origin field](https://docs.simpleiot.org/docs/ref/data.html#tracking-who-made-changes)
  to point type (#391).

## [[0.3.0] - 2022-07-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.3.0)

This release has a few bug fixes and contains new client code that will make
creating new functionality easier.

- Fix invalid users causes panic in Go code #365
- implement data.Decode/Encode for converting nodes to user structs #384
- improve startup/shutdown lifecycle #389
- implemented struct <-> type
  [encode/decode](https://github.com/simpleiot/simpleiot/blob/master/data/encode_decode_test.go)
  functions.
- improved the lifecycle management of the application so we can cleanly shut it
  down. This allows us to test the application more easily (spin up version for
  test, shutdown, repeat).
- implemented a test.Server() function to create a test server to be used in
  tests.
- Go API Change: the `nats` package has been renamed to `client`.
- defined a new Client interface and a client Manager that watches for node
  changes and creates/updates clients and sends any points changes.

## [[0.2.0] - 2022-05-31](https://github.com/simpleiot/simpleiot/releases/tag/v0.2.0)

(implemented in PR #362)

- UI: fix sorting of Rule child nodes
- highlight rule actions when active #266
- better linking of nodes for rules #251
- display clipboard contents at top of screen
- update elm/virtual-dom to 1.0.3 (helps
  [prevent xss attacks](https://jfmengels.net/virtual-dom-security-patch/))

This release improves the process of linking nodes to rule actions or
conditions. In the past, the system clipboard was used and you had to paste the
system clipboard contents into the Node ID field of rule conditions and actions.
Now, when you a copy a node, the SIOT frontend has its own clipboard and a past
button is displayed below the Node ID fields for easy pasting the node ID.
Additionally, the node description is displayed below the Node ID field so you
can easily tell which node the ID is referring to.

A [video is available](https://youtu.be/tqbLZ9CSzRU) which illustrates how node
IDs can now be copied and pasted.
[docs](https://docs.simpleiot.org/docs/user/rules.html) are also updated.

## [[0.1.0] - 2022-05-13](https://github.com/simpleiot/simpleiot/releases/tag/v0.1.0)

- docs: add list of supported devices to install
- docs: add upstream documentation
- add support for 1-wire buses, and DS18B20 temp sensor #230 #361

## [[0.0.45] - 2022-04-30](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.45)

- add DeleteNode, MoveNode, and MirrorNode to
  [nats package](https://pkg.go.dev/github.com/simpleiot/simpleiot@v0.0.44/nats).
  #344, #347
- store and display App Version in root node (see screenshot below). This value
  is extracted by the SIOT build using the `git describe` command. See
  `envsetup.sh`. #192, #349
- store and display OS version in root node (see screenshot below). On Linux,
  this value is extracted from the `VERSION` field in `/etc/os-release`. The
  field can be
  [changed](https://docs.simpleiot.org/docs/user/configuration.html) using the
  OS_VERSION_FIELD environment variable. #324, #353
- update go.bug.st/serial from v1.1.3 -> v1.3.5
- sort nodes in UI a little nicer, conditions before actions, move more often
  used nodes to the top, etc. #355, #337
- add NATS user auth API and change HTTP auth to use that. #326, #356
- fix bug where deleted nodes where still considered for user auth
- add SIOT JS library using NATS over WebSockets (#357)

![os/app version](https://user-images.githubusercontent.com/402813/163829093-14c0d644-243d-49e0-9c83-acc3c642c9ab.png)

## [[0.0.44] - 2022-04-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.44)

- UI: fix bug where copy node crashes UI if no on secure URL or localhost (#341)
- support clone/duplicate node (as well as mirror) operation (#312). Now when
  you copy and paste a node, you will be presented with a list of options as
  shown below. The new duplicate option allows you to easily replicate complex
  setups (for instance a bunch of modbus points) from an existing site to a new
  site.

![copy options](https://user-images.githubusercontent.com/402813/153455487-66bc2699-1026-40de-9ca6-4f30f91aeff9.png)

See
[documentation](https://docs.simpleiot.org/docs/user/ui.html#deleting-moving-mirroring-and-duplicating-nodes)
or a [demo video](https://youtu.be/ZII9pzx9akY) for more information.

## [[0.0.43] - 2022-03-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.43)

- improvement in UI to fix collapsing nodes #259
- implemented functionality to duplicate nodes and refactored
  copy/move/mirror/duplicate UI (#312)
- update nats-server from v2.6.6 -> v2.7.4 (and associated dependencies)

## [[0.0.42] - 2022-02-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.42)

- move HTTP API to get nodes for user to use NATS instead of direct call into
  database (#327)
- **BREAKING API CHANGE**: the Nats `inst.<id>` subject now returns an array of
  `data.NodeEdge` structs instead of a single node. Both instances of an
  upstream connection must be upgraded.
- don't send deleted nodes to frontend -- may fix #259
- default to nats/websocket being enabled on port 9222

## [[0.0.41] - 2022-01-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.41)

- with v0.0.40, if upstream URI was specified as ws://myserver.com without the
  port being specified, the NATS Go client assumed the port was 4222. If this
  port is not specified for ws or wss protocols, SIOT now sets the port to :80
  or :443. This makes the behavior more predictable, as these kinds of problems
  are very hard to debug. #315
- if upstream config changes, restart upstream connection. #258

## [[0.0.40] - 2022-01-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.40)

- support for NATS over WS connections to upstream. This is handy for cases
  where the edge network may block outgoing connections on the port NATS is
  using. HTTP(s) almost always works. In the upstream config, simply change the
  URL to something like: `ws://my.service.com`.

## [[0.0.39] - 2021-12-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.39)

- fix issue where app exits if upstream auth is incorrect (#298)
- fix issues with orphaned device nodes in upstreams. We now make sure devices
  in upstream have upstream edges or are not deleted if the device is still
  receiving points. (#299)
- only report nats stats every 1m. This makes upstream work better as these
  currently are run in sync.

## [[0.0.38] - 2021-11-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.38)

- fix population of version when building with envsetup.sh
- changes to point data structure to make it more flexible
  ([ADR-1](https://github.com/simpleiot/simpleiot/pull/279))

## [[0.0.37] - 2021-10-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.37)

- fix issue with setup where you sometimes get error: elm: Text file busy
- cleanup simpleiot.Start() so it actually returns

## [[0.0.36] - 2021-10-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.36)

- rename `db` package to `store`
- factor out siot server startup code into simpleiot package
- change `siot_run` in `envsetup.sh` to `go build` instead of `go run`

## [[0.0.35] - 2021-10-04](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.35)

- add placeholders for some UI forms
- add disable for Modbus and Modbus client nodes (#250)
- clean up locking issues and simplify DB code

## [[0.0.34] - 2021-09-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.34)

- handle audio playback where file bitrate does not match default of audio
  device (#240)
- support rule actions that trigger when rule goes inactive (instead of active).
  This allows a rule to do something with the run goes active as well as
  inactive and in some cases saves us from writing two rules (#241).
- re-enable indexes on edge up/down fields (#219)
- add point min/max to NATS packets
- add NATS api metrics (as points to root device node) (#244)
- don't color root node grey for now
- update influxdb client to 2.5.0
- switch to async influx DB API (batches data, retries, etc)
- implement caching of nodes and edges to speed up read access
- add point processing cycle time and nats client pending messages metrics
- modbus loglevel 1 only prints errors, 2 now prints transactions
- web UI auth expires in 24hr instead of 30m -- still not ideal, but one step at
  a time (#249)
- update front tar package to fix security warnings

## [[0.0.33] - 2021-08-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.33)

- fix frontend build issue with last two releases
- add rule audio playback action functionality for Linux (requires alsa-utils)
- fix various bugs with rule schedule condition functionality
- all using rule active in rule conditions (allows chaining rules)
- improve rule condition processing to process all conditions/points rather than
  just first match
- implement schedule conditions for rules
- switch from github.com/dgrijalva/jwt-go to github.com/golang-jwt/jwt/v4
- update frontend dependencies to satisfy github security checks

## [[0.0.32] - 2021-08-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.32)

- DO NOT USE, FRONTEND BUILD ISSUE

## [[0.0.31] - 2021-08-10](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.31)

- DO NOT USE, FRONTEND BUILD ISSUE

## [[0.0.30] - 2021-07-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.30)

- fix using SIOT_AUTH_TOKEN for -logNats command line option
- upgrade to NATS 2.2.2. Increases SIOT binary by about 2MB (uncompressed), 1MB
  (compressed)
- disable badger for now -- can be re-enabled in db/genji.go. Bolt seems to work
  better for the current SIOT use cases and Badger just adds bloat to the
  binary.
- implement upstream synronization support
  [#109](https://github.com/simpleiot/simpleiot/issues/109)
- update to Genji v0.13.0

## [[0.0.29] - 2021-04-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.29)

- fix sending notifications to a single user through UI

## [[0.0.28] - 2021-04-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.28)

- modbus: don't require poll period to be set for modbus server
- modbus: fix issue with reg values being sent every poll period, even if not
  changing
- modbus: add timestamp to points being sent out
- support storing Point data in Influxdb 2.0

## [[0.0.27] - 2021-04-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.27)

- slow down manual scanning to reduce background CPU usage

## [[0.0.26] - 2021-04-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.26)

- make description in nats logger and notification messages smarter
- allow modbus busses to be added to groups as well as devices
- UI:
  - don't show node + operation for nodes that can't have child-nodes
  - force email entry to always be lowercase

## [[0.0.25] - 2021-04-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.25)

- track user parent when messaging. This eliminates duplicate messages if a user
  is part of different groups with different messaging services -- we only want
  to message the group the user is a part of.

## [[0.0.24] - 2021-04-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.24)

- UI
  - display copy/move node messages for 2-3s when clicking copy/move node button
  - support multiple top level nodes -- for instance a user that is a member of
    multiple groups but not the root node
  - automatically expand node children when moving/copying a node
- Implement rule notifications

## [[0.0.23] - 2021-04-10](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.23)

- Modbus: TCP listen on all interfaces instead of just localhost
- UI
  - add dot for nodes that don't have children
  - don't sort nodes while editing, only on fetch
  - sort nodes by group, then desc, then firstname, then lastname
  - move/copy node can use node ID or description
  - add node icons to add node descriptions
  - replace edit/collapse with dot and color exp nodes
- support copying nodes
- remove remnants of Sample types (we now use Point)
- create notification and message data types and NATS/Db support
- implement node messaging (notifies all node and upstream users)
- BUILD: simplify protobuf generation
- implement Twilio SMS messaging

## [[0.0.22] - 2021-03-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.22)

- UI: change node min/max button to edit/close
- Modbus: suppress TCP conn/disc messages at debug level 0
- siot: add cmdline option (-logNats) to trace all node points. This can be run
  in parallel to the siot application to trace points flowing through the system
- genji db: update to v0.11.0 release
- rules: can now write rules that set nodes based on other nodes

## [[0.0.21] - 2021-03-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.21)

- modbus: fix server issue with requests not free resources

## [[0.0.20] - 2021-03-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.20)

- UI: add Form.onEnter utility function for adding enter handling
- UI: enter can now be used to enter sign-in form
- support for Modbus TCP, both client and server

## [[0.0.19] - 2021-02-27](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.19)

- revert Genji update as there are problems saving nodes

## [[0.0.18] - 2021-02-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.18)

- update go.bug.st/serial to support RiscV
- update Genji dependencies

## [[0.0.17] - 2021-02-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.17)

- modbus
  - fix bug in setting modbus baud rate
  - include ID in modbus logging messages
  - support for read-only coils and holding regs
- UI
  - add nodeCheckboxInput widget
  - round numbers in places
  - color digital values blue when ON
  - sort nodes by description
- fix windows build

## [[0.0.16] - 2021-02-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.16)

- UI
  - expand child nodes and add default description when adding a new node
- modbus improvements
  - send all writes to DB over NATS -- this allows system to be more responsive,
    as well as simplifies code
  - lots of cleanup and error handling

## [[0.0.15] - 2020-12-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.15)

- Implementation of tree based UI -- see demo: https://youtu.be/0ktVCPU74mw

## [[0.0.14] - 2020-11-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.14)

- fix 32bit binary build

## [[0.0.13] - 2020-11-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.13)

- edge:
  - fixed issue with backoff algorithm not adhering to max
- backend:
  - switched data structure name from device -> node -- see
  - [this issue](https://github.com/simpleiot/simpleiot/issues/91) for
    discussion
  - add page to message (currently SMS only) all users
  - UI simplification and cleanup
  - sort users on users page
  - port frontend to elm-spa.dev v5 (this really cleans up the frontend code and
    makes it more idomatic Elm)
  - changing backing store from bolthold to genji (this gives us the flexibility
    to use memory, bbolt, or badger backing stores as well as robust indexing)
  - fix bug with not support Point::Text field in Nats/Protobuf
  - fix up examples for sending device version info to portal
- frontend:
  - only show version information if available
  - don't display special points (description, version, etc) in general node
    points.
  - add -importDb command line option

Note, the database format has changed. To migrate, dump the database with the
old version of SIOT and them import with the new version.

## [[0.0.12] - 2020-11-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.12)

- backend:
  - switched data structure name from device -> node -- see
    [this issue](https://github.com/simpleiot/simpleiot/issues/91) for
    discussion
  - add page to message (currently SMS only) all users
  - UI simplification and cleanup
  - sort users on users page
  - port frontend to elm-spa.dev v5 (this really cleans up the frontend code and
    makes it more idomatic Elm)
  - changing backing store from bolthold to genji (this gives us the flexibility
    to use memory, bbolt, or badger backing stores as well as robust indexing)
  - fix bug with not support Point::Text field in Nats/Protobuf
  - fix up examples for sending device version info to portal
- frontend:
  - only show version information if available
  - don't display special points (description, version, etc) in general node
    points.
  - add -importDb command line option

Note, the database format has changed. To migrate, dump the database with the
old version of SIOT and them import with the new version.

## [[0.0.11] - 2020-09-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.11)

### Changed

- switched data storage to
  [points](https://github.com/simpleiot/simpleiot/blob/master/docs/development.md#flexible-data-structures)
  vs sensor data and config
- add token auth for device HTTP communication
- documentation improvements

## [[0.0.10] - 2020-08-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.10)

### Changed

- documentation improvements
- specify TLS certs using variables instead of embedding
- code cleanup around NATS integration
- NATS don't force TLS 1.2 in client
- remove siotutil functionality and fold into siot exe

## [[0.0.9] - 2020-08-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.9)

### Added

- NATS integration for device communication

### Changed

- documentation improvements
  - moved API documentation to simple Markdown
  - better organization
  - add list of guiding principles to the [README](./README.md)

## [[0.0.8] - 2020-08-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.8)

### Added

- moved influxDb operations to db package so they are common for all samples
- added env variable to specify Influx database SIOT_INFLUX_DB
- added device ID tag to sample data stored in influx
- add rules engine
- add SMS notifications using Twilio

### Changed

- clean up documentation organization

## [[0.0.7] - 2020-07-04](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.7)

### Added

- display device last update time
- display time since last update

## [[0.0.6] - 2020-06-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.6)

### Added

- add modbus API to change debug level at runtime
- add cloud/cloud off icon to indicate connection status of devices
- grey out devices that are not currently connected
- added background process to determine if devices are offline

### Fixed

- workaround for issue where group key in database does not match ID in struct

## [[0.0.5] - 2020-06-16](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.5)

### Fixed

- fixed critical bug where new devices were not showing up in UI

### Added

- add support in modbus pkg for decoding 32-bit int and floating point values
- started general command line modbus utility (cmd/modbus) to interactively read
  modbus devices
