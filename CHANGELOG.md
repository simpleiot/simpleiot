# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For more details or to discuss releases, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## Next

- lots of modbus work
- expand child nodes and add default description when adding a new node

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

## [Unreleased]

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
