---
id: api
title: API
sidebar_label: API
---

The Simple IoT server currently provides both Http and NATS.io APIs. We've tried
to keep the two APIs a similar as possible so it is easy to switch from one to
the other. The Http API currently accepts JSON, and the NATS API uses protobuf.

**NOTE, the Simple IoT API is not final and will continue to be refined in the
coming months.**

## HTTP

For details on data payloads, it is simplest to just refer to the Go types which
have JSON tags.

Most APIs that do not return specific data (update/delete) return a
[StandardResponse](https://github.com/simpleiot/simpleiot/blob/master/data/api.go)

- Devices
  - [data structure](https://github.com/simpleiot/simpleiot/blob/master/data/device.go)
  - `/v1/devices`
    - GET: return a list of all devices
  - `/v1/devices/:id`
    - GET: return info about a specific device
    - DELETE: delete a device
  - `/v1/devices/:id/config`
    - POST: update config for a device
  - `/v1/devices/:id/samples`
    - POST: post samples for a device
  - `/v1/devices/:id/cmd`
    - GET: gets a command for a device and clears it from the queue. Also clears
      the CmdPending flag in the Device state.
    - POST: posts a cmd for the device and sets the device CmdPending flag.
  - `/v1/devices/:id/version`
    - POST: version information sent to the server from the device that contains
      version information.
    - [DeviceVersion](../data/device.go)
- Users
  - [data structure](https://github.com/simpleiot/simpleiot/blob/master/data/user.go)
  - `/v1/users`
    - GET: default is to return list of all users. An `email` query parameter
      can also be used to find a specific user by email.
    - POST: create a new user
  - `/v1/users/:id`
    - GET: return info for a single user
    - POST: update a user
    - DELETE: delete a user
- Groups
  - [data structure](https://github.com/simpleiot/simpleiot/blob/master/data/group.go)
  - `/v1/groups`
    - GET: return list of all groups
    - POST: create a new group
  - `/v1/groups/:id`
    - GET: return info for a single group
    - POST: update a group
    - DELETE: delete a group
- Auth
  - `/v1/auth`
    - POST: accepts `email` and `password` as form values, and returns a JWT
      Auth
      [token](https://github.com/simpleiot/simpleiot/blob/master/data/auth.go).

## NATS

[NATS.io](https://nats.io/) allows more complex and efficient interactions
between various system components (device, cloud, and web UI). These three parts
of the system make IoT systems inherently distributed. NATS focuses on
simplicity and is written in Go which ensures the Go client is a 1st class
citizen and also allows for interesting possibilities such as embedding in the
NATS server in various parts of the system. This allows us to keep our
one-binary deployment model.

The `siot` binary embeds the NATS server, so there is no need to deploy and run
a separate NATS server.

For the NATS transport, protobuf encoding is used for all transfers and are
defined [here](../internal/pb).

- Devices
  - `device.<id>.samples`
    - device publishes samples and the server updates device state and stores
      sample in database.
  - `device.<id>.file`
    - is used to transfer files to a device in chunks, which is optimized for
      unreliable networks like cellular and is handy for transfering software
      update files. There is Go code [available](../api/nats-file.go) to manage
      both ends of the transfer as well as a utility to [send](../cmd/siotutil)
      files and an example [edge](../cmd/edge) application to receive files.
  - `device.<id>.cmd`
    - send a [DeviceCmd](../data/device.go) to a device. `siotutil` can be used
      to test sending commands to devices using NATS.
  - `device.<id>.version`
    - [DeviceVersion](../data/device.go) sent from device to server to inform
      the server of the versions of various components on the device.
- System
  - `error`
    - any errors that occur are sent to this subject
