---
name: siot-debug
description:
  Use when inspecting or troubleshooting a running Simple IoT instance —
  checking what a node's current values are, whether points are flowing, whether
  sync is replicating, or what is in the JetStream store. Also covers questions
  about an instance's identity and structure — node IDs, a node that appears in
  two places, a node that was deleted and is still there, which instance wrote a
  value. Triggers on requests like "check the X node", "the value is not
  changing", "is sync working", "what is in the store", "why is this node
  empty", "why is this node duplicated", or any question about the live state of
  a local or remote SIOT process.
---

# Debugging a running SIOT instance

Five ways to look at a live instance, roughly in the order to reach for them:

| Tool          | Answers                                                                                                |
| ------------- | ------------------------------------------------------------------------------------------------------ |
| `siot export` | What is the current configuration and the latest value of every point                                  |
| `siot dump`   | What the instance actually is: node and edge IDs, extra parents, deleted nodes, point origins, streams |
| `siot log`    | Which points are flowing right now, decoded, with node names and types                                 |
| `nats stream` | What is persisted, how much of it, and how far consumers have read                                     |
| HTTP API      | The same node data as JSON, including point timestamps and origins                                     |

Start with `export` when the question is about configuration or values, and
`dump` when it is about identity, structure, or replication. Both render the
whole tree in one command and need no authentication.

## Finding the instance

A machine often runs several instances at once (a device and its upstream, or a
released binary alongside a development build). Identify them before connecting
to anything.

```bash
ps aux | grep -i siot | grep -v grep
ls -l /proc/<pid>/cwd                                  # which data directory
tr '\0' '\n' < /proc/<pid>/environ | grep SIOT_        # port overrides
ss -lntp | grep <pid>                                  # ports actually bound
```

Default ports, each overridable by the environment variable in parentheses:

- `4222` NATS (`SIOT_NATS_PORT`)
- `8118` HTTP / web UI (`SIOT_HTTP_PORT`)
- `8222` NATS monitoring (`SIOT_NATS_HTTP_PORT`)
- `9222` NATS websocket (`SIOT_NATS_WS_PORT`)

The data directory is the process working directory, so `/proc/<pid>/cwd` tells
you which `jetstream/` tree belongs to which instance. Confirm the port from
`ss` rather than assuming the default — a second instance will have been started
with overrides.

## Reading state with `siot export`

```bash
./siot export -natsServer nats://127.0.0.1:4222
```

Output is the node tree in configuration YAML: node type as the key, each point
type as a key under it, children nested. Add `-nodeID <id>` for a subtree and
`-token` (or `SIOT_AUTH_TOKEN`) if the instance requires authentication.

To see whether something is moving, run it twice and compare the one field you
care about:

```bash
for i in 1 2 3; do
  ./siot export -natsServer nats://127.0.0.1:4222 2>/dev/null | grep -E '^      value:'
  sleep 3
done
```

Three samples beat two: they show the shape of the signal, not just that a
number differs. For a 0.1 Hz sine with min 0 and max 10 sampled every 3 s,
expect `5 + 5·sin(θ)` at 108° steps — 9.76, 5.00, 0.24 is a correct sine, and
recognizing that saves confirming the generator any other way.

## Describing an instance with `siot dump`

```bash
./siot dump -natsServer nats://127.0.0.1:4222
```

Export hides the identifiers so its output can be applied to another instance.
Dump reports exactly those identifiers, because they are what explains behavior:
the root node ID the instance replicates under, every node ID and type, deleted
nodes, and every parent of each node.

```
instance
  root 5bf3ea6b-d635-440e-8bf2-55cc60ac716a  device  "downstream"

streams
  boundary 5bf3ea6b-...  origin 48aa6237-...  3 msgs  (replica)
  boundary 5bf3ea6b-...  origin 5bf3ea6b-...  28649 msgs  (own)

tree from 5bf3ea6b-d635-440e-8bf2-55cc60ac716a
  variable  c149d538-4f38-4037-8f0f-4a0200ab39ce  "Var1"
  sync  f993b4e6-41a7-4449-8787-477070c14a48  "Sync to upstream"
  signalGenerator  6e04a088-d4f6-4039-a2be-5a10467bcdf2  "Sine wave"
```

