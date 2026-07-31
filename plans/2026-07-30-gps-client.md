# Plan: GPS Client

**Branch:** `feat/gps` **Branched from:** `b3ee632a`

## Context

Simple IoT has no first-class GPS support. There is a legacy `gps/` package
(`gps/gps.go`) written before the client architecture existed: it opens a serial
port with `github.com/jacobsa/go-serial`, parses NMEA GGA sentences with
`github.com/adrianmo/go-nmea`, and pushes `data.GpsPos` structs onto a channel.
Nothing in the tree imports it, and it predates NATS point publishing entirely.

This plan adds a proper `gps` client node with three sources:

1. **Serial** — read NMEA sentences directly from a receiver on a serial port.
2. **gpsd** — subscribe to the `gpsd` daemon's JSON stream over TCP. This is the
   right choice on any Linux system where gpsd already owns the receiver, where
   several processes need the same fix, or where gpsd's device autodetection and
   driver handling are worth having.
3. **Simulation** — generate a plausible track, moving at a configured speed on
   a heading that drifts randomly, so the rest of the system (rules, graphing,
   sync, dashboards) can be exercised without hardware.

All three publish the same points. That is the main design constraint, and with
three sources reporting fix status in three different vocabularies, normalizing
them is the most substantial design work in this plan.

## Design Decisions

**Node type is `gps`.** One node per receiver, added under a device or group
node, consistent with `serialDev` and `canBus`.

**Source selection via a `gpsSource` point** with values `serial`, `gpsd`, and
`sim`, rendered as a radio in the UI. The UI shows only the fields relevant to
the selected source.

**Output points are source-independent.** Latitude, longitude, altitude, speed,
heading, fix type, fix quality, satellite count, and HDOP are published on the
GPS node itself. Speed is normalized to m/s and heading to degrees true (0–360)
regardless of source.

**Fix status is normalized into two points, `fixType` and `fixQuality`.** The
sources disagree on how to describe a fix:

- NMEA GGA reports a fix _quality_ string (`0`–`6`: invalid, GPS, DGPS, PPS,
  RTK, float RTK, estimated) but says nothing about 2D versus 3D.
- NMEA GSA reports a fix _type_ (none / 2D / 3D) but nothing about augmentation.
- gpsd TPV reports `mode` (0 unknown, 1 no fix, 2 2D, 3 3D) and an optional
  `status` (0–9) covering the augmentation dimension.

