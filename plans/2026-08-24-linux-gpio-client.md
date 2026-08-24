# Plan: Linux GPIO Client

**Branch:** current checkout **Branched from:** `179d06ea`

## Context

Simple IoT can read a 1-Wire temperature sensor, poll a Modbus register, and
talk to a Shelly relay, but it cannot read a switch wired to a pin or drive a
relay from one. A GPIO line is the simplest IO a Linux board offers, and on the
edge devices Simple IoT targets — Raspberry Pi, BeagleBone, i.MX and TI-based
industrial gateways — it is usually the first thing someone wants to connect: a
door switch, a float switch, a pump enable, a status LED, an alarm relay.

Everything downstream of the point already works. A rule condition can watch a
`value` point and a rule action can write a `valueSet` point, the db client
records both, and the UI graphs them. What is missing is the client that turns a
kernel GPIO line into those points.

The Linux kernel offers two interfaces for this. The `/sys/class/gpio` sysfs
interface is deprecated, has no debounce, and requires polling for edges. The
GPIO character device (`/dev/gpiochipN`) replaced it: it is the supported
interface, it delivers edge events with kernel timestamps, and version 2 of its
ioctl API adds hardware debounce, bias (internal pull-up and pull-down), and
drive configuration. This client uses the character device.

## Design Decisions

**One `gpio` node per line, not a chip node with line children.** Modbus and
1-Wire model a bus node with IO children because the children share a transport,
a poll timer, and an error budget, and cannot act alone. GPIO lines share none
of that — each line is an independent request on `/dev/gpiochipN` with its own
file descriptor, its own edge stream, and its own debounce setting. The chip is
just a string on the node.

Modeling each line as its own node has two concrete advantages. The first is
where the node lives: a `gpio` node goes next to the thing it controls — the
pump enable inside the pump group, the door switch inside the door group —
rather than under a heading that mirrors the board instead of the system. The
second matters more in practice. `client.Manager` stops and rebuilds a client
whenever a child node is added or removed, so with a chip parent, adding a ninth
line would release every line request on that chip; releasing an output line
returns it to input, so a relay or enable line would drop while someone edits
configuration in the UI. With one node per line, editing one line disturbs only
that line.

**Lines are added deliberately, not detected.** A 1-Wire bus enumerates the
sensors on it, and creating a node for each one saves real work. A GPIO chip
exposes every line the SoC has — 54 on a Raspberry Pi, over 100 on some SoCs —
almost all of them unrelated to the application or already claimed by a driver.
So there is no detection pass and no chip listing. A person adds a `gpio` node
for each line they want, which is also the set of lines worth showing in the UI.

What discovery does exist happens on the node itself: `line` accepts either a
line offset or the kernel's name for the line, and the client publishes back the
resolved offset and name. When the request fails because another driver holds
the line, the error point names the consumer that holds it, which is the
question a chip listing would have been used to answer.

**`valueSet` commands an output, `value` reports the line.** This follows the
Modbus IO convention. A rule action, the UI, or an upstream instance writes
`valueSet`; the client drives the line, reads it back, and publishes `value`.
Input lines publish `value` only. Keeping the two separate means the client's
own report of the line state can never be mistaken for a command, and a failed
write leaves `valueSet` and `value` visibly disagreeing.

**Edge events by default, polling when asked.** An input line is requested with
both edges and an event handler, so a state change reaches the point stream in
about a millisecond with no poll timer running. Some lines cannot do this —
expanders behind an I2C bridge, chips still on uAPI v1, kernels without
interrupt support for the pin — so setting `pollPeriod` to a non-zero value
switches the line to a plain periodic read instead. The initial value is read
and published at request time either way, because edge events only report
changes.

