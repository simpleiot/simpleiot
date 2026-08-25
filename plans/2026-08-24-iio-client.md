# Plan: Linux IIO Client

**Branch:** current checkout **Branched from:** `8454edb6`

## Context

The GPIO client gave Simple IoT digital IO on a Linux board: a switch it can
read, a relay it can drive. The analog half is still missing. There is no way to
read a 4-20 mA loop through an ADS1115, a pressure sensor on an I2C bus, a
thermocouple through a MAX31856, or the accelerometer already sitting on a
gateway's board.

Linux exposes all of these through one subsystem. Industrial I/O (IIO) is the
kernel's framework for ADCs, DACs, and the sensors that behave like them —
accelerometers, gyroscopes, magnetometers, pressure, humidity, light, and
proximity. A driver registers a device under `/sys/bus/iio/devices/iio:deviceN`
and publishes one set of attributes per channel. The attribute layout is defined
by the IIO ABI and is the same for every driver, which is what makes a single
client worth writing:

```
/sys/bus/iio/devices/iio:device0/
    name                      ads1015
    sampling_frequency        1600
    in_voltage0_raw           1583
    in_voltage0_scale         2.000000000
    in_voltage0-voltage1_raw  -12
    in_temp_raw               2634
    in_temp_scale             0.062500000
    in_temp_offset            -1092
    out_voltage0_raw          2048
```

The value of a channel is `(raw + offset) * scale`, in units the ABI fixes per
channel type. That formula and that directory layout are the whole read path,
whether the device is a 16-bit ADC or a six-axis IMU.

Everything downstream already works. A `value` point feeds rule conditions, the
db client records it, and the UI graphs it — the same path 1-Wire and Modbus
readings take today. What is missing is the client that turns an IIO channel
into those points.

**Why `iio` and not `adc`.** An `adc` client would read `in_voltage0` and stop
there, and the first accelerometer or pressure sensor would need a second client
that repeated all of the same code against `in_accel_x` or `in_pressure`. The
kernel does not draw that line and neither should we. The name costs some
discoverability with a person who only wants to read a voltage, which the
documentation page and the node description carry instead.

**PWM is a separate subsystem and stays a separate client.** `/sys/class/pwm`
has `export`, `period`, `duty_cycle`, and `enable`, and shares no attribute, no
value formula, and no discovery path with IIO. It is worth building, and it is
not part of this plan.

## Design Decisions

**One `iio` device node with `iioChannel` children.** This is the Modbus and
1-Wire shape rather than the GPIO one, and the reason is what each node maps
onto. A GPIO line is an independent kernel request holding its own file
descriptor, so a line earns a node of its own; adding a node under a chip parent
would have released every other line on that chip when `client.Manager` rebuilt
the client. An IIO channel holds nothing. It is a set of files in a directory
shared with every other channel on the device, alongside device level settings —
`sampling_frequency` and `oversampling_ratio` — that apply to all of them at
once. Restarting the client when a channel is added costs a poll interval and
releases nothing, so the hazard that shaped the GPIO client does not apply.

Reading is better off grouped too. Three axes of an accelerometer belong to one
sample, and one poll of the device that reads all enabled channels is both
simpler and more honest than three pollers arriving at unrelated moments.

**Channels are detected, unlike GPIO lines.** A GPIO chip exposes every line the
SoC has, most of them unrelated to the application, which is why lines are added
by hand. An IIO device exposes exactly the channels it has — typically between
one and eight — and every one of them is something the hardware measures. So the
client globs `in_*_raw` and `out_*_raw` on each poll and creates a child node
for any channel that does not have one yet, the way the 1-Wire client creates a
node per sensor it finds on the bus. A person adds one node naming the device
and then edits descriptions, rather than transcribing attribute names out of
sysfs.

