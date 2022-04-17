# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For more details or to discuss releases, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## [Unreleased]

- add DeleteNode, MoveNode, and MirrorNode to
  [nats package](https://pkg.go.dev/github.com/simpleiot/simpleiot@v0.0.44/nats).
- store and display App Version in root node

## [[0.0.44] - 2022-04-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.44)

- UI: fix bug where copy node crashes UI if no on secure URL or localhost (#341)
- support clone/duplicate node (as well as mirror) operation (#312). Now when
  you copy and paste a node, you will be presented with a list of options as
  shown below. The new duplicate option allows you to easily replicate complex
  setups (for instance a bunch of modbus points) from an existing site to a new
  site.

![copy options](https://user-images.githubusercontent.com/402813/153455487-66bc2699-1026-40de-9ca6-4f30f91aeff9.png)

See
[documenation](https://docs.simpleiot.org/docs/user/ui.html#deleting-moving-mirroring-and-duplicating-nodes)
or a [demo video](https://youtu.be/ZII9pzx9akY) for more information.

## [[0.0.43] - 2022-03-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.43)

- improvement in UI to fix collapsing nodes #259
- implemented functionality to duplicate nodes and refactored
  copy/move/mirror/duplicate UI (#312)
- update nats-server from v2.6.6 -> v2.7.4 (and associated dependencies)

## [[0.0.42] - 2022-02-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.42)

- move HTTP API to get nodes for user to use NATS instead of direct call into
  database (#327)
- **BREAKING API CHANGE**: the Nats `node.<id>` subject now returns an array of
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
- update frontend dependencies to satisify github security checks

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
  - don't sort nodes while editting, only on fetch
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
  - fixed issue with backoff algorith not adhearing to max
- backend:
  - switched data structure name from device -> node -- see
  - this issue for dicussion
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
    [this issue](https://github.com/simpleiot/simpleiot/issues/91) for dicussion
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
  - add list of guiding principles to the [README](README.md)

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