**`go-gpiocdev` for the character device.**
[`github.com/warthog618/go-gpiocdev`](https://github.com/warthog618/go-gpiocdev)
v0.9.1 is pure Go over the chardev ioctls, with edge events, debounce, and bias
support, and it negotiates uAPI v2 with a v1 fallback. Its only non-test
dependency is `golang.org/x/sys`, which Simple IoT already carries, so it
cross-compiles to ARM with the rest of the binary and adds very little to the
binary size. periph.io would bring a board abstraction layer this client has no
use for.

**A simulated chip makes the client testable without hardware.** Setting `chip`
to `sim` gives the node an in-memory line instead of a kernel one. Sim lines are
kept in a process-wide registry keyed by offset, and writing a sim output
delivers an edge event to every sim input requested at the same offset — a
virtual wire. That is enough to test the whole path end to end inside
`server.TestServer`: a rule writes `valueSet` on an output node, the sim line
changes, an input node on the same offset sees the edge, and its `value` point
appears on the bus. The GPS client uses the same approach with its `sim` source.

**The chardev code is Linux-only; the client is not.** `go-gpiocdev` builds only
on Linux, so the request path lives in `client/gpio-cdev.go` behind
`//go:build linux`, with `client/gpio-cdev_other.go` returning a clear error on
other platforms, following `client/network-manager_nonlinux.go`. The sim path
builds everywhere, so the tests run on every platform.

**Counting and PWM are out of scope for this plan.** Pulse counting for flow and
energy meters is a natural extension of the edge handler and is sketched as a
follow-on phase below. PWM output uses a different kernel interface
(`/sys/class/pwm`) and belongs in its own client.

## Phase 1 — GPIO Client

New files: `client/gpio.go`, `client/gpio-cdev.go`, `client/gpio-cdev_other.go`,
`client/gpio-sim.go`, `client/gpio_test.go`.

### Config type

`data.ToCamelCase` lowercases an all-uppercase prefix, so the Go type `GPIO`
produces the node type `gpio`, the same way `GPS` produces `gps`.

```go
// GPIO describes a single line on a Linux GPIO character device. A node is
// added for each line the application uses; lines are not detected, because a
// chip exposes far more lines than any one application cares about.
type GPIO struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Disabled    bool   `point:"disabled"`
	Debug       int    `point:"debug"`

	// Line selection. Chip is a chip name ("gpiochip0"), a chip label, or a
	// full device path. Line is a line offset ("17") or the kernel's name for
	// the line ("GPIO17", "PIN_12").
	Chip string `point:"chip"`
	Line string `point:"line"`

	// Line configuration
	Direction    string `point:"direction"`
	Bias         string `point:"bias"`
	Drive        string `point:"drive"`
	ActiveLow    bool   `point:"activeLow"`
	Debounce     int    `point:"debounce"`
	InitialValue bool   `point:"initialValue"`
	PollPeriod   int    `point:"pollPeriod"`

	// State
	Value    bool `point:"value"`
	ValueSet bool `point:"valueSet"`

	// Status
	Connected       bool   `point:"connected"`
	LineOffset      int    `point:"lineOffset"`
	LineName        string `point:"lineName"`
	Error           string `point:"error"`
	ErrorCount      int    `point:"errorCount"`
	ErrorCountReset bool   `point:"errorCountReset"`
}
```

| Point          | Values                                              | Meaning                                                     |
| -------------- | --------------------------------------------------- | ----------------------------------------------------------- |
| `chip`         | `gpiochip0`, a chip label, a path, or `sim`         | Which GPIO chip the line is on                              |
| `line`         | offset or kernel line name                          | Which line on that chip                                     |
| `direction`    | `input` (default), `output`                         | Line direction                                              |
| `bias`         | empty (as-is), `pullUp`, `pullDown`, `biasDisabled` | Internal bias, inputs mainly                                |
| `drive`        | empty (push-pull), `openDrain`, `openSource`        | Output drive                                                |
| `activeLow`    | bool                                                | Invert: a low line reads and drives as active               |
| `debounce`     | ms                                                  | Kernel debounce for edge events, inputs only                |
| `initialValue` | bool                                                | Value driven at request time, outputs only                  |
| `pollPeriod`   | ms                                                  | Non-zero switches an input from edge events to polling      |
| `valueSet`     | bool                                                | Requested output state                                      |
| `value`        | bool                                                | Line state as read back, published by the client            |
| `connected`    | bool                                                | The line is currently requested                             |
| `lineOffset`   | number                                              | Resolved offset, useful when `line` was given as a name     |
| `lineName`     | text                                                | Kernel name for the resolved line                           |
| `error`        | text                                                | Why the last request or access failed, empty when connected |

`description`, `disabled`, `debug`, `pollPeriod`, `value`, `valueSet`,
`initialValue`, `connected`, `error`, `errorCount`, and `errorCountReset` are
existing point types. `data/schema.go` gains `NodeTypeGPIO` and the new point
types and values:

```go
NodeTypeGPIO = "gpio"

PointTypeChip       = "chip"
PointTypeLine       = "line"
PointTypeLineOffset = "lineOffset"
PointTypeLineName   = "lineName"

PointTypeDirection = "direction"
PointValueInput    = "input"
PointValueOutput   = "output"

PointTypeBias         = "bias"
PointValuePullUp      = "pullUp"
PointValuePullDown    = "pullDown"
PointValueBiasDisabled = "biasDisabled"

PointTypeDrive       = "drive"
PointValuePushPull   = "pushPull"
PointValueOpenDrain  = "openDrain"
PointValueOpenSource = "openSource"

PointTypeActiveLow = "activeLow"
PointTypeDebounce  = "debounce"

PointValueSim = "sim"
```

### Line abstraction

The client talks to one small interface, which both the chardev and the sim
implement. Keeping it this narrow is what lets the tests run on any platform.

```go
// gpioLine is the part of a requested GPIO line this client uses.
type gpioLine interface {
	Value() (bool, error)
	SetValue(bool) error
	Close() error
}

// gpioLineConfig is a resolved request, built from the node config.
type gpioLineConfig struct {
	Chip      string
	Offset    int
	Output    bool
	Bias      string
	Drive     string
	ActiveLow bool
	Debounce  time.Duration
	Initial   bool
	// Edges is nil when the line is polled rather than edge-driven.
	Edges chan<- bool
}

// gpioRequest resolves the chip and line named in the config, requests the
// line, and returns it along with its resolved offset and kernel name.
func gpioRequest(cfg gpioLineConfig) (gpioLine, gpioLineInfo, error)
```

`client/gpio-cdev.go` (`//go:build linux`) maps `gpioLineConfig` onto
`gpiocdev.RequestLine`: `gpiocdev.AsInput` or `gpiocdev.AsOutput(v)`,
`WithPullUp` / `WithPullDown` / `WithBiasDisabled`, `AsOpenDrain` /
`AsOpenSource`, `AsActiveLow`, `WithDebounce`, `WithConsumer("simpleiot")`, and
for an edge-driven input `WithBothEdges` plus a `WithEventHandler` that converts
`gpiocdev.LineEvent` to a bool and sends it non-blocking on `Edges`. Resolving
the line uses `chip.FindLine(name)` when `line` is not a number, and
`chip.LineInfo(offset)` supplies the name and, when a request fails with
`EBUSY`, the consumer holding it. `client/gpio-cdev_other.go`
(`//go:build !linux`) returns an error naming the platform.

`client/gpio-sim.go` builds everywhere and keeps a package-level registry of sim
lines keyed by offset, guarded by a mutex. `SetValue` on a sim line records the
value and delivers an edge to every other sim line registered at that offset
with an `Edges` channel. Debounce is ignored by the sim, since it is a kernel
behavior.

### Client structure

`GPIOClient` follows the shape of the other clients: `nc`, `config`, `stop`,
`newPoints`, `newEdgePoints`, plus the runtime state that does not belong in the
config — the open line, the edge channel, and the timers.

`Run` requests the line once and then selects over the stop channel, the two
point channels, the edge channel, the poll timer, a retry timer, and a refresh
timer:

- **Request.** Build a `gpioLineConfig` from the config and call `gpioRequest`.
  On success, publish `connected` true, `lineOffset`, `lineName`, an empty
  `error`, and the value read back from the line. On failure, publish
  `connected` false and the error text, count the error, and arm the retry timer
  using the existing `client.Backoff` helper — a line held by a driver that has
  not loaded yet should recover on its own.
- **Configuration points.** `data.MergePoints` applies the update, then any of
  `chip`, `line`, `direction`, `bias`, `drive`, `activeLow`, `debounce`,
  `initialValue`, `pollPeriod`, or `disabled` closes the line and requests it
  again. `gpiocdev` can reconfigure a line in place, but re-requesting is one
  path instead of two and covers a changed chip or line as well.
- **`valueSet`.** For an output, drive the line, read it back, and publish
  `value`. For an input, publish an error rather than silently ignoring it.
- **Edge events.** Publish `value`.
- **Poll timer.** Runs only when `pollPeriod` is non-zero: read the line, and
  publish `value` when it differs from the last published state.
- **Refresh timer.** Republishes `value` every ten minutes even when unchanged,
  so a graph or an upstream instance has a recent sample. This is what the
  1-Wire client does with `oneWireValueRefresh`.
- **`errorCountReset`.** Zeroes `errorCount` and clears the request.

`Stop` closes the stop channel; `Run` closes the line on the way out, which
releases it back to the kernel.

An empty `chip` or `line` is not an error — the client logs once and idles until
a point completes the configuration, the way `SerialDevClient` handles an unset
port. `disabled` closes the line, which is the only way to hand a line back
without deleting the node.

Register `NewGPIOClient` in `client.DefaultClients`.

### Tests

`client/gpio_test.go` follows `client/serial_test.go`, using `server.TestServer`
and `client.NodeWatcher`, with every case on the sim chip so no hardware or root
access is involved.

| Case              | Setup                                        | Assertion                                                          |
| ----------------- | -------------------------------------------- | ------------------------------------------------------------------ |
| Output write      | one output node, sim offset 1                | `valueSet` true drives the sim line and `value` true is published  |
| Input edge        | output and input nodes on sim offset 1       | toggling the output publishes `value` true then false on the input |
| Initial value     | output node with `initialValue` true         | the line reads true as soon as it is requested                     |
| Active low        | output node with `activeLow`                 | the sim line level is inverted, `value` still reports active       |
| Line by name      | sim line registered with a name              | `lineOffset` and `lineName` are published                          |
| Polled input      | input node with `pollPeriod` 20 ms, no edges | a sim line change is picked up within a few poll periods           |
| Request failure   | `chip` set to a chip that does not exist     | `connected` false, `error` set, `errorCount` increments, retries   |
| Reconfigure       | input node switched to output at runtime     | the line is re-requested and `valueSet` then works                 |
| Write to an input | `valueSet` on an input node                  | `error` is set and the line is unchanged                           |
| Disable           | `disabled` true on a connected node          | `connected` goes false and the sim line is released                |

A rule test is worth adding to the same file: a rule whose condition watches the
input node and whose action writes `valueSet` on the output node, wired through
the sim to confirm the whole loop a person would actually build.

## Phase 2 — Frontend

- `frontend/src/Api/Node.elm` — add and export `typeGpio`.
- `frontend/src/Api/Point.elm` — add and export the new point type and value
  constants from Phase 1.
- `frontend/src/Components/NodeGpio.elm` — new component, patterned on
  `NodeModbusIO.elm`. Collapsed, it shows the icon, the description, the current
  value, and `(disabled)` when disabled. Expanded, it shows description, chip,
  line, a `direction` option input, `bias` and `drive` option inputs,
  `activeLow` and `disabled` checkboxes, `debounce` and `pollPeriod` number
  inputs, an on/off input for `valueSet` on outputs, the resolved line name, the
  error text when it is non-empty, and the error counter with reset.
- `frontend/src/Pages/Home_.elm` — import the component, add the `"gpio"` case
  to the view dispatch, add a `nodeCustomSortRules` entry, add `nodeDescGpio`,
  and offer the type under both the device and group parents in `viewAddNode`.
- `frontend/src/UI/Icon.elm` — reuse `Icon.io`, which already reads as an IO
  point, rather than adding an icon.

`gpio` has no children, so `nodeTypesThatHaveChildNodes` is unchanged.

## Phase 3 — Documentation

- `docs/user/gpio.md` — new page covering what the client does, adding a node
  and setting chip and line, the option table from Phase 1, and a schema example
  for `siot import`:

  ```yaml
  nodes:
    - gpio:
        description: Pump enable
        chip: gpiochip0
        line: "17"
        direction: output
        initialValue: 0
    - gpio:
        description: Float switch
        chip: gpiochip0
        line: FLOAT_SW
        direction: input
        bias: pullUp
        debounce: 20
  ```

  Practical notes belong here as well: finding the chip and line with
  `gpiodetect` and `gpioinfo` from libgpiod; that a Raspberry Pi's 40-pin header
  is `gpiochip0` on most kernels and that Pi 5 moved it, which is exactly why
  the chip is configurable; that access to `/dev/gpiochipN` needs group
  membership or a udev rule when Simple IoT does not run as root; that a line
  already claimed by a driver cannot be requested, and the error point names the
  driver holding it; and that `chip: sim` gives a line with no hardware behind
  it for trying out rules.

- `SUMMARY.md` — add the page next to the other client pages.
- `CHANGELOG.md` — an entry under `## Next`.
- `CLAUDE.md` — add GPIO to the list of common client types.

## Phase 4 — Pulse Counting (follow-on)

Flow meters, energy meters, and anemometers all present as a line that pulses,
and the edge handler already sees every pulse with a kernel timestamp. A
`direction` value of `counter` would add:

- `count` — pulses since the counter was last reset, published on a configurable
  interval rather than per pulse, since a flow meter can produce hundreds of
  pulses a second.
- `countReset` — clears the count.
- `rate` — pulses per second over the reporting interval, and with `scale` and
  `units`, engineering units such as liters per minute.

This is deliberately separate: it needs its own thought about reporting
intervals, counter persistence across a restart, and how a rate is averaged, and
none of that should hold up the input and output cases.

PWM output is not a GPIO chardev feature at all — it is `/sys/class/pwm` — and
belongs in a separate client if it is wanted.