**`_input` first, then `_raw` with `_scale` and `_offset`.** Some drivers
publish an already-converted value in `in_<channel>_input` and no raw attribute
at all. Where it exists the client uses it; otherwise it reads `_raw` and
applies `(raw + offset) * scale`. Both `_scale` and `_offset` are optional — a
missing scale is 1 and a missing offset is 0 — and both may be published per
channel type rather than per channel, so `in_voltage0_scale` falls back to
`in_voltage_scale` before defaulting. This is ABI behavior, not driver quirk
handling, and it belongs in the client rather than in each person's `scale`
point.

**The client normalizes millis to base units.** The IIO ABI specifies millivolts
for voltage, milliamps for current, and millidegrees Celsius for temperature.
Publishing those verbatim would mean a rule comparing a temperature to `25`
works against a 1-Wire sensor and fails against an IIO one, and that a voltage
graph reads `3300`. So the client divides by a thousand for those three channel
types and sets `units` accordingly at detection. Every other type is published
in the ABI unit, which is already a base unit. The conversion table is fixed, is
documented on the user page, and is applied before the node's own `scale` and
`offset`.

**Two layers of scale, and they mean different things.** The kernel's `_scale`
and `_offset` convert a raw count into the ABI's physical unit and are the
driver's business. The node's `scale` and `offset` points convert that physical
unit into the quantity the sensor is actually wired to measure — a 4-20 mA loop
into a tank level, a divided voltage into the battery voltage before the
divider. Keeping them separate means a person editing a node never has to know
the device's full-scale range, and replacing an ADS1115 with an ADS1015 changes
nothing on the node.

**`minChange` keeps ADC noise out of the store.** A 1-Wire temperature is
quantized coarsely enough that an unchanged reading really is unchanged. An
ADC's low bits dither on every sample, so publishing on any change would write a
point per poll forever. A `minChange` point sets how far the value must move
from the last published one before it is sent, with the ten minute refresh
underneath it so a graph and an upstream instance always have a recent sample.
This is new, and it is worth adding here rather than working around it in the db
client.

**Output channels take `valueSet`, following the Modbus and GPIO convention.** A
channel detected as `out_*` gets a `valueSet` point; a rule action or the UI
writes it, the client inverts the conversion chain to a raw count, writes
`out_<channel>_raw`, reads it back, and publishes `value`. Writing `valueSet` on
an input channel publishes an error rather than being ignored.

**No build tag, and a fixture directory instead of a simulator.** The GPIO
client needed `//go:build linux` because chardev ioctls do not exist elsewhere,
and needed a simulated chip because those ioctls cannot be faked with files. IIO
is sysfs and nothing else, so the client is ordinary file IO that compiles
everywhere and simply finds no devices on a machine without the tree. Tests
point an `IIODevicePath` variable at a `t.TempDir()` laid out like the real
thing, the way `OneWireDevicePath` already works. A fixture exercises the real
code path — attribute discovery, the scale fallback, the conversion chain, the
write — where a simulator would only exercise the simulator.

**This client polls sysfs, and that is the whole design.** The assumption
throughout is low rate data: a tank level, a loop current, a board temperature,
a battery voltage, read every second or few seconds and published when it moves.
One `read()` of a sysfs attribute per channel per poll is entirely adequate for
that, and it keeps the client to file IO with no device handle held between
polls, no trigger to configure, and nothing to tear down when a node changes.

The high rate path exists in IIO and this client deliberately does not reach for
it. Capturing a waveform means enabling scan elements, attaching a trigger,
sizing a buffer, and decoding packed binary records from `/dev/iio:deviceN`
against each channel's `_type` descriptor — and then answering the harder
question of what a kilohertz stream should publish, since a point per sample
would overwhelm the store. That is a different client with a different data
model, and folding a `mode` switch into this one now would shape every decision
above around a case we are not building. If the need arrives, it gets its own
plan.

The practical consequence is a bound worth stating: `pollPeriod` below roughly
100 ms is not what this client is for. Nothing enforces it, but a person asking
for faster sampling than that wants the buffered path, and the documentation
should say so rather than let them discover it as jitter.

## Phase 1 — IIO Client

New files: `client/iio.go`, `client/iio-sysfs.go`, `client/iio_test.go`.

### Config types

