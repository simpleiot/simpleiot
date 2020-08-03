---
id: api
title: API
sidebar_label: API
---

The Simple IoT server currently provides both Http and NATS.io APIs. We've tried
to keep the two APIs a similar as possible so it is easy to switch from one to
the other. The Http API currently accepts JSON, and the NATS API uses protobuf.

## Http

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
  - `/v1/devices/:id/config
    - POST: update config for a device
  - `/v1/devices/:id/samples
    - POST: post samples for a device
  - `/v1/devices/:id/cmd
    - GET: gets a command for a device and clears it from the queue. Also clears
      the CmdPending flag in the Device state.
    - POST: posts a cmd for the device and sets the device CmdPending flag.
- Users
  - [data structure](https://github.com/simpleiot/simpleiot/blob/master/data/user.go)
  - `/v1/users`
    - GET: default is to return list of all users. An `email` query parameter
      can also be used to find a specific user by email.
    - POST: create a new user
  - `/v1/users/:id
    - GET: return info for a single user
    - POST: update a user
    - DELETE: delete a user
- Groups
  - [data structure](https://github.com/simpleiot/simpleiot/blob/master/data/group.go)
  - `/v1/groups`
    - GET: return list of all groups
    - POST: create a new group
  - `/v1/groups/:id
    - GET: return info for a single group
    - POST: update a group
    - DELETE: delete a group
- Auth
  - `/v1/auth`
    - POST: accepts `email` and `password` as form values, and returns a JWT
      Auth
      [token](https://github.com/simpleiot/simpleiot/blob/master/data/auth.go).

## NATS

- Devices
  - `device.<id>.samples`
    - device publishes samples and the server updates device state and stores
      sample in database.
  - `device.<id>.file`
    - is used to transfer files to a device in chunks, which is optimized for
      unreliable networks like cellular.

## Old

Previous, the API documentation was done with API Blueprint and will eventually
be removed.

(the current API documentation is not current -- looking for a better way to
document that API)

The REST API used by the frontend and devices is documented (below is no longer
current -- currently evaluating the best path forward to document that API).
[here](https://htmlpreview.github.io/?https://github.com/simpleiot/simpleiot/blob/master/docs/api.html)
using [API Blueprint](api.apibp).

### Examples of looking at API data

- install `wget` and `jq`
- `wget -qO - http://localhost:8080/v1/devices | jq -C`