Two flags add detail, and `-all` turns on both:

- `-points` prints every node point and edge point with the origin that wrote it
  and its timestamp. This is what to compare when two instances disagree about a
  value: the origin names which instance wrote it, and the timestamp says which
  side is behind.
- `-streams` lists the boundary-origin streams with message counts and labels
  each `own`, `replica`, or `written by this instance`, which shows in one line
  who this instance replicates with.

`-nodeID <id>` limits the tree to one subtree.

Three things dump reports that no other command does:

- **A node under more than one parent** is annotated `[also under <id>]`. A node
  that should live in one place and appears in two explains duplicated points
  and surprising edge behavior.
- **Deleted nodes** are shown with `[deleted]`. A node that should be gone and
  is not explains as much as a missing one.
- **An `anomalies` section** lists any node other than the root carrying the
  virtual `root` parent. That means the instance serves a second root, which
  usually follows a bad sync or a hand-edited store.

The same environment precedence applies as everywhere else — `SIOT_NATS_SERVER`
wins whenever `-natsServer` is left at `nats://127.0.0.1:4222`. Read the connect
banner in the output before trusting a dump.

## Watching points flow with `siot log`

```bash
./siot log -natsServer nats://<ip address>:4222
```

This subscribes to `p.>` and prints every node point decoded, so you see the
values themselves rather than binary payloads:

```
2026/08/10 09:14:22 NODE: Signal Generator (signalGenerator) (a1b2c3d4-...)
   - POINT: T:value V:9.755 O:a1b2c3d4 2026-08-10T09:14:22-04:00
```

Each message starts with the node description, node type, and node ID, followed
by one line per point: `T:` type, `V:` value or text, `K:` key when set, `O:`
origin, `Tomb` for a tombstoned point, and the point timestamp.

That combination answers most "is anything happening" questions in one command:
it shows which point types are live, how fast they update, what the values are,
and which instance originated them. Use `-token` (or `SIOT_AUTH_TOKEN`) when the
instance requires authentication.

The command resolves each node's description over NATS as messages arrive, so it
needs the same access `export` does and runs until interrupted.

The address is whatever the instance actually binds. Use `127.0.0.1` for a local
process and the device address for a remote one:

```bash
./siot log -natsServer nats://192.168.1.50:4222
```

**`SIOT_NATS_SERVER` takes precedence whenever `-natsServer` holds the default
`nats://127.0.0.1:4222`.** The commands consult the environment only when the
flag is left at its default, and passing that value explicitly is
indistinguishable from omitting it. A shell prepared by `envsetup.sh` for a
second instance exports `SIOT_NATS_SERVER`, so
`-natsServer nats://127.0.0.1:4222` connects to the other instance instead. The
connect banner names the server actually used — read it before trusting the
output. `unset SIOT_NATS_SERVER` to take control. The same applies to `export`,
`dump`, `import`, and `store`.

Filter with `grep` when a busy instance produces more than you want to read:

```bash
./siot log -natsServer nats://127.0.0.1:4222 | grep -A2 'Modbus'
```

### Subscribing directly

`siot log` renders node points. For edge points and high-rate points, and for
measuring raw message rates, subscribe with `nats`:

```bash
nats -s nats://127.0.0.1:4222 sub 'ep.>'         # edge points
nats -s nats://127.0.0.1:4222 sub 'phrup.>'      # high-rate points
nats -s nats://127.0.0.1:4222 sub 'p.<nodeID>.>' # one node, raw
```

Subject forms:

- `p.<nodeID>.<pointType>.<key>` — node points
- `ep.<nodeID>.<parentID>` — edge points
- `phr.<nodeID>` / `phrup.<parentID>.<nodeID>` — high-rate points

Payloads are the compact binary point encoding, so the body prints as noise and
only the subject and message rate are readable. Prefer `siot log` whenever you
want the values.

**A frozen value usually means nothing writes that point type any more.** Points
persist as the node's last known state forever, so a point left over from an
earlier configuration keeps its final value indefinitely and looks identical to
a stalled client. Before concluding data is stuck, run `siot log` and see which
point type is actually being published — a signal generator whose
`destination.pointType` was changed from `volt` to the default leaves a stale
`volt` point next to a live `value` point, and both appear in `export`.