`data.ToCamelCase` lowercases an all-uppercase prefix, so `IIO` produces the
node type `iio` and `IIOChannel` produces `iioChannel`.

```go
// IIODevicePath is the root of the Linux IIO sysfs tree. It is a variable so
// that tests can point it at a fixture directory; nothing else should change
// it.
var IIODevicePath = "/sys/bus/iio/devices"

// IIO describes one Linux Industrial I/O device: an ADC, a DAC, or a sensor
// the kernel presents through the same interface. A device node is added by
// the person configuring the system, who sets the device name. The channels on
// that device are then detected and added as children.
type IIO struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Debug       int    `point:"debug"`

	// Device selects the IIO device: the driver's name for it ("ads1015"),
	// the sysfs directory name ("iio:device0"), or a full path. Matching by
	// name is preferred, because the device number depends on probe order and
	// is not stable across boots.
	Device string `point:"device"`

	PollPeriod int `point:"pollPeriod"`

	// Device level settings, written to sysfs when set and left alone when
	// empty. Both still matter for a polled client: many ADCs convert
	// continuously and a sysfs read returns the most recent conversion, so
	// the sample frequency decides how stale a reading can be, and the
	// oversampling ratio trades conversion time for noise.
	SampleFrequency float64 `point:"sampleFrequency"`
	Oversampling    int     `point:"oversampling"`

	// Status
	Connected       bool   `point:"connected"`
	DeviceName      string `point:"deviceName"`
	DevicePath      string `point:"devicePath"`
	Error           string `point:"error"`
	ErrorCount      int    `point:"errorCount"`
	ErrorCountReset bool   `point:"errorCountReset"`

	Channels []IIOChannel `child:"iioChannel"`
}

// IIOChannel describes one channel on an IIO device.
type IIOChannel struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`

	// Channel is the sysfs attribute prefix, without the trailing _raw:
	// "in_voltage0", "in_voltage0-voltage1" for a differential pair,
	// "in_accel_x", "in_temp", "out_voltage0". Filled in by detection.
	Channel string `point:"channel"`
	// ChannelType is the measured quantity parsed out of the channel name:
	// "voltage", "current", "temp", "accel", and so on. It selects the
	// default units and the milli conversion.
	ChannelType string `point:"channelType"`
	// Direction is "input" for an in_* channel and "output" for an out_*
	// channel, reusing the GPIO client's values. Only an output accepts
	// valueSet.
	Direction string `point:"direction"`

	// Conversion applied after the kernel's own scale and offset
	Scale     float64 `point:"scale"`
	Offset    float64 `point:"offset"`
	Units     string  `point:"units"`
	MinChange float64 `point:"minChange"`

	Value    float64 `point:"value"`
	ValueSet float64 `point:"valueSet"`

	Error           string `point:"error"`
	ErrorCount      int    `point:"errorCount"`
	ErrorCountReset bool   `point:"errorCountReset"`
}
```

Device node points:

| Point             | Values                                   | Meaning                                                        |
| ----------------- | ---------------------------------------- | -------------------------------------------------------------- |
| `device`          | `ads1015`, `iio:device0`, or a full path | Which IIO device                                               |
| `pollPeriod`      | ms, default 3000                         | How often every enabled channel is read                        |
| `sampleFrequency` | Hz                                       | Written to `sampling_frequency` when non-zero                  |
| `oversampling`    | count                                    | Written to `oversampling_ratio` when non-zero                  |
| `connected`       | bool                                     | The device was found and is readable                           |
| `deviceName`      | text                                     | The device's `name` attribute, useful when `device` was a path |
| `devicePath`      | text                                     | The resolved `iio:deviceN` directory                           |
| `error`           | text                                     | Why the device could not be resolved or read                   |

Channel node points:

| Point         | Values                                      | Meaning                                                   |
| ------------- | ------------------------------------------- | --------------------------------------------------------- |
| `channel`     | `in_voltage0`, `in_accel_x`, `out_voltage0` | Attribute prefix, filled in by detection                  |
| `channelType` | `voltage`, `current`, `temp`, `accel`, …    | Measured quantity, filled in by detection                 |
| `direction`   | `input`, `output`                           | Filled in by detection; only an output accepts `valueSet` |
| `scale`       | number, default 1                           | Applied to the converted value                            |
| `offset`      | number, default 0                           | Added after `scale`                                       |
| `units`       | text                                        | Defaulted from `channelType`, editable                    |
| `minChange`   | number                                      | How far the value must move before it is published        |
| `value`       | number                                      | Converted reading, published by the client                |
| `valueSet`    | number                                      | Requested output value, outputs only                      |

`description`, `disabled`, `debug`, `pollPeriod`, `scale`, `offset`, `units`,
`value`, `valueSet`, `connected`, `error`, `errorCount`, and `errorCountReset`
are existing point types, as are `device` and `channel`. `direction` and its
`input` and `output` values came in with the GPIO client and mean the same thing
here, which is why the channel carries a direction rather than an `output`
boolean — an `output` point type would also collide with the existing
`PointValueOutput` string. `data/schema.go` gains:

```go
NodeTypeIIO        = "iio"
NodeTypeIIOChannel = "iioChannel"