Collapsing these into one point would lose information that matters — RTK fixed
versus RTK float is the difference between centimeter and decimeter accuracy,
and 2D versus 3D determines whether altitude is trustworthy. So `fixType` covers
the dimensionality and `fixQuality` covers the augmentation, with SIOT-defined
values that each source maps into. Both are numeric rather than string enums so
they survive a metrics-only database — see
[Time-Series Storage and Mapping](#time-series-storage-and-mapping). The simpler
alternative — a single `fixType` point, dropping augmentation detail — is worth
considering if RTK is not a use case you care about.

**A source publishes only what it actually reports.** If a receiver emits no GSA
sentences, `fixType` is left unpublished rather than guessed at from GGA. Points
are omitted, never fabricated.

**gpsd is spoken directly rather than through a library.** The protocol is a
line-delimited JSON stream: connect to TCP 2947, write
`?WATCH={"enable":true,"json":true}`, and decode objects tagged by a `class`
field. Only TPV and SKY are needed, which is roughly 100 lines with
`encoding/json`.
[`github.com/stratoberry/go-gpsd`](https://github.com/stratoberry/go-gpsd) is
the established Go client (v1.3.0, 2024, no external dependencies) and is a
reasonable alternative, but given the small surface area actually used and this
project's stated focus on minimal dependencies and binary size, hand-rolling
wins here.

**gpsd fields are decoded into pointers.** TPV omits fields that are
unavailable, and zero is a meaningful value for several of them — `speed: 0`
means stationary, and latitude and longitude of 0 is a real position in the Gulf
of Guinea. Decoding into `*float64` distinguishes absent from zero so the client
publishes only fields gpsd actually sent.

**No `Destination` support in the first pass.** The signal generator uses
`client.Destination` to route high-rate data to a parent or over `phrup`. GPS
updates arrive at 1–10 Hz, which the normal node point path handles comfortably.
Adding `Destination` later is additive.

**The simulator is a separate file** (`client/gps-sim.go`) with no NATS, serial,
or network dependencies, so its movement math is directly unit-testable.

**Great-circle movement, not a flat-earth approximation.** The destination-point
formula is four lines and stays correct near the poles and across the
antimeridian, which the equirectangular shortcut does not.

## Point and Node Types

Added to `data/schema.go`:

```go
// GPS client
NodeTypeGPS = "gps"

PointTypeGPSSource        = "gpsSource"
PointValueGPSSourceSerial = "serial"
PointValueGPSSourceGpsd   = "gpsd"
PointValueGPSSourceSim    = "sim"

// GPS output points
PointTypeLatitude  = "latitude"   // degrees, +N
PointTypeLongitude = "longitude"  // degrees, +E
PointTypeAltitude  = "altitude"   // meters above mean sea level
PointTypeSpeed     = "speed"      // meters/second over ground
PointTypeHeading   = "heading"    // degrees true, 0-360
PointTypeNumSat    = "numSat"     // satellites used in fix
PointTypeHDOP      = "hdop"       // horizontal dilution of precision
PointTypeGPSTime   = "gpsTime"    // Unix epoch seconds reported by the source

// Normalized fix dimensionality, following gpsd's TPV mode encoding.
// Numeric so it stores in metrics-only databases -- see "Time-Series Storage".
PointTypeFixType  = "fixType"
PointValueFixNone = 0 // no fix, or fix status unknown
PointValueFix2D   = 2
PointValueFix3D   = 3

// Normalized fix augmentation quality, following the NMEA GGA fix quality
// encoding, which already covers every case the three sources report.
PointTypeFixQuality           = "fixQuality"
PointValueFixQualityNone      = 0
PointValueFixQualityGPS       = 1
PointValueFixQualityDGPS      = 2
PointValueFixQualityPPS       = 3
PointValueFixQualityRTKFixed  = 4
PointValueFixQualityRTKFloat  = 5
PointValueFixQualityEstimated = 6
PointValueFixQualityManual    = 7
PointValueFixQualitySimulated = 8

// gpsd source config
PointTypeGpsdAddress = "gpsdAddress" // host:port, default localhost:2947

// GPS simulation config
PointTypeSimLatitude    = "simLatitude"    // starting latitude
PointTypeSimLongitude   = "simLongitude"   // starting longitude
PointTypeSimSpeed       = "simSpeed"       // meters/second
PointTypeSimHeading     = "simHeading"     // starting heading, degrees true
PointTypeSimHeadingRate = "simHeadingRate" // max heading change, degrees/second
```

Reused existing types: `description`, `disabled`, `port`, `baud`, `debug`,
`device` (selects which gpsd device to watch when gpsd manages more than one),
`period`, `connected`, `errorCount`, `errorCountReset`, `rx`, `rxReset`.

`period` is the simulation update interval in seconds (default 1). It is ignored
by the serial and gpsd sources, where the receiver sets the rate.

### Fix Normalization Tables

`fixType` — SIOT adopts gpsd's `mode` encoding directly, so the gpsd path is a
pass-through and only the NMEA path maps:

| SIOT             | NMEA GSA `FixType` | gpsd TPV `mode`       | sim |
| ---------------- | ------------------ | --------------------- | --- |
| `0` none/unknown | `1` FixNone        | `0` unknown, `1` none |     |
| `2` 2D           | `2` Fix2D          | `2`                   |     |
| `3` 3D           | `3` Fix3D          | `3`                   | ✓   |
| unpublished      | no GSA emitted     |                       |     |

`fixQuality` — SIOT adopts the NMEA GGA encoding, so the serial path is a
pass-through and only the gpsd path maps:

| SIOT           | NMEA GGA `FixQuality` | gpsd TPV `status`            | sim |
| -------------- | --------------------- | ---------------------------- | --- |
| `0` none       | `0` Invalid           | `0` Unknown with mode 0 or 1 |     |
| `1` GPS        | `1` GPS               | `1` Normal                   | ✓   |
| `2` DGPS       | `2` DGPS              | `2` DGPS                     |     |
| `3` PPS        | `3` PPS               | `9` P(Y)                     |     |
| `4` RTK fixed  | `4` RTK               | `3` RTK Fixed                |     |
| `5` RTK float  | `5` FRTK              | `4` RTK Floating             |     |
| `6` estimated  | `6` EST               | `5` DR, `6` GNSSDR           |     |
| `7` manual     | `7` manual input      | —                            |     |
| `8` simulated  | `8` simulation mode   | `8` Simulated                |     |

Note that `go-nmea` only defines constants through `6` EST, though the NMEA
specification defines `7` and `8` as well; the client handles the full range.

NMEA `3` PPS means Precise Positioning Service, the military precise code, not
pulse-per-second — so mapping gpsd's `9` P(Y) onto it is correct. gpsd `status`
values with no SIOT equivalent (`7` Time surveyed) publish `1` GPS when `mode`
is 2 or 3, since the position is valid; the position is what consumers act on.

## Time-Series Storage and Mapping

The GPS points need to land in the time-series database and plot on a Grafana
geomap. Both work, but three constraints fall out of how the
[database client](../docs/user/database.md) writes points, and they shape the
design above.

**The db client writes every point as Influx measurement `points` with fields
`value` (float) and `text` (string), tagged with `type`, `key`, `node.id`,
`node.type`, `node.description`, and any configured custom tags.** InfluxDB
stores both fields. VictoriaMetrics accepts the same line protocol and splits
each field into its own series — `points_value` and `points_text` — but
[converts non-numeric field values to 0](https://docs.victoriametrics.com/victoriametrics/integrations/influxdb/).
The line is still ingested, so numeric points are unaffected, but the content of
any string point is lost.

**Therefore fix status and GPS time are numeric points, not strings.** This is
why `fixType` and `fixQuality` use the numeric encodings above rather than the
readable string enums an earlier draft of this plan proposed, and why `gpsTime`
is Unix epoch seconds rather than an ISO 8601 string. Numeric values keep the
data queryable in VictoriaMetrics, and the frontend renders the labels. The cost
is that a raw database query shows `4` instead of `rtkFixed`; the mapping tables
above belong in the user documentation for that reason, and Grafana value
mappings can restore the labels in a panel.

**Every point in one fix must carry an identical timestamp.** `client.SendPoints`
stamps each point with its own `time.Now()` when `Point.Time` is zero, which
would leave latitude and longitude microseconds apart. A geomap needs both
coordinates in the same row, so the client takes one timestamp per fix and
applies it to every point in that fix before publishing. Without this, joining
latitude and longitude in Grafana yields rows where one is null.

### Grafana Geomap

The geomap panel's Coordinates location mode needs latitude and longitude as two
numeric fields in the same frame. Getting there differs by database:

- **InfluxDB** — a Flux `pivot()` on `_field`/`type` puts latitude and longitude
  into one row natively, which is the simpler path. Rename the resulting columns
  to `latitude` and `longitude` and the panel auto-detects them.
- **VictoriaMetrics** — query `points_value{type="latitude"}` and
  `points_value{type="longitude"}` as two queries, then apply a
  **Join by field** transformation on Time, an **Organize fields**
  transformation to rename them to `latitude` and `longitude`, and set the panel
  to Coordinates mode. A range query aligns both series to the same step grid,
  so set the panel's min step at or below the GPS update rate to avoid
  decimating the track.

Two things to verify during Phase 6 rather than assume:

- The db client's tag names contain dots (`node.id`), which is not a valid
  Prometheus label name. VictoriaMetrics is more permissive than Prometheus
  here, but Grafana's Prometheus datasource may need quoted-label selector
  syntax. VictoriaMetrics' `-usePromCompatibleNaming` flag rewrites these to
  `node_id` and is likely the smoother configuration.
- Because the db client writes a `text` field on every point, VictoriaMetrics
  gets a `points_text` series holding zeros alongside every real series. Harmless
  but wasteful. Reducing it means changing the db client to omit empty text
  fields, which is out of scope here — worth noting as a follow-up.

## Client Configuration Struct

```go
// GPS client configuration
type GPS struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Source      string `point:"gpsSource"`
	Debug       int    `point:"debug"`

	// serial source
	Port string `point:"port"`
	Baud string `point:"baud"`

	// gpsd source
	GpsdAddress string `point:"gpsdAddress"`
	Device      string `point:"device"`

	// simulation source
	SimLatitude    float64 `point:"simLatitude"`
	SimLongitude   float64 `point:"simLongitude"`
	SimSpeed       float64 `point:"simSpeed"`
	SimHeading     float64 `point:"simHeading"`
	SimHeadingRate float64 `point:"simHeadingRate"`
	Period         float64 `point:"period"`

	// status / output
	Latitude        float64 `point:"latitude"`
	Longitude       float64 `point:"longitude"`
	Altitude        float64 `point:"altitude"`
	Speed           float64 `point:"speed"`
	Heading         float64 `point:"heading"`
	FixType         int     `point:"fixType"`
	FixQuality      int     `point:"fixQuality"`
	GPSTime         float64 `point:"gpsTime"`
	NumSat          int     `point:"numSat"`
	HDOP            float64 `point:"hdop"`
	Connected       bool    `point:"connected"`
	Rx              int     `point:"rx"`
	RxReset         bool    `point:"rxReset"`
	ErrorCount      int     `point:"errorCount"`
	ErrorCountReset bool    `point:"errorCountReset"`
}
```

`Baud` is a string to match `SerialDev`, which lets the UI present it as a text
field and keeps the two clients consistent.

## Implementation Plan

### Phase 1: Schema and Client Skeleton

**Goal:** A `gps` node can be created, and its client starts, stops, and
restarts cleanly on config changes.

1. Add node and point type constants to `data/schema.go`.
2. Create `client/gps.go` with the `GPS` config struct, `GPSClient`,
   `NewGPSClient`, and the standard `Run`/`Stop`/`Points`/`EdgePoints` methods.
   Follow the `SignalGeneratorClient` shape: an inner goroutine started by
   `Run`, a `chStopSource` channel, and a restart of that goroutine when a
   config point that affects the source arrives.
3. Define the internal `gpsFix` struct that all three sources produce and a
   single `publish(fix gpsFix)` path that converts it to points. Every source
   feeds this one function, which is what keeps the three sources
   indistinguishable downstream.
4. In `publish`, take `now := time.Now()` once and set `Time: now` explicitly on
   every point in the fix. This is load-bearing for the Grafana geomap, not a
   detail — `SendPoints` otherwise stamps each point with its own `time.Now()`,
   leaving latitude and longitude microseconds apart and unjoinable. Add a test
   asserting all points from one fix share a timestamp.
4. Register the manager in `client/DefaultClients`:
   `g.Add(NewManager(nc, NewGPSClient, nil))`.
5. Apply defaults when unset: `Source` to `serial`, `Baud` to `9600`,
   `GpsdAddress` to `localhost:2947`, `Period` to `1`, `SimSpeed` to `10`,
   `SimHeadingRate` to `5`.

Restart triggers: `gpsSource`, `port`, `baud`, `gpsdAddress`, `device`,
`disabled`, `period`, `simLatitude`, `simLongitude`, `simSpeed`, `simHeading`,
`simHeadingRate`. `debug` is applied live without a restart.

**Verify:** `go build ./...`, then create a GPS node through the UI and confirm
the client logs start and stop.

### Phase 2: Simulation Source

**Goal:** A simulated GPS produces a smooth, plausible track.

1. Create `client/gps-sim.go` with a `gpsSim` type holding position, heading,
   and speed, plus a `step(dt time.Duration) gpsFix` method. No NATS, no serial,
   no network.
2. Movement per step:
   - `heading += (rand.Float64()*2 - 1) * headingRate * dt`, wrapped to
     `[0, 360)`.
   - Distance `d = speed * dt`, angular distance `δ = d / 6371000`.
   - Great-circle destination:
     - `lat2 = asin(sin(lat1)·cos(δ) + cos(lat1)·sin(δ)·cos(θ))`
     - `lon2 = lon1 + atan2(sin(θ)·sin(δ)·cos(lat1), cos(δ) − sin(lat1)·sin(lat2))`
   - Normalize longitude to `[-180, 180)`.
3. Emit a full fix each step: `fixType` `3` (3D), `fixQuality` `1` (GPS),
   satellite count in the 8–12 range, HDOP around 0.8–1.5, altitude held near a
   constant with small jitter, `gpsTime` set to now.
4. In `client/gps.go`, drive the simulator from a ticker at `Period` and feed
   each fix to the shared publish path.

**Verify:** `go test -race ./client/` plus a manual run — create a GPS node with
source `sim` and watch latitude and longitude advance in the UI.

**Test:** `client/gps_test.go`

- Stepping due north (heading 0) at a known speed for a known duration moves
  latitude by the expected amount within tolerance.
- Heading stays in `[0, 360)` and longitude in `[-180, 180)` across many steps,
  including a track started near the antimeridian and near a pole.
- A zero heading rate produces a straight track; a nonzero rate produces a
  bounded turn per step.
- Every point published from one simulated fix carries the same timestamp.

### Phase 3: Serial Source

**Goal:** Read NMEA from a receiver on a serial port.

1. Open the port with `go.bug.st/serial` and `serial.Mode{BaudRate: baud}`,
   matching `client/serial.go`. Retry on a timer when the open fails, and use
   `fsnotify` on the port path to reopen on hotplug — the same pattern
   `SerialDevClient` uses.
2. Read line-by-line with a `bufio.Scanner`, parse with `nmea.Parse`, and switch
   on `s.DataType()`:
   - `nmea.TypeGGA` — latitude, longitude, altitude (MSL), fix quality,
     satellite count, HDOP.
   - `nmea.TypeGSA` — fix type (none / 2D / 3D), HDOP when GGA is absent.
   - `nmea.TypeRMC` — latitude, longitude, speed (knots × 0.514444 → m/s),
     course over ground, UTC date and time; ignore sentences whose validity
     field is not `A`.
   - `nmea.TypeVTG` — speed and course when RMC is absent.
   - Anything else is counted and dropped.

   `nmea.Parse` handles the `GP`/`GL`/`GN`/`GA` talker prefixes, so
   multi-constellation receivers work without extra cases.

3. Accumulate sentences into a `gpsFix` and publish on the GGA or RMC that
   completes it, so one fix produces one coherent set of points rather than a
   dribble of partial updates.
4. Maintain `connected`, `rx`, and `errorCount`, and honor `rxReset` and
   `errorCountReset` the way `SerialDevClient` does.
5. Log raw sentences at `debug >= 4` and parse failures at `debug >= 2`.

**Verify:** `go build ./... && go test -race ./client/`, then a bench test
against a USB GPS receiver if one is available.

**Test:** Factor sentence handling into a pure
`func parseNMEA(line string, fix *gpsFix) (bool, error)` so tests can feed
canned sentences without a port:

- A known GGA sentence yields the expected latitude, longitude, altitude, and
  `fixQuality`.
- A known GSA sentence yields the expected `fixType`.
- A known RMC sentence yields the expected speed in m/s and heading.
- An RMC sentence with validity `V` is rejected.
- A corrupt sentence and a bad checksum both return an error rather than
  publishing a partial fix.
- A receiver stream with no GSA leaves `fixType` unpublished.

For an end-to-end port test, reuse the `test.NewFifoB` approach that
`client/serial.go` takes when `Port == "serialfifo"`, so a test harness can
write NMEA into a FIFO.

### Phase 4: gpsd Source

**Goal:** Subscribe to a running gpsd and publish the same points.

1. Create `client/gps-gpsd.go`. Dial `GpsdAddress` (default `localhost:2947`)
   with a timeout.
2. On connect, gpsd sends a `VERSION` object and then a `DEVICES` object. Log
   the release string at `debug >= 2` — useful when diagnosing field differences
   between gpsd versions. Then write the watch command:
   - `?WATCH={"enable":true,"json":true}`, or
   - `?WATCH={"enable":true,"json":true,"device":"/dev/ttyUSB0"}` when the
     `device` point is set.
3. Decode the stream with `json.Decoder` into `json.RawMessage`, read the
   `class` field, and unmarshal into the matching struct:
   - **TPV** — `mode`, `status`, `time`, `lat`, `lon`, `altMSL`, `altHAE`,
     `alt`, `speed`, `track`. All numeric fields are `*float64` (`*int` for
     `mode` and `status`) so absent is distinguishable from zero.
   - **SKY** — `hdop`, `uSat` (satellites used), `nSat` (satellites seen), and
     the `satellites` array as a fallback for older gpsd versions that omit
     `uSat`, counting entries with `used: true`.
   - Any other class is ignored.
4. Altitude: prefer `altMSL`, fall back to `altHAE`, and use the deprecated
   `alt` only if neither is present. gpsd deprecated `alt` because it was
   ambiguous between the two datums; older gpsd builds still send only `alt`,
   and this ordering handles both. Note in the docs that `altHAE` is height
   above the WGS84 ellipsoid and will differ from the serial source's MSL
   altitude by the local geoid separation, tens of meters in places.
5. Speed is already m/s and `track` is already degrees true — no conversion,
   unlike the NMEA path.
6. Map `mode` and `status` through the normalization tables above.
7. Reconnect on EOF or error using `ExpBackoff` from `client/backoff.go`, capped
   at a minute. Publish `connected` false on disconnect and true on a successful
   watch.
8. Run a stale-fix watchdog: if no TPV arrives for a configurable multiple of
   the expected interval, publish `connected` false. gpsd holds the TCP
   connection open when the receiver goes quiet or is unplugged, so a live
   socket is not evidence of a live fix.

**Verify:** `go build ./... && go test -race ./client/`, then against a real
gpsd. `gpspipe -w` is useful for capturing a reference stream, and `gpsfake` can
replay a recorded NMEA log through a real gpsd instance without hardware.

**Test:** Factor decoding into a pure
`func decodeGpsd(msg json.RawMessage, fix *gpsFix) (bool, error)`:

- A captured TPV object with `mode: 3` yields the expected position, speed in
  m/s, heading, and `fixType` `3`.
- A TPV with `status: 3` (gpsd RTK Fixed) yields `fixQuality` `4`, exercising the
  fact that the two encodings differ and the mapping is not an identity.
- A TPV omitting `speed` leaves the previous speed untouched rather than
  publishing zero.
- A TPV at `lat: 0, lon: 0` publishes that position rather than treating it as
  absent.
- A TPV with only `alt` (old gpsd) populates altitude; one with both `altMSL`
  and `altHAE` prefers `altMSL`.
- A SKY object yields the expected HDOP and satellite count, including the
  `satellites` array fallback when `uSat` is absent.
- VERSION and DEVICES objects are ignored without error.

Capture a real gpsd session with `gpspipe -w` into a testdata file so these run
against genuine output rather than hand-written JSON.

### Phase 5: Frontend

**Goal:** GPS nodes are creatable and configurable in the web UI.

1. `frontend/src/Api/Point.elm` — add and expose `typeGpsSource`,
   `typeLatitude`, `typeLongitude`, `typeAltitude`, `typeSpeed`, `typeHeading`,
   `typeFixType`, `typeFixQuality`, `typeNumSat`, `typeHdop`, `typeGpsdAddress`,
   `typeSimLatitude`, `typeSimLongitude`, `typeSimSpeed`, `typeSimHeading`,
   `typeSimHeadingRate`.
2. `frontend/src/Api/Node.elm` — add and expose `typeGps = "gps"`.
3. `frontend/src/UI/Icon.elm` — add `mapPin = icon FeatherIcons.mapPin`.
4. `frontend/src/Components/NodeGps.elm` — new component modeled on
   `NodeSignalGenerator.elm`:
   - Collapsed summary: icon, description, current latitude and longitude
     rounded to five decimal places, and `(disabled)` when disabled.
   - Expanded detail: description, disabled checkbox, source radio (Serial /
     gpsd / Simulated), then the fields for the selected source:
     - serial — port, baud
     - gpsd — address, device (blank watches all devices)
     - sim — start latitude, start longitude, speed, heading, heading rate,
       period
   - Read-only status: altitude, speed, heading, fix type, fix quality,
     satellites, HDOP, connected, rx count, error count. Fix type and fix
     quality are stored numerically, so the component renders them as labels
     (`3D`, `RTK fixed`) rather than showing bare integers.
5. `frontend/src/Pages/Home_.elm` — import the component, add the `"gps" ->`
   branch to the view dispatch, add `nodeDescGps`, add a `nodeCustomSortRules`
   entry, and add `Input.option Node.typeGps nodeDescGps` to the device and
   group add-node lists.

**Verify:** `siot_build_frontend`,
`cd frontend && npx elm-review && npx elm-test`.

### Phase 6: Documentation and Cleanup

1. `docs/user/gps.md` — new page following `docs/user/signal-generator.md`:
   overview, a section per source, the published point list with units, the fix
   normalization tables from this plan (they are the most useful thing to hand a
   user comparing sources), and a YAML node export example. Call out the MSL
   versus HAE altitude difference between the serial and gpsd sources.
2. `docs/user/gps.md` — add a "Plotting a track in Grafana" section with the
   Flux `pivot()` query for InfluxDB and the Join by field / Organize fields
   transformation sequence for VictoriaMetrics, plus Grafana value mappings that
   turn the numeric fix codes back into labels. Verify both paths against a live
   Grafana before writing them up, including whether the dotted `node.id` tag
   needs `-usePromCompatibleNaming` on the VictoriaMetrics side.
3. `docs/user/database.md` — extend the existing VictoriaMetrics section to note
   that non-numeric field values are stored as 0, which is why string points do
   not survive there. This currently affects any client publishing text points,
   not just GPS.
4. `SUMMARY.md` — add `- [GPS](docs/user/gps.md)` under Clients.
5. `CHANGELOG.md` — entry under `## Next` describing the GPS client and its
   three sources.
6. `plans/plans.md` — mark this plan complete.
7. Retire the legacy `gps/` package. It is dead code superseded by this client,
   but it is exported from a public Go module, so removing it is an API break
   for any downstream importer. Recommendation: delete `gps/` and `data/gps.go`
   and call it out in the changelog as a breaking change. If that is too
   aggressive for this release, add deprecation comments pointing at the new
   client and remove them in the next major version instead. This step is
   deliberately separable from the rest of the plan.

**Verify:** `siot_test`.

## Files Touched

**New**

- `client/gps.go`
- `client/gps-sim.go`
- `client/gps-gpsd.go`
- `client/gps_test.go`
- `client/testdata/gpsd-session.json`
- `frontend/src/Components/NodeGps.elm`
- `docs/user/gps.md`

**Modified**

- `data/schema.go`
- `client/client.go`
- `frontend/src/Api/Point.elm`
- `frontend/src/Api/Node.elm`
- `frontend/src/UI/Icon.elm`
- `frontend/src/Pages/Home_.elm`
- `docs/user/database.md`
- `SUMMARY.md`
- `CHANGELOG.md`
- `plans/plans.md`

**Removed (Phase 6, optional)**

- `gps/gps.go`
- `data/gps.go`

## Open Questions

- Should the simulation source publish `fixQuality` as `1` (GPS) or `8`
  (simulated)? The plan uses `1` so the simulator is a faithful stand-in — the
  point of simulation mode is that downstream logic behaves identically. The
  node's `gpsSource` point already makes the mode visible to anyone inspecting
  the node. If simulated position could ever reach a real actuator, `8` is the
  safer default, and both NMEA and gpsd already define that value.
- Is the two-point fix representation (`fixType` plus `fixQuality`) worth the
  complexity, or is a single `fixType` point enough? The split preserves RTK
  fixed versus float, which matters only if centimeter-accuracy receivers are in
  scope.
- Should a stationary receiver publish position points at every fix, or only
  when the position moves beyond a threshold? The plan publishes every fix so
  the track has continuous timestamps; a `minDistance` point could be added
  later if store growth becomes a concern.
- Is a map view in the UI wanted? Out of scope here — the points are enough for
  Grafana or a rule to consume, and a map component is a larger frontend effort.
- Should the db client omit the `text` field when a point has no string value?
  It would halve the series count VictoriaMetrics stores for every SIOT
  deployment, not just GPS ones. Out of scope for this plan, but the GPS work is
  what surfaced it.
