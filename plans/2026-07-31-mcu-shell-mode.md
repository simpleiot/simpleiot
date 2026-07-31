# Plan: MCU Serial Client Shell Mode

**Branch:** `feat/mcu-shell` **Branched from:** `3e2e4023`

## Context

The serial client (`client/serial.go`) speaks one protocol today: COBS-framed
packets carrying a sequence byte, a 16-byte subject, a binary point payload, and
a CRC (see [`docs/ref/serial.md`](../docs/ref/serial.md)). It is compact and
efficient, and it works well when the MCU firmware is built to speak it.

It also has a cost. The link is opaque. A developer cannot open `tio` on the
port and see what is happening, cannot type a value to test a behavior, and
cannot tell a framing error from a wiring error without attaching SIOT and
turning on debug level 8. Bringing up a new board means getting the binary
protocol working before you can observe anything at all.

Meanwhile the Zephyr firmware tree at `/scratch/simpleiot/zephyr-siot/siot`
already models everything as points, and already exposes them on a shell. Every
subsystem publishes to a zbus `point_chan`, and `lib/point.c` registers a
`p <type> <key> <INT|FLT|STR> <data>` command that publishes a point from the
console. What is missing is the other direction: nothing streams points out.

The `apps/siot-serial` application in that tree is a twelve-line stub, so there
is no working MCU implementation of the binary protocol to talk to at all. Shell
mode is therefore not just a debugging convenience, it is the shortest path from
this firmware tree to a live SIOT link.

This plan adds a second mode to the serial client that talks to that shell
directly. The MCU emits every point it publishes as a line of ASCII on the
console, and SIOT writes points back using the `p` command that already exists.
Everything on the wire is human-readable, so the same link a developer uses for
debugging is the link SIOT uses for data. A debug level on the node mirrors the
whole console to the SIOT server log, so attaching SIOT to a board does not cost
the visibility a `tio` session would have given.

## Bring-Up Target

The first target is `apps/siot-net` on an **ST Nucleo-H743ZI** development
board. It was chosen because the whole path is already in place:

- The board's console and shell are both on USART3
  (`zephyr/boards/st/nucleo_h743zi/nucleo_h743zi.dts`), which routes through the
  on-board ST-LINK virtual COM port. One USB cable carries flashing, debugging,
  and the SIOT link, and SIOT connects to `/dev/ttyACM0` at 115200 with no extra
  wiring or adapter.
- `apps/siot-net` already publishes real points with no new firmware sensors:
  `board` and `versionFW` from `lib/zbus.c`, `uptime` and `metricSysCPUPercent`
  from `lib/metrics.c`, and `bootCount`, `description`, `staticIP`, `address`,
  `netmask`, and `gateway` from the NVS-persisted set in
  `apps/siot-net/src/main.c`. That is enough to prove both directions end to end
  on day one.
- `apps/siot-net/src/web.c` already maintains a point cache (`web_points[40]`,
  fed by a zbus message-subscriber thread) and serves it as JSON over HTTP. The
  shell emitter is the same pattern pointed at the console instead of a socket,
  so there is a working reference to copy.
- The board is already in the build matrix:
  `siot_build_nucleo_h743zi apps/siot-net/`, flashed with `siot_flash`. Note
  that `siot_net_frontend_build` must run first in a fresh workspace, since the
  app embeds its web assets.

The console on this target is chatty. `apps/siot-net/prj.conf` enables
`CONFIG_LOG`, `CONFIG_NET_LOG`, `CONFIG_NET_SHELL`, and `CONFIG_I2C_SHELL`, so
network and driver log lines share the wire with point data. That is a feature
for this plan rather than a problem: it forces the reader to be noise-tolerant
from the start, which is what makes the mode work on any Zephyr console.

## Design Decisions

**A mode on the existing `serialDev` node, not a new node type.** Everything
around the protocol is unchanged: port and baud configuration, hotplug
detection, statistics, parent synchronization, tags, and the node's place in the
tree. Only framing and encoding differ. A new node type would duplicate all of
it.

**The mode point is `protocol`**, reusing the existing point type (Modbus
already uses it), with values `binary` and `shell`. Empty means `binary`, so
every existing node keeps working with no migration.

**Both directions use the same field format, differing only in the verb.**

```
pt uptime 0 INT 3600                      MCU to MPU
pt board 0 STR nucleo_h743zi
pt description 0 STR "lab bench H7"

p description 0 STR "lab bench H7"        MPU to MCU
p staticIP 0 INT 1
```

`p` is the command the firmware already registers and a developer already types.
`pt` is the same four fields with a different verb. One tokenizer on each side
serves both directions, and an emitted line becomes a replayable command by
changing one character.

An earlier draft of this plan sent JSON objects (`pt {"t":"uptime",...}`) MCU to
MPU, on the grounds that the firmware already has `point_json_encode` and that
JSON handles string data containing spaces without escaping rules of our own.
Neither argument survives inspection:

- The reuse is partly illusory. Streaming arbitrary points through
  `point_to_point_js` is what makes its broken `default:` branch reachable (see
  the MCU section), so the JSON path would need repair before it could be
  trusted. Formatting the four fields directly from the `point` struct is less
  code than the encoder plus its fix, and it needs only the `ftoa` and `itoa`
  helpers the firmware already has in `siot-string.h`.
- The escaping is needed regardless. SIOT must already quote and escape when it
  writes `p description 0 STR "lab bench H7"`, because that is what the Zephyr
  shell tokenizer expects. Having the MCU use those same rules when it emits
  costs a small helper on each side and removes a second format entirely, rather
  than adding one.