PointTypeDeviceName = "deviceName"
PointTypeDevicePath = "devicePath"

PointTypeSampleFrequency = "sampleFrequency"
PointTypeOversampling    = "oversampling"

PointTypeChannelType = "channelType"
PointTypeMinChange   = "minChange"
```

### sysfs layer

`client/iio-sysfs.go` holds everything that touches the filesystem, with no NATS
and no node types in it, so the conversion rules can be tested directly.

```go
// iioDevice is a resolved IIO device directory
type iioDevice struct {
	Path string
	Name string
}

// iioChannelInfo is a channel discovered on a device
type iioChannelInfo struct {
	// Channel is the attribute prefix without _raw, e.g. "in_voltage0"
	Channel string
	// Type is the measured quantity, e.g. "voltage"
	Type string
	// Output is true for an out_* channel
	Output bool
}

// iioFind resolves a device given a name, an "iio:deviceN" directory name, or
// a path, and reads back its name attribute.
func iioFind(root, device string) (iioDevice, error)

// iioChannels lists the channels a device publishes, by globbing *_raw and
// *_input and parsing the prefix.
func iioChannels(dev iioDevice) ([]iioChannelInfo, error)

// iioRead reads one channel and converts it to the ABI unit, preferring
// _input and otherwise applying (raw + offset) * scale with the per channel
// then per type attribute fallback.
func iioRead(dev iioDevice, ch string) (float64, error)

// iioWrite converts an ABI unit value back to a raw count and writes it to
// the channel's _raw attribute.
func iioWrite(dev iioDevice, ch string, v float64) error