## Inspecting the store

Streams are named `inst_<boundaryID>_<originID>` (see ADR-7). Each sync/AuthZ
boundary gets a stream per origin instance, so a device that has adopted an
upstream has both its own stream and a replica. `siot dump -streams` lists the
same inventory with the roles already worked out, which is quicker when you only
want to know who replicates with whom.

```bash
nats -s nats://127.0.0.1:4222 stream ls
nats -s nats://127.0.0.1:4222 stream report          # messages, consumers, size
nats -s nats://127.0.0.1:4222 stream subjects <name> # per-subject counts
nats -s nats://127.0.0.1:4222 stream info <name> -j  # retention, state
```

Stream subjects mirror the wire subjects with routing tokens prepended:
`inst.<boundary>.<origin>.<nodeID>.p.<type>.<key>` and
`inst.<boundary>.<origin>.<parentID>.ep.<childID>`.

`stream subjects` sorted by count is the fastest way to see which signals
dominate and whether a given point is being persisted at all. Per-subject
retention defaults to 5000 messages, so a busy signal sitting at exactly 5000 is
at the cap and working as intended, not truncated by an error.

To check replication, compare the same stream on both instances, and read
consumer positions with `nats consumer report <stream>`.

## HTTP API

Useful when you want point timestamps and origins, or want to script against
JSON. Three details make the difference between a working request and an opaque
error:

```bash
# 1. Log in. The "email" is whatever the user node's email point holds,
#    which is often a bare name such as "admin" rather than an address.
TOKEN=$(curl -s -X POST -d "email=admin&password=admin" \
  http://localhost:8118/v1/auth | jq -r .token)

# 2. The Authorization header needs the Bearer prefix.
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8118/v1/nodes

# 3. A single-node GET takes the parent ID in the request body; "all" works.
curl -s -X GET -H "Authorization: Bearer $TOKEN" --data "all" \
  "http://localhost:8118/v1/nodes/<nodeID>"
```

Without the `Bearer` prefix the response is `Unauthorized`; with no header at
all it is `invalid user`; with no parent in the body it is
`parent must be set to valid ID, or all`.

Each point carries the raw `dataType` and base64 `data` plus a decoded `value`
or `text`, so read `value`/`text` and ignore the encoded pair.

## Working across a sync pair

When a device syncs to an upstream, the same node exists on both sides and the
values move independently between replications. Export both and compare:

```bash
./siot export -natsServer nats://127.0.0.1:4222   # device
./siot export -natsServer nats://127.0.0.1:4333   # upstream
```

If both show live, changing values, replication is working and any problem is
downstream of the data. Confirm this before investigating the store or a client,
because it rules out most of the system in one step.

When they disagree, dump both and compare:

```bash
./siot dump -all -natsServer nats://127.0.0.1:4222 > /tmp/device.txt
./siot dump -all -natsServer nats://127.0.0.1:4333 > /tmp/upstream.txt
diff /tmp/device.txt /tmp/upstream.txt
```

This separates the two failure modes in one step. Instances that disagree about
their root IDs, or that are missing a stream for each other, have a replication
problem. Instances that agree on structure but differ on a point's origin or
timestamp have a configuration problem — something on one side is writing a
value the other side never asked for.

## Deciding where the problem is

For a report that a displayed value is not changing:

1. `export` twice. If the value moves, the data layer is healthy and the problem
   is in the frontend or in which point the display reads.
2. If it does not move, run `siot log -natsServer nats://<ip address>:4222` and
   watch what the node publishes. If points are publishing under a different
   type than the one you are reading, the configuration and the reader disagree
   — this is the common case.
3. If nothing publishes, the producing client is stopped or misconfigured. Check
   its config in the `export` output and the instance log.
4. If points publish but do not persist, compare `stream subjects` counts
   against the wire.

For a report that a node is missing, duplicated, or shows values from somewhere
else, start with `siot dump` instead. A node under two parents, a node that was
deleted and came back, and a second root all look like ordinary configuration in
`export` output, and all of them are labeled in a dump.

Elm components read a fixed point type, so a component that hardcodes one type
while its client writes a configurable destination type will display a constant
default. `Point.getValue` returns `0` when it finds nothing, which renders as a
plausible reading rather than as an obvious absence.