JSON would also roughly triple the bytes per point, which is not a deciding
factor at these rates but is not an argument in its favor either.

**Quoting follows the Zephyr shell's rules**, since one side of the link is
literally that tokenizer. A value containing a space, a double quote, a
backslash, or a control character is wrapped in double quotes with backslash
escaping. Everything else is emitted bare. The MCU applies this when emitting;
SIOT applies it when writing and reverses it when parsing.

**The verbs are deliberately asymmetric.** The MCU emits `pt `, SIOT writes
`p `. If shell echo is ever left on, or a developer is typing at the console at
the same time, SIOT will not mistake an echoed command for a point report. This
is the one place the two directions must not look alike.

`p` is fixed by the existing command. `pt` for the emitted line is admittedly
arbitrary — it abbreviates "point", which describes both directions equally.
Direction-based pairs (`pi`/`po`) were considered and set aside: they need an
anchor explained ("in" and "out" relative to which end) and differ by a single
character, which is easy to misread in a fast-scrolling console. Spelling the
emitted verb out as `point`, or using `pub`, would read better and remains an
easy change, since only the prefix constant moves. Keeping `pt` for now.

**SIOT stamps the points it writes; the MCU carries that stamp back.** These are
two separable questions that an earlier draft of this plan wrongly treated as
one:

- _MCU-originated_ points need a real clock the firmware does not have.
  `point.time` exists in the `point` struct but is only ever set to zero
  (`lib/point.c:248`), and the Nucleo has no RTC by default. So the MCU sends no
  timestamp on points it originates, and SIOT stamps those on arrival.
- _SIOT-originated_ points need no clock on the MCU at all. The MCU stores the
  value SIOT stamped and echoes it back verbatim, never needing to know what
  time it is. It is a carrier, not a timekeeper.

The second costs almost nothing and pays for itself immediately in loop
suppression, below. So the fifth field is populated from day one on `p`
commands, and comes back on the `pt ` echo.

**The timestamp is RFC 3339 UTC with fixed nine-digit nanoseconds.**

```
p description 0 STR "lab bench H7" 2026-07-31T12:00:00.000000000Z
```

Readability is the reason. The whole premise of shell mode is that a developer
can read the transcript, and `1785499200000000000` defeats that in a way
`2026-07-31T12:00:00.000000000Z` does not. It contains no spaces, so it needs no
quoting, and at 30 characters it is nowhere near the 256-byte command buffer.

**Use a fixed-width layout, not `time.RFC3339Nano`.** Go's `RFC3339Nano` trims
trailing zeros from the fractional part, so the same instant renders as
`...00Z`, `...00.1Z`, `...00.12Z`, or `...00.123456789Z` depending on its value.
Two consequences:

- The encoding is not canonical, which makes a byte-identical echo depend on the
  MCU reproducing Go's trimming rule exactly. That is a fragile thing to require
  of a second implementation.
- The strings do not sort lexically. `2026-07-31T12:00:00Z` is _earlier_ than
  `2026-07-31T12:00:00.5Z`, but sorts after it, because `.` is below `Z` in
  ASCII. Go's own documentation warns about this.

The layout `2006-01-02T15:04:05.000000000Z07:00` always emits nine digits, which
is canonical, sorts correctly, and is reproducible by any correct formatter on
either end. Always format as UTC so the MCU parser never has to handle a numeric
zone offset.

Separately, **compare parsed times rather than strings** when detecting an echo
(`time.Time.Equal`). The fixed layout should make the bytes match anyway, but
depending on that is depending on two formatters agreeing, and there is no
reason to.

**The reader tolerates arbitrary noise.** This console carries the Zephyr boot
banner, log lines (`[00:00:12.345,000] <inf> siot: Network connected`), the
shell prompt (`uart:~$ `), output from `net` and `i2c` shell commands, and
possibly VT100 escape sequences. Any line that is not a well-formed `pt ` line
is not an error. Log lines are recognized and forwarded as `log` points, every
line goes to the server log when debug is on, and anything left over is ignored.
This is what makes the mode robust on a link that was never designed to be
machine-only.

**Loop suppression lives on the SIOT side and keys on the timestamp.** The MCU
emitter subscribes to `point_chan`, and the `p` command handler publishes to
`point_chan`, so a point SIOT writes comes straight back as a `pt ` line. Left
alone this loops indefinitely: NATS point, `p` write, `pt ` echo, NATS point,
and around again.

What keeps that loop alive is the timestamp being regenerated on every trip. A
point with no time is stamped `time.Now()` on arrival (`store/sqlite.go:484`),
so each lap looks newer than the last and the store accepts it forever. Carrying
SIOT's original stamp through the MCU and back breaks that: the echo arrives
bearing the exact time SIOT wrote, so the client can recognize it by identity.
The rule is "I already wrote this exact type, key, value, and timestamp" — an
equality test, with no time window and no guessing.

The window is what this replaces. An earlier draft matched on value alone and
dropped an inbound point only if it arrived within two seconds of the write,
which is heuristic in both directions: a congested link delays the echo past the
window and the loop returns, while a genuine MCU-side change back to the same
value inside the window is wrongly swallowed. Timestamp identity has neither
failure mode.

**Suppression is still required; timestamps alone do not end the loop.** It is
tempting to assume the store's merge rule handles it, and it does not. The
accept condition is `pDb.Time.Before(pIn.Time) || pDb.Time.Equal(pIn.Time)`
(`store/sqlite.go:495`), so a point arriving with a timestamp equal to the
stored one is written and re-broadcast rather than ignored. Making that
comparison strict would end the loop with no client-side logic at all, but it is
core store semantics affecting every client and is presumably deliberate, so
this plan does not touch it.