// iioWriteAttr writes a device level attribute such as sampling_frequency,
// and reports a missing attribute distinctly so an unsupported setting is not
// counted as an error.
func iioWriteAttr(dev iioDevice, attr string, v string) error
```

Channel name parsing takes `in_voltage0` to type `voltage`, `in_accel_x` to
`accel`, `in_voltage0-voltage1` to `voltage`, and `out_voltage0` to `voltage`
with `Output` true. The type is the alphabetic run after the `in_`/`out_`
prefix, stopping at the first digit or the differential dash.

The unit table, applied in `iioRead` after the ABI conversion:

| Channel type       | ABI unit       | Published as | `units` |
| ------------------ | -------------- | ------------ | ------- |
| `voltage`          | millivolts     | volts        | `V`     |
| `current`          | milliamps      | amps         | `A`     |
| `temp`             | millidegrees C | degrees C    | `C`     |
| `accel`            | m/s²           | as-is        | `m/s^2` |
| `anglvel`          | rad/s          | as-is        | `rad/s` |
| `magn`             | Gauss          | as-is        | `G`     |
| `pressure`         | kilopascals    | as-is        | `kPa`   |
| `humidityrelative` | percent        | as-is        | `%`     |
| `illuminance`      | lux            | as-is        | `lx`    |
| `proximity`        | unitless       | as-is        | empty   |
| anything else      | driver defined | as-is        | empty   |

### Client structure

`IIOClient` follows `OneWireClient` closely: `nc`, `config`, `stop`,
`newPoints`, `newEdgePoints`, the `devicePath` captured at construction so a
test that repoints `IIODevicePath` does not race a running client, a `created`
map so a channel detected again before its node appears is not added twice, and
a `lastSent` map keyed by channel node ID.

`Run` starts a poll ticker and selects over the stop channel, the two point
channels, and the ticker:

- **Resolve.** On the first poll after a configuration change, call `iioFind`.
  On success publish `connected` true, `deviceName`, `devicePath`, and an empty
  `error`, and write `sampling_frequency` and `oversampling_ratio` when they are
  set. On failure publish `connected` false and the error text and count it; the
  next tick tries again, which covers a device whose driver has not probed yet.
- **Device points.** `pollPeriod` resets the ticker. `device` clears the
  resolution and re-resolves. `sampleFrequency` and `oversampling` are written
  through. `errorCountReset` zeroes the count.
- **Channel points.** `valueSet` on an output channel converts and writes, reads
  back, and publishes `value`; on an input channel it publishes an error.
  `errorCountReset` zeroes that channel's count. `scale`, `offset`, and `units`
  only change how the next reading is converted, so they need no side effect.
- **Poll tick.** Skip when disabled. Detect channels, then read every enabled
  channel, apply the node's `scale` and `offset`, and publish `value` when it
  has moved by at least `minChange` or when `iioValueRefresh` has elapsed since
  the last publish. A failed read counts an error against both the device and
  the channel, the way `OneWireClient.logError` does.

`Stop` closes the stop channel. There is nothing to release on the way out,
since a sysfs read holds no handle between polls.

An empty `device` is not an error — the client logs once and idles until a point
sets it. `disabled` on the device stops polling; `disabled` on a channel skips
that channel and leaves the rest.

Register `NewIIOClient` in `client.DefaultClients`.

### Tests

`client/iio_test.go` has two layers. The sysfs functions are tested directly
against a fixture tree, and the client is tested through `server.TestServer` and
`client.NodeWatcher` the way `client/onewire_test.go` and `client/gpio_test.go`
are. A `writeIIO` helper lays out a device directory from a map of attribute
names to contents, mirroring `writeW1`.

| Case                  | Setup                                                    | Assertion                                                           |
| --------------------- | -------------------------------------------------------- | ------------------------------------------------------------------- |
| Resolve by name       | two fixture devices, `device` set to the second's name   | `devicePath` names the right directory, `connected` true            |
| Resolve by dir        | `device` set to `iio:device0`                            | resolves, `deviceName` published from the `name` attribute          |
| Missing device        | `device` names nothing present                           | `connected` false, `error` set, `errorCount` increments, retries    |
| Channel detection     | device with three `in_*_raw` and one `out_*_raw`         | four child nodes appear with `channel`, `channelType`, `direction`  |
| Detection is once     | poll several times                                       | no duplicate children                                               |
| Raw conversion        | `in_voltage0_raw` 1583, `_scale` 2, no offset            | `value` is 3.166 volts                                              |
| Offset applied        | `in_temp_raw` 2634, `_scale` 0.0625, `_offset` -1092     | `value` is the millidegree result divided by a thousand             |
| Scale fallback        | `in_voltage_scale` only, no per channel scale            | both voltage channels use it                                        |
| Missing scale         | `_raw` only                                              | scale 1, offset 0                                                   |
| `_input` preferred    | both `_input` and `_raw` present with different values   | the `_input` value is published                                     |
| Node scale and offset | node `scale` 25, `offset` -100 on a 4-20 mA channel      | `value` is the engineering value                                    |
| minChange             | `minChange` 0.5, fixture value moved by 0.1 then by 1.0  | only the second change publishes                                    |
| Refresh               | value unchanged past `iioValueRefresh`                   | `value` is republished                                              |
| Output write          | `valueSet` 1.5 on an `out_voltage0` channel with scale 2 | `out_voltage0_raw` contains 750                                     |
| Write to an input     | `valueSet` on an `in_*` channel                          | `error` is set and no attribute is written                          |
| Device attributes     | `sampleFrequency` 100 set on the device node             | `sampling_frequency` contains 100                                   |
| Unsupported attr      | device with no `oversampling_ratio`                      | setting it is reported, not counted as a read error                 |
| Read failure          | `_raw` attribute removed after detection                 | channel `error` set, both error counts increment, others still read |
| Disable               | `disabled` on one channel                                | that channel stops publishing, the others continue                  |

A rule test belongs here too, matching the one in `client/gpio_test.go`: a rule
whose condition watches an input channel's `value` and whose action writes
`valueSet` on an output channel, run against the fixture tree to confirm the
loop a person would build.

## Phase 2 — Frontend

- `frontend/src/Api/Node.elm` — add and export `typeIio` and `typeIioChannel`.
- `frontend/src/Api/Point.elm` — add and export the new point types from
  Phase 1.
- `frontend/src/Components/NodeIio.elm` — new component, patterned on
  `NodeOneWire.elm`. Collapsed, it shows the icon, the description, the resolved
  device name, and `(disabled)` when disabled. Expanded, it shows description,
  device, poll period, sample frequency, oversampling, the resolved device path,
  the error text when non-empty, and the error counter with reset.
- `frontend/src/Components/NodeIioChannel.elm` — new component, patterned on
  `NodeOneWireIO.elm` with the value editing from `NodeModbusIO.elm`. Collapsed,
  it shows the description and the current value with units. Expanded, it shows
  description, the channel name and type as read-only text, scale, offset,
  units, min change, a value input for `valueSet` on output channels, the error
  text, and the error counter with reset.
- `frontend/src/Pages/Home_.elm` — import both components, add the `"iio"` and
  `"iioChannel"` cases to the view dispatch, add `nodeCustomSortRules` entries,
  add `nodeDescIio` and `nodeDescIioChannel`, add `Node.typeIio` to
  `nodeTypesThatHaveChildNodes`, and offer `iio` under the device and group
  parents and `iioChannel` under an `iio` parent in `viewAddNode`.
- `frontend/src/UI/Icon.elm` — reuse `Icon.io`, as the GPIO component does.

Adding a channel by hand is worth offering even though detection covers the
common case: a driver that publishes a channel through `_input` with no `_raw`
still needs a node, and a person who wants only two of a device's eight channels
can delete the rest and re-add one later.

## Phase 3 — Documentation

- `docs/user/iio.md` — new page covering what the client does, the device types
  it reaches, adding a device node and finding the device name, channel
  detection, the two layers of scale, the unit table from Phase 1, and a schema
  example for `siot import`:

  ```yaml
  nodes:
    - iio:
        description: Tank level ADC
        device: ads1015
        pollPeriod: 1000
        sampleFrequency: 128
  ```

  Practical notes belong here as well: that
  `cat /sys/bus/iio/devices/iio:device*/name` lists what is present and
  `iio_info` from libiio prints the channels and their attributes; that the
  device number depends on probe order, which is why matching by name is
  preferred; that reading sysfs attributes needs group membership or a udev rule
  when Simple IoT does not run as root; that many ADCs and sensors need a device
  tree overlay or an I2C instantiation before they appear at all, with the
  Raspberry Pi `dtoverlay=ads1015` line as the example; a worked 4-20 mA loop
  showing the sense resistor, the `scale` and `offset` values it produces, and
  the resulting engineering units; that `minChange` is the setting to reach for
  when a channel is writing a point per poll; and that this client is built for
  low rate readings, so a `pollPeriod` much below 100 ms is outside what it is
  meant to do, with `sampleFrequency` and `oversampling` being the settings that
  actually improve a reading rather than a faster poll.

- `SUMMARY.md` — add the page next to the other client pages.
- `CHANGELOG.md` — an entry under `## Next`.
- `CLAUDE.md` — add IIO to the list of common client types.
