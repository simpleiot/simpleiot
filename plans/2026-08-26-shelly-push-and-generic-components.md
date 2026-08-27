# Plan: Shelly Push Updates and Generic Component Support

**Branch:** current checkout **Branched from:** `98e0fa88`

## Context

The Shelly client works well for the handful of devices it knows about and
poorly for everything else, and it holds the network open constantly to do it.
Both limits come from the same decision: the client decides what a device is,
and what it can do, from tables in `client/shelly-io.go` keyed by a model string
parsed out of the mDNS hostname.

Home Assistant's Shelly integration (and the `aioshelly` library under it) takes
the opposite approach on both counts, and the Shelly API supports it directly.
This plan brings the two ideas across.

### What Home Assistant does

**Components are read from the device, not from a table.** `aioshelly` calls
`Shelly.GetDeviceInfo` for the generation, model, and firmware, then
`Shelly.GetConfig` and `Shelly.GetStatus`. The top-level keys of those responses
are the components: `switch:0`, `input:1`, `cover:0`, `light:0`, `rgb:0`,
`em:0`, `temperature:100`, `humidity:0`, `devicepower:0`, and so on. Home
Assistant enumerates a component by prefix match on `"<name>:"` and builds an
entity for each instance it finds. Capability follows the same way: a `switch:0`
status carrying `apower`, `voltage`, and `current` has power monitoring, so no
table records which models do.

`aioshelly` does carry a model list, but only for display names, generation,
minimum firmware, and a few per-model quirks. It is not what decides what a
device exposes, which is why a Gen2 device released after the library was
written still works.

**Status is pushed, not polled.** `aioshelly` opens one WebSocket to
`ws://<ip>/rpc` and leaves it open. Once a request frame carrying a `src` has
been sent, the device pushes JSON-RPC notifications on that same socket:
`NotifyStatus` with the changed fields, which the library deep-merges into its
cached status, `NotifyFullStatus` on reconnect, and `NotifyEvent` for button
clicks, configuration changes, and OTA progress. Home Assistant polls
`Shelly.GetStatus` only every 60 seconds, as a backstop for the few values that
do not push, and treats loss of the WebSocket as the device going offline.

For Gen1 devices the same job is done by CoIoT: the device multicasts CoAP
status frames, and the library merges them into a block/sensor map built once
from `cit/d`. For battery devices Home Assistant configures the device's own
outbound WebSocket (`Ws.SetConfig`) to point back at Home Assistant, so a
sleeping device connects out when it wakes.

### What ours does

`client/shelly.go` parses the mDNS hostname with `shelly(.*)-(.*).local` and
stores the middle as `type`. Everything after that is a table lookup on that
string: `shellyGenMap` for the generation, `shellyCompMap` for which components
exist, `shellySettableOnOff` for whether it can be driven, `shellyHasPM` for
whether to read power. `ShellyIOClient` then polls each component individually
every 2 seconds — for a Plus 2PM that is four HTTP round trips every 2 seconds,
per device — and a set command waits for the next tick before it takes effect.

The tables have drifted from each other, and the gaps are the kind that only a
data-driven approach avoids:

- `Plus1PM` is in `shellyCompMap` and `shellyHasPM` but not in `shellyGenMap`,
  so `Gen()` returns `ShellyGenUnknown`, `getConfig` fails with "unsupported
  device", and `GetStatus` returns nothing. The device is listed as supported in
  `docs/user/shelly.md`.
- `shellyCompMap[PlusI4]` is
  `{{"input", 0}, {"input", 0}, {"input", 0}, {"input", 0}}` — four entries all
  with id 0. Input 0 is read four times and inputs 1 through 3 never appear.
- `rgbw2` is in `shellyGenMap` and `shellySettableOnOff` but has no
  `shellyCompMap` entry, so `GetStatus` returns nothing for it.
- Gen1 `SetOnOff` builds `http://<ip>/switch/<n>?turn=on`. Gen1 relays are at
  `/relay/<n>`; `/switch/<n>` does not exist, so Gen1 relay control cannot work.
  The response is also decoded as a light status whatever the component was.
- `gen1GetSwitch` indexes `swi.Relays[index]` and `swi.Meters[index]` with no
  length check, so a short response panics.
- Any device whose hostname stem is not in the maps becomes a node that cannot
  be read at all.

Reading components from the device removes the tables, and with them every one
of these.

## Design Decisions

**Keep one `shellyIo` node per device.** The component model changes what points
a node carries, not how nodes are arranged. A device stays one node, and each
component instance stays a point keyed by its component id, which is what the
frontend already renders.

**Point type comes from the component name, key from the component id.** A
`switch:1` status becomes `switch` points with key `1`; `cover:0` becomes
`cover` points with key `0`. The frontend already sorts and displays points by
type and key and formats them through a dictionary in `NodeShellyIO.elm`, so a
component we have not seen renders as a labeled row rather than an error. Fields
inside a component map to point types the same way: `apower` to `power`,
`voltage` to `voltage`, `current` to `current`, `tC` to `temp`.