Doing the suppression in the client rather than the MCU keeps the firmware
simple, and the echo stays useful as confirmation the MCU accepted the write.

**No high-rate, file transfer, or ack/retry in shell mode.** The `phr`, `file`,
and `ack` subjects have no shell equivalent, and inventing one would mean base64
over a console. Nodes that need those stay on the binary protocol. The UI hides
the corresponding controls when the mode is `shell`.

**Console visibility is the server log, not a view in the web UI.** An
interactive terminal panel in the frontend is the obvious feature to want here,
and it was in an earlier draft of this plan. It is also most of the frontend
work in the plan: a ring buffer, a NATS request subject, an HTTP endpoint, a new
Elm API module, and stateful terminal handling threaded through
`Pages/Home_.elm`, since components in this tree are pure views with no state of
their own.

Mirroring the console to stdout gets most of the value for a fraction of that.
It works headless, it lands in the journal next to everything else SIOT logs,
and it needs no new API surface. The web UI is left with configuration only. A
terminal panel remains a clean later addition, and nothing here forecloses it.

The one capability deferred with it is sending arbitrary shell commands from
SIOT. Points still flow both ways, so the loss is limited to commands like
`net iface` and `kernel threads`, which `tio` covers while SIOT is detached.

## Wire Protocol

### Framing

Lines terminated by `\n`, with an optional preceding `\r`. No COBS, no CRC. The
link is a console; the shell already defines the framing.

Reader rules, in order:

1. Strip VT100 escape sequences (`ESC [ ... <final byte>`). The connect
   handshake disables colors, but a firmware built with them on, or a session
   already in progress, should not produce garbage.
2. Strip a leading shell prompt if present (default `uart:~$ `, configurable).
3. Trim trailing whitespace.
4. Classify:
   - starts with `pt ` and the remaining fields parse: a point
   - matches the Zephyr log pattern `[HH:MM:SS.mmm,uuu] <lvl> module: text`: a
     log line, forwarded as a `log` point
   - anything else: ignored

   Classification does not affect the server log. Every line reaches stdout when
   `logConsole` is on, regardless of which bucket it falls into.

5. A line exceeding `maxMessageLength` is discarded and counted in `errorCount`,
   and the reader resynchronizes at the next `\n`.

### Point line

Identical in both directions apart from the verb:

```
pt <type> <key> <INT|FLT|STR|JSN> <data> [<time>]     MCU to MPU
p  <type> <key> <INT|FLT|STR|JSN> <data> [<time>]     MPU to MCU

p description 0 STR "lab bench H7" 2026-07-31T12:00:00.000000000Z
pt uptime 0 INT 3600
```

| Field     | Notes                                                                                          |
| --------- | ---------------------------------------------------------------------------------------------- |
| type      | required                                                                                       |
| key       | required; `0` when the point has no key                                                        |
| data type | `FLT`, `INT`, `STR`, or `JSN`                                                                  |
| data      | string form of the value; quoted when it needs to be                                           |
| time      | optional; RFC 3339 UTC, always nine fractional digits and a `Z`. Absent means stamp on arrival |

Who populates the time field:

| Direction                        | Time field                                  |
| -------------------------------- | ------------------------------------------- |
| SIOT to MCU                      | always set, from the point's own timestamp  |
| MCU echo of a SIOT-written point | the stamp SIOT sent, unchanged              |
| MCU-originated point             | absent, until the firmware has a real clock |

An absent time on an inbound point means SIOT stamps it on arrival, which is
also what the store does for a zero time (`store/sqlite.go:484`). A present time
that matches one SIOT wrote identifies an echo — see loop suppression above.

Mapping to `data.Point`:

| Field | `data.Point`                                              |
| ----- | --------------------------------------------------------- |
| `FLT` | `DataType` `PointDataTypeFloat`, parsed with `ParseFloat` |
| `INT` | `DataType` `PointDataTypeInt`, parsed with `Atoi`         |
| `STR` | `DataType` `PointDataTypeString`                          |
| `JSN` | `DataType` `PointDataTypeJSON`                            |

The firmware's key convention is `0` for keyless points (`points_merge` rewrites
an empty key to `0`). SIOT uses `""`. The client translates in both directions
so neither side sees the other's convention.

### Quoting

Fields are separated by single spaces. A field is emitted bare unless it
contains a space, a double quote, a backslash, or a control character, in which
case it is wrapped in double quotes with `\"`, `\\`, `\r`, and `\n` escaped.
This matches what the Zephyr shell tokenizer accepts, which is the constraint
that fixes the rules: SIOT has to satisfy it when writing `p` commands, so using
the same rules for `pt` costs nothing extra.

In practice only string points ever need quoting, and among the points this
target publishes only `description` is likely to.

### Limits

Field widths are bounded by the firmware's `point` struct (`include/point.h`):
type 24 bytes, key 20, data 20. SIOT truncates nothing on receive, but the
client warns at debug level 2 when a point it is about to send would be
truncated by the MCU, since a silently shortened `address` string is hard to
diagnose from the far end.

`CONFIG_SHELL_CMD_BUFF_SIZE` defaults to 256 bytes, so the client refuses to
send a command longer than that and counts it as an error rather than letting
the shell silently truncate it. A worst-case point is well under: 24 + 20 + 3 +
20 plus separators and quoting is around 75 bytes.

### Connect handshake

On open, and after any reconnect, the client writes:

