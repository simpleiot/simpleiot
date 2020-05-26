# Simple IoT Development

This document attempts to outlines the basic architecture and development
philosophy. The basics are covered in the [readme](../README.md). As the name
suggests, a core value of the project is simplicity. Thus any changes should be
made with this in mind. This project is far from perfect and there are likely
many better ways to do things.

## Coding Standards

Please run `siot_test` from `envsetup.sh` before submitting pull requests. All
code should be formatted and linted before committing.

## flexible data structures

As we work on IoT systems, common data structures (types) tend to emerge. Common
data structures allow us to develop common algorithms and mechanism to process
data. Instead of defining a new data type for each type of sensor, define one
type that will work with all sensors. Then it is trivial to support new sensors
and applications.

These are currently defined in the `data` directory for Go code, and
`frontend/src/Data` directory for Elm code. A few of these data structures
include:

- `Device`: represents a IoT device that is capable of communicating with a siot
  server. Typically a Device functions as a gateway or edge device agregating
  data from a number of sensors and then sending the data to the siot server.
- `Sample`: a sample of sensor data.
- `Config`: defines a configuration parameter for a Device.

## Frontend architecture

Much of the frontend architecture is already defined by the Elm architecture.
However, we still have to decide how data flows between various modules in the
frontend. If possible, we'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible. Thoughts on how to accomplish this:

- single data model at top level
- modifications to the backend database are sent to the top level, the model is
  modified first, and then a request is sent to the backend to modify the
  database. This ensures the value does not flash or revert to old value while
  the backend request is being made.

## Backend architecture