**Discovery reads the device rather than the hostname.** After mDNS finds a
host, the client calls `Shelly.GetDeviceInfo`, which returns `gen`, `model`,
`id`, and `mac` directly. `type` on the node becomes the model string the device
reports. This also settles Gen1 versus Gen2 without a table: `gen` is in the
response, and a device that does not answer `/rpc/Shelly.GetDeviceInfo` is Gen1.

**The WebSocket carries status; a slow poll remains as a backstop.** The
WebSocket is the primary path and its state is the online/offline signal, which
is both faster and more accurate than counting five failed polls. A 60-second
`Shelly.GetStatus` stays behind it, matching Home Assistant, to cover anything
the device does not push and to re-sync after a missed frame.

**Set commands go out immediately.** With status arriving by push, there is no
tick to wait for. A `switchSet` point calls `Switch.Set` when it arrives, and
the resulting `NotifyStatus` reports what actually happened.

**Gen1 stays on polling for now.** Gen1 `/status` returns the whole device in
one request, so polling it costs one round trip rather than one per component.
CoIoT would remove that too, but it needs a CoAP listener and the `cit/d`
block/sensor decoding, which is a larger piece of work than it earns while Gen1
devices are a small part of the range. Phase 4 covers it if we want it.

## Phases

### Phase 1: Read components from the device — COMPLETE

- Add `Shelly.GetDeviceInfo` and store `gen`, `model`, `mac` on the node.
- Replace `shellyGenMap`, `shellyCompMap`, `shellySettableOnOff`, and
  `shellyHasPM` with a parse of `Shelly.GetConfig` and `Shelly.GetStatus` into a
  list of `{name, id}` components and their available fields.
- Generate the node's component points from that list at discovery, replacing
  `addCompPoints` in `client/shelly.go`.
- Keep the existing polling loop; only the source of the component list changes.
- Fix the Gen1 relay path and the unchecked slice indexing while the Gen1 code
  is open.
- Update `docs/user/shelly.md`: the supported-device list becomes a statement
  that any Gen2 or later device is read from its own component list.

### Phase 2: Push updates over the RPC WebSocket — COMPLETE

- Add a WebSocket RPC connection to `ShellyIOClient` for Gen2+ devices: connect
  to `ws://<ip>/rpc`, send an initial `Shelly.GetStatus` carrying a `src`, then
  read frames in a goroutine and deliver them to the `Run` select loop.
- Handle `NotifyStatus` by merging into the cached status, `NotifyFullStatus` by
  replacing it, and `NotifyEvent` for input events.
- Drive `offline` from the connection state, and reconnect on a 60-second
  interval while disconnected.
- Reduce the poll to 60 seconds as a backstop, and drop the 2-second tick.
- Send set commands on point arrival rather than on the tick.
- `github.com/gorilla/websocket` is already in `go.sum` as an indirect
  dependency; this promotes it to direct.

### Phase 3: Components beyond switch, light, and input — COMPLETE

- Add point types and set handling for the components the generic enumeration
  now surfaces: `cover` (position, open/close/stop), `rgb` and `rgbw`, `em` and
  `em1` energy meters, `temperature`, `humidity`, `devicepower`.
- Add formatters for the new point types in `NodeShellyIO.elm`.

### Phase 4 (optional): CoIoT for Gen1 — NOT DONE

- Listen for CoAP status frames on 224.0.1.187:5683, decode `cit/d` into a block
  and sensor map, and merge `cit/s` updates into it.
- Only worth doing if Gen1 devices remain in use; Phase 1 already fixes what is
  broken about them. Gen1 is now read with one request rather than one per
  component, so the cost of polling it is much lower than it was.

## What was built

Phases 1 through 3 landed together, since the component list Phase 1 produces is
what Phase 2 pushes and Phase 3 extends.

- `client/shelly-io.go` reads components and points out of a status response.
  `shellyCompPoints` maps a component name to the function that converts it, and
  is the only place a component type is named. The four model tables are gone.
- `client/shelly-rpc.go` holds the JSON-RPC frames, the HTTP call, and
  `shellyWatcher`, which keeps the WebSocket open and delivers what the device
  pushes. Only one request goes out over the socket, the `Shelly.GetStatus` that
  both fetches the starting status and, by carrying a `src`, tells the device to
  start notifying.
- `ShellyIo` holds its component state in maps keyed by component id rather than
  in arrays, because add-on components are numbered from 100 and an array would
  have to be that long to hold one.
- A node created before this change carries no generation, so the client asks
  the device for one on startup and records it along with the model.

## Testing

- `client/shelly_test.go` currently covers the hostname regex. Add table tests
  that parse recorded `Shelly.GetStatus` and `Shelly.GetConfig` responses for a
  Plus 2PM, a Plus i4, a Plus Plug US, and a Pro 3EM into the expected component
  list and points, which is the check the model tables never had.
- Add a test that a `NotifyStatus` frame merges into cached status without
  clearing fields it does not mention.
- `go test -race ./...` and `golangci-lint run` before each phase lands.