```
<newline>              clear any partial line left in the shell buffer
shell echo off         stop the shell echoing our writes
shell colors off       stop VT100 color sequences
siot stream on         start point streaming
siot dump              ask for every cached point
```

`siot stream` and `siot dump` are new MCU commands. There is no time
synchronization step, since the firmware has no clock to set.

`connected` becomes true once any recognizable line arrives, not merely when the
port opens, and reverts to false after `timeout` seconds with no line received
(default 60). An open file descriptor on a USB serial port says nothing about
whether anything is alive on the other end, and on this board the port survives
an MCU reset.

## Point and Node Types

Added to `data/schema.go`, under the serial MCU client section:

```go
PointValueProtocolBinary = "binary" // COBS-framed binary packets (default)
PointValueProtocolShell  = "shell"  // Zephyr console shell, ASCII

PointTypeTimeout    = "timeout"    // seconds without traffic before disconnected
PointTypeLogConsole = "logConsole" // mirror the MCU console to the server log
```

`PointTypeProtocol` already exists (`data/schema.go:99`). `PointTypeLog`,
`PointTypeConnected`, `PointTypeRx`, `PointTypeTx`, `PointTypeErrorCount` and
the reset points are all reused unchanged, as is `PointTypeDebug`.

## Client Configuration

`SerialDev` in `client/serial.go` gains:

```go
Protocol   string `point:"protocol"`   // "" or "binary", or "shell"
Timeout    int    `point:"timeout"`    // seconds; 0 means default 60
LogConsole bool   `point:"logConsole"` // mirror MCU console to the server log
```

## Console Logging and Debug Levels

Once SIOT holds the port, nothing else can read it, so mirroring the console to
the server log is what keeps the board observable. It works headless, it lands
in the journal when SIOT runs under systemd, and it lets a developer watch a
board come up in the same terminal they started SIOT in. With it on, a separate
`tio` session is only needed before a SIOT node is attached to the port at all.

**Console logging is its own checkbox, not a debug level.** A new boolean point,
`logConsole`, turns it on. It is not folded into the `debug` level because the
two are not points on one "more verbose" axis — they answer different questions,
and a developer wants them independently. Watching a board boot means the
console with no protocol chatter; diagnosing why a point is not arriving means
the protocol detail with no network log spam. A numeric level forces one to
imply the other. A checkbox is also easier to find than remembering that 1 means
console.

When on, every line the MCU produces — boot banner, Zephyr log lines, shell
command output, and `pt ` point lines alike — goes to the server log verbatim,
after prompt and escape-sequence cleanup. Two details make it usable rather than
merely present:

- **Every line is tagged with the node description**, following the existing
  `log.Printf("Serial client %v: log: %v", sd.config.Description, ...)` pattern
  in `client/serial.go`. With several boards attached, untagged console output
  is worse than none.
- **It is loud, and the documentation should say so.** The Nucleo target builds
  with `CONFIG_NET_LOG` and several shell modules enabled, so a busy network
  stack produces a steady stream. That is the intent, but it belongs in a bench
  workflow rather than left on in a deployment.

The `debug` point keeps its existing role, covering protocol detail only:

| Level | Shell mode behavior                                                  |
| ----- | -------------------------------------------------------------------- |
| 0     | nothing                                                              |
| 2     | malformed `pt ` lines, oversize lines, and field-truncation warnings |
| 4     | adds each point decoded and each `p` command written to the MCU      |
| 9     | adds raw bytes as read from the port, before line assembly           |

Level 1 has no shell-mode meaning. In binary mode it logs ASCII strings received
from the MCU, which is what `logConsole` now does explicitly, so shell mode
leaves it equivalent to 0 rather than quietly duplicating the checkbox.

Neither control affects the `log` point path. Lines matching the Zephyr log
format are still forwarded as `log` points on NATS regardless, which is what the
existing binary mode does and what feeds the "Last log" field in the node UI.
All three can be active at once, and they answer different questions: the `log`
point is the most recent message visible in the UI, the server log is the full
console history, and the debug levels are the protocol view.

## MCU Work

In `/scratch/simpleiot/zephyr-siot/siot`. Most of it belongs in the shared
`lib/` so every application in the tree gets it, not only `siot-net`.

### `lib/point-cache.c` (new)

`apps/siot-net/src/web.c` already maintains a point cache: `web_points[40]`
guarded by `web_points_lock`, filled by a `ZBUS_MSG_SUBSCRIBER_DEFINE` thread
calling `points_merge`, and serialized with `points_json_encode` for the HTTP
GET. The shell emitter needs exactly the same thing. Building a second copy
inside the shell code would leave two subscriber threads on `point_chan`
maintaining two caches that drift apart, which is a poor trade for a file name.

So this lands as `lib/point-cache.c`: the library's canonical point cache, with
the shell as its first consumer. The web server becomes the second consumer
later, deleting its private copy. That migration is deliberately **not** part of
this plan — it touches a working HTTP path for no benefit to the link being
built here — but the module is shaped so the change is a deletion rather than a
rewrite.

The cache:

- One static `point` array sized by Kconfig, one mutex, one
  `ZBUS_MSG_SUBSCRIBER_DEFINE` thread on `point_chan` merging each arriving
  point with `points_merge`.
- The lock is internal. Callers get an API rather than a shared array, which is
  what makes the eventual web migration mechanical:

  ```c
  int point_cache_merge(const point *p);
  int point_cache_count(void);
  int point_cache_foreach(int (*cb)(const point *p, void *ud), void *ud);
  int point_cache_json_encode(char *buf, size_t len);  /* what web.c needs */
  ```

- **Size is capped at `POINT_JS_ARRAY_MAX`.** `points_json_encode` returns
  `-ENOMEM` when handed more points than that (`lib/point.c:328`), which is 47
  today. `web_points[40]` sits just under it. The Kconfig option defaults to 40
  to match, and the module should fail the build or clamp rather than let a
  larger cache silently break JSON encoding for the web path that will depend on
  it. Note also that `points_json_encode` puts a
  `char data_buf[POINT_JS_ARRAY_MAX][20]` on the stack, so the encoding thread
  needs close to a kilobyte of headroom on top of its own use.

The shell layer, in the same file for now:

- Emission from the subscriber thread, as each point is merged:
  `shell_print(shell_backend_uart_get_ptr(), "pt %s %s %s %s", type, key, dt, data)`,
  with `data` run through the quoting helper, and a trailing RFC 3339 field
  appended via `timeconv_rfc3339_from_epoch_ns_utc` when `point.time` is
  nonzero. No JSON encoder is involved.
- The data-to-string step is the same conversion `point_to_point_js` already
  does — `ftoa` at four digits for floats, `itoa` base 10 for ints, a copy for
  strings. Extract it as
  `point_data_to_string(const point *p, char *buf, size_t len)` in `lib/point.c`
  and have both the JSON encoder and this emitter call it, rather than growing a
  second copy of the switch.
- `shell_fprintf` and the print macros are documented as callable from a thread
  but not from an interrupt context
  (`zephyr/include/zephyr/shell/shell.h:1214`), which is what makes asynchronous
  emission from this thread legitimate rather than a workaround.
- `SHELL_CMD_REGISTER(siot, ...)` with subcommands:
  - `siot dump` — walk the cache with `point_cache_foreach` and emit one `pt `
    line per point. Never a single blob; a full cache would exceed any sane line
    length, which is what `point_cache_json_encode` is for over HTTP.
  - `siot stream on|off` — control streaming, so a human at the console is not
    flooded. Default off, turned on by the SIOT handshake.
  - `siot status` — cache occupancy and streaming state, for debugging.

Keeping the emitter here rather than in its own file is a judgment call worth
revisiting: emitting as part of the merge is genuinely the right place for it,
and the shell portion is small. If it grows past the cache itself, split it into
`lib/point-shell.c` and have it consume the cache API like any other client.

### `lib/point.c`

Required for this plan:

1. Extract `point_data_to_string`, as described above, and have
   `point_to_point_js` use it.
2. Handle `POINT_DATA_TYPE_JSON` in `point_js_to_point`.
   `POINT_DATA_TYPE_JSON_S` is defined in `include/point.h` but decoding `JSN`
   currently falls through to `POINT_DATA_TYPE_UNKNOWN` and returns an error, so
   the `p` command cannot accept a `JSN` point today. The shell parser is the
   same code path.
3. Extend `handle_sendpoint` to accept an optional fifth positional argument
   (`argv[5]`) holding an RFC 3339 time, and store the converted value in
   `p->time`. When the argument is absent, leave `p->time` at zero as today.
   Note that `point_js_to_point` currently forces `p->time = 0`
   (`lib/point.c:248`), so the time must be applied after that call or that line
   must stop clobbering it.
4. The emitter skips points whose data type is unknown or out of range rather
   than formatting garbage.

### `lib/timeconv.c` (new, ported)

RFC 3339 conversion is the one thing this tree lacks. It exists already in the
sibling openity tree as `lib/timeconv.c` — 91 lines, two functions:

```c
int      timeconv_rfc3339_from_epoch_ns_utc(uint64_t epoch_ns, char *buf, size_t buf_len);
uint64_t timeconv_epoch_ns_from_rfc3339(const char *buf, size_t buf_len);
```

Port it rather than writing it again, and confirm the formatter emits nine
fractional digits rather than trimming, since that is what makes the encoding
canonical. Add native-sim coverage under `tests/` for the round trip, a leap
year, and a value with trailing zeros in the fraction.

The MCU still never _interprets_ the timestamp in the sense of caring what time
it is. It converts the wire form to `point.time`, keeps it in the cache, and
converts back on the way out. No clock and no RTC are involved. The conversion
is also not wasted work: any firmware that eventually gains a real clock needs
these two functions regardless, and porting them now converges the two Zephyr
trees on one time representation.

Worth fixing while here, though the shell path no longer depends on it:

`point_to_point_js` leaves `p_js->dt` unassigned in its `default:` branch and
sets `d.start` to `NULL`, and `point_json_encode` carries the comment that all
fields must be filled in or the encoder will crash. This is reachable today
without any of this plan's changes: `points_merge` rejects an _existing_ cache
entry with an unknown data type, but does not check the _incoming_ point before
storing it in an empty slot (`lib/point.c`), so an unknown point can enter
`web_points` and crash the next HTTP GET through `points_json_encode`. An
earlier draft of this plan described this as a defect the emitter would expose;
it is really a live bug on the existing web path, and worth fixing on its own
terms. The eventual migration of `web.c` onto the shared cache makes it more
pressing, not less.

Still future work, needing a real clock rather than just a carrier: stamping
MCU-originated points. That is the half of the timestamp story this plan does
not attempt.

### `lib/Kconfig` and `lib/CMakeLists.txt`

Two options inside the existing `if LIB_SIOT` block, split along the same seam
as the file itself so the web server can later depend on the cache without
pulling in the shell:

| Option                          | Default | Purpose                                                                               |
| ------------------------------- | ------- | ------------------------------------------------------------------------------------- |
| `SIOT_POINT_CACHE`              | n       | build the cache and its subscriber thread                                             |
| `SIOT_POINT_CACHE_SIZE`         | 40      | entries; must not exceed `POINT_JS_ARRAY_MAX`                                         |
| `SIOT_POINT_CACHE_THREAD_STACK` | —       | subscriber thread stack                                                               |
| `SIOT_POINT_CACHE_THREAD_PRIO`  | —       | subscriber thread priority                                                            |
| `SIOT_POINT_SHELL`              | n       | the `siot` command and streaming; `select`s `SIOT_POINT_CACHE` and `depends on SHELL` |

Add `point-cache.c` to `zephyr_library_sources`, guarded on
`CONFIG_SIOT_POINT_CACHE`.

### `apps/siot-net/prj.conf`

```
CONFIG_SIOT_POINT_SHELL=y
CONFIG_SHELL_PRINTF_BUFF_SIZE=128
```

The shell print buffer defaults to 30 bytes, which chunks every emitted line
into four or five UART flushes. Raising it is a straightforward win.

### Documentation

`apps/siot-net/README.md` gains the new shell commands and a short section on
connecting SIOT to `/dev/ttyACM0`. `lib/README.md` documents the point cache and
its API, noting that it is intended to become the single cache in the tree and
that `web.c` should adopt it. `CHANGELOG.md` in that repo gets an entry.

## Phases

### Phase 1: Schema, Config, and Port Abstraction

**Goal:** The serial client can open a port in either mode, with no shell
behavior yet.

1. `data/schema.go` — add `PointValueProtocolBinary`, `PointValueProtocolShell`,
   `PointTypeTimeout`, and `PointTypeLogConsole`.
2. `client/serial.go` — add `Protocol`, `Timeout`, and `LogConsole` to
   `SerialDev`.
3. Introduce a small interface so the client is not tied to COBS:

   ```go
   type serialPort interface {
       io.ReadWriteCloser
       SetDebug(int)
   }
   ```

   `*CobsWrapper` already satisfies it. Replace the `portCobsWrapper` field with
   `port serialPort` throughout. This is a mechanical rename; behavior does not
   change.

4. `openPort` selects the wrapper by protocol, treating an empty `Protocol` as
   `binary`.
5. Add `protocol` to the point types that trigger a port reopen, alongside
   `port`, `baud`, `disabled`, and `maxMessageLength`.

**Verify:** `go build ./... && go test -race ./client/` — the existing
`TestSerial` and `TestSerialLargeMessage` must pass unchanged, which is the real
check that the rename was faithful.

### Phase 2: Line Wrapper

**Goal:** A tested `io.ReadWriteCloser` that returns one cleaned console line
per `Read`.

1. `client/line-wrapper.go` —
   `NewLineWrapper(port io.ReadWriteCloser, maxLen int)`, modeled on
   `client/cobs-wrapper.go`:
   - `Read` accumulates until `\n`, returns one line with escape sequences and
     prompt stripped and whitespace trimmed.
   - Overlong lines return `ErrLineTooLong` and resynchronize at the next `\n`.
   - `Write` is a pass-through; callers supply their own terminators.
   - `SetDebug` and `SetPrompt`. `SetDebug` handles level 9 raw-byte logging
     inside the wrapper, exactly as `CobsWrapper.SetDebug` does today. Console
     logging belongs in the client, where the node description is available for
     tagging.

**Test:** `client/line-wrapper_test.go`, feeding a `bytes.Buffer` or `net.Pipe`:

- `\r\n` and bare `\n` both terminate a line.
- A line split across two underlying reads assembles correctly.
- Two lines in one read return on successive `Read` calls.
- A VT100 color sequence around text is stripped, text preserved.
- A leading `uart:~$ ` prompt is stripped, including the case where a prompt
  precedes real output on the same line.
- A line longer than `maxLen` returns an error and the following line reads
  cleanly.
- A trailing partial line with no terminator does not return until terminated.

### Phase 3: Shell Protocol Encode and Decode

**Goal:** Pure functions converting between console lines and points.

1. `client/serial-shell.go`:

   ```go
   // splitFields and quoteField implement the Zephyr shell tokenizer's
   // quoting rules and are used by both directions.
   func splitFields(line string) ([]string, error)
   func quoteField(s string) string

   func parseShellLine(line string) (data.Point, lineKind, error)
   func formatPointWrite(p data.Point) (string, error)
   ```

   `lineKind` is one of point, log, or other. `parseShellLine` handles the key
   translation between `0` and `""`, the data type mapping above, and the
   optional trailing time field. `formatPointWrite` emits the same field layout
   with the `p` verb, and enforces the command length limit.

   Both directions share `splitFields` and `quoteField`, which is the point of
   using one format: there is one set of quoting rules in the codebase, tested
   once.

**Test:** `client/serial-shell_test.go`:

- Each of `FLT`, `INT`, `STR`, `JSN` round-trips through `formatPointWrite` then
  `parseShellLine`, after swapping the verb.
- A line with no time field leaves `Time` zero for the caller to stamp; one with
  a time field yields exactly that instant, to the nanosecond.
- A timestamp round-trips through `formatPointWrite` and back without drift,
  which is what echo detection depends on.
- `formatPointWrite` emits nine fractional digits for a whole-second time, a
  tenth-of-a-second time, and a full-nanosecond time. This is the
  `time.RFC3339Nano` trimming trap, and the assertion that guards against
  someone swapping the layout constant later.
- The formatted output is always UTC, including when the point's `time.Time`
  carries a non-UTC location.
- A trimmed-fraction time (`...00.5Z`) and a no-fraction time (`...00Z`) both
  still _parse_, since the MCU or a hand-typed command may produce them. Only
  the formatter is strict.
- A malformed time field is an error, not a silently zeroed time.
- Key `0` decodes to `""`, and `""` encodes to `0`.
- Quoting round-trips: a value with a space, one with an embedded double quote,
  one with a backslash, one that is empty, and one that needs no quoting at all
  and must not acquire any.
- A line with an unterminated quote returns an error rather than a partial
  point.
- Real captured lines from a Nucleo-H743ZI boot are classified correctly: the
  Zephyr banner, `[00:00:00.310,000] <inf> siot: Network connected`, a
  `uart:~$ ` prompt, and `net iface` output are all classified as log or other,
  never as errors.
- A `pt ` line with too few fields, an unknown data type, or an empty type
  returns an error.
- A string containing spaces, a double quote, and a backslash survives
  `formatPointWrite` and the shell tokenizer's rules.
- A point whose formatted command exceeds `CONFIG_SHELL_CMD_BUFF_SIZE` returns
  an error rather than a truncated command.
- A float formats without exponent notation, which the firmware's `atof` handles
  but which is unreadable at a console.
- A point whose data exceeds the firmware's 20-byte field is flagged for the
  truncation warning.

### Phase 4: Shell Mode in the Client

**Goal:** End-to-end point flow over a shell link.

1. Restructure the `serialReadData` case in `Run()` so decoding produces
   `(subject, points, err)` and the shared publish, statistics, and rate blocks
   run once for both protocols. The `ack`, `phr`, and `file` branches move under
   a binary-only guard.
2. Shell decode path: classify the line, convert points, forward log lines as
   `log` points, count unparseable `pt ` lines in `errorCount`.
3. Shell send path: `sendPointsToDevice` writes one `p` command per point,
   incrementing `tx` per point. No sequence numbers, no acks.
4. Echo suppression: a `map[pointKey]struct{data []byte, dataType, time}`
   recording each write. An inbound point whose value **and timestamp** both
   match the recorded write is an echo; drop it and clear the entry. Compare
   times with `time.Time.Equal`, not string equality. Anything else is accepted,
   including the same value at a different time, so a genuine MCU-side change is
   never masked.

   Entries also expire on a generous timer (a minute, say) so a point written
   just before a disconnect does not linger and swallow a legitimate report
   after reconnect. This is a memory bound, not the correctness mechanism —
   correctness comes from the timestamp equality.

5. Connect handshake as specified above, written on port open.
6. Connection watchdog: a timer that publishes `connected` false after `timeout`
   seconds with no line received, and true again on the next line.
7. Console logging: when `logConsole` is set, write every received line to
   stdout tagged with the node description. Log it at one place, immediately
   after a line comes back from the wrapper and before classification. Logging
   inside the classification branches would silently drop whatever falls through
   to the ignored case, which is exactly the output worth seeing when something
   unexpected is on the wire.
8. Debug levels per the table above, covering protocol detail only. These are
   independent of `logConsole`; neither implies the other.

**Test:** Extend `client/serial_test.go` with a shell-mode test using the
existing `test.NewFifoA`/`serialfifo` harness:

- A `pt ` line written into the FIFO appears as a point on NATS.
- A point sent to the node appears as a `p` command in the FIFO.
- The handshake commands appear in order on connect.
- A point echoed back with the same value and timestamp does not produce a
  second write, which is the regression test for the loop.
- The same point echoed back with a different value is accepted and republished.
- The same value echoed back with a _different_ timestamp is accepted, since
  that is a real MCU-side report and not an echo. This is the case value-only
  matching got wrong.
- An echo delayed well past any plausible time window is still suppressed. This
  is the other case the old window got wrong, and it is the reason for keying on
  the timestamp.
- A point SIOT writes carries its timestamp in the `p` command, and a point the
  MCU originates without one is stamped on arrival.
- Interleaved log lines and boot banner text do not increment `errorCount`.
- `connected` goes false after the timeout with no traffic and recovers on the
  next line.
- `SyncParent` behaves the same in shell mode as in binary mode. Nothing about
  it is protocol-specific, so this is an assumption worth pinning down rather
  than trusting.
- With `logConsole` set, every line reaches the log, including one that fails to
  classify. Capture the log output and assert on it, since the whole point of
  the option is the lines nothing else reports.
- With `logConsole` clear and debug at 4, points are logged but console lines
  are not. The two controls are independent, and that is worth pinning down.

**Verify:** `go test -race ./client/`, then against the Nucleo board once Phase
6 firmware is flashed.

### Phase 5: Frontend

**Goal:** Shell mode is configurable in the web UI. Configuration only; no
terminal view.

1. `frontend/src/Api/Point.elm` — expose `typeProtocol` if not already exposed,
   plus `typeTimeout` and `typeLogConsole`.
2. `frontend/src/Components/NodeSerialDev.elm`:
   - A Protocol radio (Binary / Shell).
   - A Timeout number input, shown only in shell mode.
   - A "Log console to server log" checkbox, shown only in shell mode, using the
     existing `checkboxInput` helper the component already uses for `syncParent`
     and `disabled`.
   - Hide HR Dest, HR counters, and file download in shell mode; none of them
     apply.

No change to `Pages/Home_.elm` or `NodeOptions.elm` is needed. This component
stays a pure view.

**Verify:** `siot_build_frontend`,
`cd frontend && npx elm-review && npx elm-test`, then a manual pass against the
board.

### Phase 6: MCU Firmware

**Goal:** `apps/siot-net` on the Nucleo-H743ZI streams points to the console and
accepts writes.

1. `lib/point.c` — the `point_data_to_string` extraction, `JSN` decoding, and
   the optional time argument on `handle_sendpoint`.
2. `lib/timeconv.c` — ported from the openity tree, with native-sim tests.
3. `lib/point-cache.c` — the cache, its subscriber thread, and the API, with a
   build-time check that `SIOT_POINT_CACHE_SIZE` does not exceed
   `POINT_JS_ARRAY_MAX`.
4. The `siot` shell command and streaming, on top of that API.
5. `lib/Kconfig` options and the `lib/CMakeLists.txt` entries.
6. `apps/siot-net/prj.conf` — enable `SIOT_POINT_SHELL` and raise the print
   buffer.
7. `apps/siot-net/README.md`, `lib/README.md`, and the repository
   `CHANGELOG.md`.

`web.c` keeps its own `web_points[40]` for now. Migrating it to the shared cache
is a follow-up, tracked in the open questions.

**Verify:**

```bash
cd /scratch/simpleiot/zephyr-siot/siot
. envsetup.sh
siot_net_frontend_build
siot_build_nucleo_h743zi apps/siot-net/
siot_flash
tio /dev/ttyACM0
```

At the console, confirm `siot stream on` produces `pt ` lines for `uptime` and
`metricSysCPUPercent` on the metrics cadence, that `siot dump` produces the full
cache including `board` and `versionFW`, and that
`p description 0 STR "lab bench H7"` round-trips back as a `pt ` line. Then
point a SIOT serial node at `/dev/ttyACM0` at 115200 in shell mode and confirm
points flow both ways.

Add a native-sim test under `tests/` for the JSON encoding of each data type,
including the unknown and JSON cases that currently crash, and run it with
`siot_test_native`.

This phase can be developed in parallel with Phases 2 through 4. A recorded
`tio` capture of a booting board is enough to build and test the SIOT side
against, and is worth committing as test data regardless.

### Phase 7: Documentation

1. `docs/ref/serial.md` — a Shell Protocol section covering framing, the point
   object, the write command, the handshake, and what is not supported. The
   existing content becomes the Binary Protocol section.
2. `docs/user/mcu.md` — how to choose a mode, and a worked Nucleo-H743ZI
   walkthrough from flashing to points appearing in the UI. Extend the existing
   Debug Levels section, which documents the binary meanings today, with the
   shell-mode table, and document the console logging checkbox alongside it:
   what it does, that it is independent of the debug level, and that it is loud
   on a board with network logging enabled. The existing Zephyr Examples section
   points at the `zephyr-siot` repository and should name `apps/siot-net` as the
   reference application.
3. `CHANGELOG.md` — an entry under `## Next`.
4. `plans/plans.md` — mark this plan complete.

**Verify:** `siot_test`.

## Files Touched

**New, this repository**

- `client/line-wrapper.go`
- `client/line-wrapper_test.go`
- `client/serial-shell.go`
- `client/serial-shell_test.go`
- `client/testdata/h743-console.txt` (captured boot and run output)

**Modified, this repository**

- `data/schema.go`
- `client/serial.go`
- `client/serial_test.go`
- `frontend/src/Api/Point.elm`
- `frontend/src/Components/NodeSerialDev.elm`
- `docs/ref/serial.md`
- `docs/user/mcu.md`
- `CHANGELOG.md`
- `plans/plans.md`

**zephyr-siot repository**

- `lib/point-cache.c` (new)
- `lib/timeconv.c` and `include/timeconv.h` (new, ported from openity)
- `lib/point.c`
- `lib/Kconfig`
- `lib/CMakeLists.txt`
- `lib/README.md`
- `apps/siot-net/prj.conf`
- `apps/siot-net/README.md`
- `tests/` (JSON encoding and time conversion coverage)
- `CHANGELOG.md`

## Open Questions

- Is the server log enough, or does the terminal panel need to come back? The
  answer depends on how the bench workflow actually feels once this is running.
  If reading the SIOT log turns out to be awkward in practice, the design in the
  removed section still holds: a ring buffer behind a `terminal.<nodeID>` NATS
  request subject, proxied by `api/nodes.go`, polled by the frontend only while
  open. Nothing in this plan makes that harder to add later.
- Should the MCU rate-limit emission? A firmware publishing at high rate would
  saturate a 115200 baud console, and `siot-net` runs a 500 ms ticker that could
  easily grow into a fast publisher. Nothing today publishes fast enough to be a
  problem, but a `siot stream <period>` throttle is a small addition if it
  becomes one.
- The Nucleo-H743ZI shares its console with the ST-LINK VCP, which is also what
  a developer has open in `tio`. Two readers on one port do not coexist. The
  `logConsole` option covers the common case by making SIOT the one reader and
  mirroring what it sees, but any product built from this should consider
  putting the SIOT link on a second UART and leaving the console free.
- Should the store's point merge treat an equal timestamp as "not newer" and
  ignore it, rather than accepting and re-broadcasting (`store/sqlite.go:495`)?
  That single change would end the echo loop with no client-side suppression at
  all, and would arguably match the "newer wins" semantics the rest of the
  system describes. It also affects every client and every replay path, so it is
  well outside this plan. Worth raising separately.
- Should `apps/siot-serial` be retired or repointed? It is a twelve-line stub
  that implies a binary-protocol implementation that does not exist. Out of
  scope here, but shell mode makes its purpose harder to explain.
