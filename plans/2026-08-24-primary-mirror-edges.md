# Plan: Primary and Mirror Edges

**Branch:** current checkout **Branched from:** `fc017969` **Issue:**
[#618](https://github.com/simpleiot/simpleiot/issues/618)

## Context

A node can live in more than one place in the tree. Mirroring is how a user
belongs to two groups, and how a sensor buried inside a device tree is made
visible in a portal group that a customer has access to. The edge carries the
relationship, so one node reached through two parents is one node with one set
of points.

That works for organizational nodes and breaks for hardware. When a Shelly IO
node is mirrored from inside a device tree into a group on the portal, the
portal's client manager finds it under that group and starts a second Shelly IO
client. Two instances now poll and command the same relay, from two machines,
with no coordination between them. The same happens for any node whose client
owns something outside the tree: a Modbus IO, a 1-Wire sensor, a GPIO line, a
GPS receiver.

The manager scans from the root and recurses into `group` nodes and the parent
types a client declares, which is why a device subtree synced to the portal does
not start clients — `device` is not a parent type it recurses into. A mirror
into a group escapes that boundary, and there is currently nothing on the edge
that says the escape happened.

The fix the issue settles on is to say it on the edge. One edge is the primary —
the node in the place it actually lives — and every other edge is a mirror,
which exists for organization and access control and does not run a client.

## Design Decisions

**Two edge points, `primary` and `mirror`, and an edge that carries neither.**
The third state matters as much as the first two. A user, a group, a rule, a
`db` node, and a `msgService` node have no primary location: their meaning comes
from where they sit, and running several instances is the intended behavior. A
`db` client records the subtree under its parent, a `msgService` client sees
notifications raised under its parent, and a user mirrored into two groups holds
a role in each. Those edges stay unmarked and nothing about them changes.

So the rule is not "mark every edge" but "mark the edges of nodes that own
something outside the tree." Unmarked is also what every edge in an existing
installation looks like, which means this change starts out behaving exactly as
today and takes effect as nodes are created and mirrored.

The two points are mutually exclusive by construction through a single accessor,
`NodeEdge.EdgeRole()`. When an edge somehow carries both, mirror wins: declining
to run a client is the safe direction to fail.

**Primary means the node owns something outside the tree.** The test for a given
node type is whether the client's behavior comes from where the node sits or
from a resource it holds. The codebase already answers this and the answer is
greppable: a scope-driven client reads `config.Parent`. `db` walks up from each
point to decide whether it falls under its parent (`client/db.go:717`), and
`msgService`, `user`, and `rule` subscribe to `up.<parent>.>`
(`client/msg-service.go:85`, `client/user.go:54`, `client/rule.go:257`). Mirror
one of those into a second group and the second instance does a different and
correct job, so it stays unmarked.

Every other client keys off a chip name, a serial port, an IP address, or a URL,
and two instances would contend for one resource. Where both apply, owning the
resource decides: the serial client uses `config.Parent` to build its `phrup`
subject (`client/serial.go:154`) and also holds a port, so it is primary.

| Primary                                                            | Owns                               |
| ------------------------------------------------------------------ | ---------------------------------- |
| `modbus`, `modbusIo`                                               | a serial or TCP bus, a register    |
| `oneWire`, `oneWireIO`                                             | a bus, a sensor                    |
| `shelly`, `shellyIo`                                               | a device on the network            |
| `gpio`                                                             | a kernel GPIO line                 |
| `gps`                                                              | a receiver                         |
| `serialDev`, `canBus`                                              | a port                             |
| `particle`                                                         | a cloud device session             |
| `networkManager`, `networkManagerDevice`, `networkManagerConn`     | this host's networking             |
| `ntp`                                                              | this host's clock                  |
| `browser`                                                          | this host's display                |
| `update`                                                           | this host's software               |
| `provisioning`                                                     | this host's provisioning directory |
| `sync`                                                             | an upstream connection             |
| `metrics`                                                          | this host's counters               |
| `signalGenerator`                                                  | a destination point stream         |
| `mqtt`, `mqttSub`                                                  | a broker subscription              |
| `mqttDevice`, `sparkplugGroup`, `sparkplugNode`, `sparkplugDevice` | a device publishing to the broker  |

Unmarked: `device`, `user`, `group`, `rule`, `condition`, `action`,
`actionInactive`, `variable`, `db`, `msgService`, `file`, and any custom type. A
type a user invents is unmarked and behaves as it does today, which is the right
default for a type the system knows nothing about.

`signalGenerator` is primary even though it holds no hardware, because the thing
it owns is a destination point stream. It reads `config.Parent` only to resolve
where its output goes (`client/signal-generator.go:162`), and
`Destination.Subject` (`client/subject.go:72`) resolves that to the generator's
own node ID by default, or to an explicit `NodeID` when one is set. Both of
those are the same destination for every edge, since a mirrored node is one node
reached twice, so two instances would write the same waveform twice into one
point stream at twice the sample rate. Only the `Parent` destination varies by
edge, and that one setting is not enough to make the type as a whole
tree-scoped. This is the case that shows why the `config.Parent` grep is a
signal rather than a decision procedure: what matters is whether the destination
it computes is shared.

`mqtt` is primary for a stronger reason than double-writing. The client
subscribes to broker traffic and writes points into its `mqttSub` nodes, which
are one set of nodes reached through both edges, so two instances would double
every point. The topic schema and Sparkplug builders then make it structural:
`ensureNodes` in both (`client/mqtt-schema.go:288`, `client/sparkplug.go:424`)
keeps its map of topic to node ID in memory on the client instance and mints a
fresh `uuid.New()` for any topic it has not seen, with no lookup against what is
already in the store. Two instances would therefore build two parallel node
trees for the same devices under the same `mqtt` node, each writing into its own
tree. The nodes those builders create are marked primary as well, since they
stand for a device publishing to the broker and mirroring one into a dashboard
group is exactly the case this plan is for. They are created through `SendNode`,
so they pick the mark up without the builders changing.

**The classification is a forced choice, not a default.** A table in
`data/schema.go` is a second place to remember when adding a client, and nothing
fails if it is forgotten: a new hardware type would silently come out unmarked
and reintroduce exactly the bug this plan closes. So the two groups are two
explicit maps rather than one map and a fallthrough, and a test parses the
`NodeType*` constants out of `data/schema.go` and fails on any type that appears
in neither. Adding a client without deciding then breaks the build at the moment
the decision is due.

**`SendNode` consults the table; `NewManager` cannot.** Every creation path runs
through `client.SendNode`: the UI's add-node call (`api/nodes.go:287`),
`siot import` (`client/apply.go:111`), and clients that discover hardware
(`client/shelly.go:177`, `client/onewire.go:268`). `SendNode` already fills in a
missing tombstone and node type edge point, so the role joins work it is already
doing.

Declaring the role at `NewManager` instead would read better, since a client
author edits that line anyway and a custom client built with the SDK would get
the behavior for free. It does not work: `siot import` runs in a bare CLI
process that holds a NATS connection and constructs no managers, so a registry
populated by manager construction would be empty there and imports would create
unmarked edges. Stamping in the store on a first-edge-wins basis is the other
option, and it covers every path including sync, but it still needs this same
table to know that `gpio` differs from `user`. It would buy only the heuristic,
at the cost of putting the policy in the storage layer.

**"Don't allow moving of primary nodes" is implemented as a parent-type rule
instead.** The issue proposes blocking moves of primary nodes outright, on the
grounds that it keeps the physical structure consistent. Applied literally that
also blocks reorganizing a `gpio` node between two groups, which the GPIO design
specifically encourages — a line node belongs next to the thing it drives, and
where that is may change.

The move that actually breaks something is the one that takes a node out from
under the parent that owns it: a `modbusIo` away from its `modbus` bus, a
`shellyIo` away from its `shelly` device, a `condition` away from its `rule`.
Those nodes are found by their parent, not by the tree root, so the move leaves
them inert. That is a property of the node type, not of the primary bit, so it
gets its own small table — `data.NodeTypeOwner` — and `MoveNode` consults it. A
`gpio` or `modbus` node has no owning type and stays freely movable.

The two rules are independent and both are worth having: the owner table stops a
node being moved somewhere it cannot work, and the primary/mirror bits stop a
second client running somewhere it should not.

**Deleting the primary deletes the mirrors.** A mirror of a deleted node is a
node that exists only as an entry in someone's group, with no client, no points
arriving, and no way to tell from the UI that the original is gone. Deleting the
primary edge deletes the remaining mirror edges in the same operation. Deleting
a mirror edge removes only that mirror.

**No automatic backfill for existing installations.** Edges created before this
change carry neither point, so they keep running clients exactly as they do now
— including the mirrors that motivated the issue. Marking them correctly
requires deciding which of several existing edges was meant to be the primary,
and for a node with more than one edge there is no reliable way to tell.
Re-creating the mirror (delete the mirror edge, mirror again from the primary)
marks both edges correctly, and that is what the changelog and the documentation
will say. The [design priorities](../CLAUDE.md) favor this over carrying a
migration for a small user base.

## Phase 1 — Edge Role in the Data Model

Files: `data/schema.go`, `data/node.go`, `data/edge.go`, `data/node_test.go`.

Add the point types next to the other edge point types in `data/schema.go`:

```go
// edge points that describe a node's relationship to a parent.
// See docs/ref/data.md for how primary and mirror edges behave.
PointTypePrimary = "primary"
PointTypeMirror  = "mirror"
```

Add the role type and accessor:

```go
// EdgeRole describes what an edge means for the node below it.
type EdgeRole int

const (
	// EdgeRoleNone is an edge for a node with no primary location --
	// a user, a group, a rule. Several such edges are meaningful and
	// each one runs a client.
	EdgeRoleNone EdgeRole = iota
	// EdgeRolePrimary is the one edge that owns the node. The client
	// runs here.
	EdgeRolePrimary
	// EdgeRoleMirror is an edge that exists for organization or access
	// control. No client runs here.
	EdgeRoleMirror
)

// EdgeRole returns the role this edge plays for the node below it.
// An edge carrying both points is treated as a mirror, because not
// running a client is the safe direction to fail.
func (n NodeEdge) EdgeRole() EdgeRole
```

Add the two classification maps and their lookups. Both groups are listed
explicitly so that a type belonging to neither is an omission the test catches
rather than a silent default:

```go
// primaryTypes own something outside the tree -- a bus, a line, a
// socket, this host's clock -- so exactly one client may act on them.
var primaryTypes = map[string]bool{ ... }

// treeScopedTypes take their meaning from where they sit, so several
// instances are meaningful and each one runs a client.
var treeScopedTypes = map[string]bool{ ... }

// NodeTypeIsPrimary reports whether a node of this type owns something
// outside the tree. An unclassified type returns false, which leaves it
// behaving as it does today.
func NodeTypeIsPrimary(typ string) bool

// NodeTypeOwner returns the parent type a node of this type must live
// under, or "" when the type may live anywhere. A modbusIo is found
// through its modbus bus, so moving it elsewhere leaves it inert.
func NodeTypeOwner(typ string) string
```

`NodeTypeOwner` entries: `modbusIo`→`modbus`, `oneWireIO`→`oneWire`,
`shellyIo`→`shelly`, `mqttSub`→`mqtt`, `condition`→`rule`, `action`→`rule`,
`actionInactive`→`rule`, `networkManagerDevice`→`networkManager`,
`networkManagerConn`→`networkManager`, `provisioningFile`→`provisioning`,
`sparkplugGroup`→`mqtt`, `sparkplugNode`→`sparkplugGroup`,
`sparkplugDevice`→`sparkplugNode`.

`mqttDevice` gets no entry, because the topic schema builder puts it under the
plain `group` nodes it creates for intermediate topic levels rather than
directly under `mqtt`. The three Sparkplug entries hold: `sparkplugState`
rebuilds its topic-to-node map by walking groups under the MQTT node, edge nodes
under each group, and devices under each edge node
(`client/sparkplug.go:165-204`), so breaking that chain would lose the map on
restart.

Tests cover `EdgeRole` for each of the three states and for an edge carrying
both points. A further test parses the `NodeType\w+ = "..."` constants out of
`data/schema.go` and asserts each value appears in exactly one of `primaryTypes`
and `treeScopedTypes`, so adding a client without classifying its node type
fails the build. The same test confirms no map entry names a type that does not
exist, and that `NodeTypeOwner` entries name real types on both sides.

## Phase 2 — Setting the Points

Files: `client/node.go`, `api/nodes.go`, `client/node_test.go`.

**On creation.** `SendNode` already fills in a missing tombstone and node type
edge point, and it is the one function every creation path reaches. Add the same
treatment for the role: when `data.NodeTypeIsPrimary` holds for the node type
and the caller supplied neither point, add `primary = 1`. A caller that supplies
its own role point — an import restoring a mirror, `MirrorNode` below — keeps
it. This covers every creation path, including the UI's add-node call, the
Shelly and 1-Wire clients discovering hardware, and `siot import`.

`SendNode` is also the update path, though: `ApplySend` in `client/apply.go`
carries a `Created` flag precisely because an import sends existing nodes
through the same call. Marking unconditionally would stamp `primary` on a node
that had been mirrored into a group, which is the failure this plan exists to
prevent. So the mark is applied only when the edge does not already exist, which
costs one `GetNodes` call and only for a primary-type node with no role
supplied. Leaving existing edges alone is also what makes the upgrade quiet: an
edge from before this change stays unmarked rather than being guessed at.

**On mirror.** `MirrorNode` reads the source edge's role. If the source is
primary or a mirror, the new edge gets `mirror = 1`; otherwise it gets neither
point, so mirroring a user into a second group is unchanged. The source edge is
the one the UI copied from, so `MirrorNode` gains the source parent ID:

```go
func MirrorNode(nc *nats.Conn, id, oldParent, newParent, origin string) error
```

`api/nodes.go` passes `nodeCopy.OldParent`, and the Elm `NodeCopy` encoder gains
the field. The frontend already tracks the source parent in `CopyMove`.

**On move.** `MoveNode` returns an error when `data.NodeTypeOwner(node.Type)` is
non-empty and the new parent is not of that type, naming both types in the
message so the UI can show it.

`MoveNode` also has to carry the role across. A move writes a fresh edge under
the new parent with a tombstone and a node type point and tombstones the old
one, so nothing rides along on its own: a moved node would come out unmarked and
a moved mirror would start running a client. The role of the edge under
`oldParent` is read and rewritten onto the new edge.

**On duplicate.** `duplicateNodeHelper` gives each copy a new ID, so a duplicate
of a primary node is a new primary. It writes the node's edge points through
`SendNode`, which means dropping any `mirror` point from the source edge and
letting `SendNode` add `primary` — otherwise duplicating a mirror produces a
node with no primary anywhere.

**On delete.** `DeleteNode` fetches all live edges for the node first. When the
edge being deleted is the primary, it tombstones the remaining mirror edges as
well, then the primary. Deleting a mirror or an unmarked edge is unchanged.

Tests in `client/node_test.go` against `server.TestServer`: creating a `gpio`
node marks its edge primary; creating a `group` marks nothing; mirroring the
gpio node marks the new edge mirror and leaves the primary alone; mirroring a
user marks neither; moving a `modbusIo` to a group fails and moving a `gpio` to
a group succeeds; duplicating a mirror produces a primary; deleting the primary
removes the mirror.

## Phase 3 — Clients Do Not Run on Mirrors

Files: `client/manager.go`, `client/manager_test.go`.

In `Manager.scan`, skip any node whose `EdgeRole()` is `EdgeRoleMirror` — it is
not added to `found`, so an edge that becomes a mirror while running also stops
its client through the existing removal path.

`scanHelper` recurses into `group` and the declared parent types. A mirrored
`group` should not have its children's clients started twice either, so the
recursion skips mirrored parents as well.

The manager already re-scans when a `nodeType` edge point arrives on
`up.root.>`. Add `primary` and `mirror` to that trigger so flipping a role takes
effect immediately rather than on the next minute tick.

Verify `GetNodes` returns `EdgePoints` on the nodes the manager scans; if it
does not, the role has to be read through a separate edge fetch and that is
worth knowing before the rest of the phase is written.

A test in `client/manager_test.go` builds a node under a group, mirrors it into
a second group, and asserts exactly one client instance starts; then deletes the
primary and asserts no client remains.

## Phase 4 — UI

Files: `frontend/src/Api/Node.elm`, `frontend/src/Api/Point.elm`,
`frontend/src/Pages/Home_.elm`.

Add `Point.typePrimary` and `Point.typeMirror`, and a `Node.edgeRole` helper
returning the same three states.

**Badge.** A mirrored node shows a small `mirror` label next to its description,
so a node that looks inert in a group is explained on sight. A primary node
shows nothing — it is the ordinary case and a badge on almost every node is
noise.

**Paste options.** `viewPasteNode` already knows the source and destination.
When the node's type has an owning parent type and the destination is not of
that type, hide `move` and `duplicate` and offer only `mirror`, with a line
saying where the type belongs. This turns the Phase 2 backend error into
something the user never hits.

**Delete confirmation.** When the edge being deleted is a primary with mirrors,
`viewDeleteNode` says how many mirrors go with it.

Run `npx elm-review` and `npx elm-test` in `frontend/`.

## Phase 5 — Documentation

Files: `docs/ref/data.md`, `docs/user/ui.md`, `docs/ref/client.md`,
`docs/diagrams.drawio`, `CHANGELOG.md`, `CLAUDE.md`.

`docs/ref/data.md` gains a **Primary and mirror edges** section under _Node
Topology changes_, covering the three edge roles, which node types are primary
and why the rest are not, what a mirror does and does not do, and the deletion
behavior. The `Copy`, `Move`, and `Delete` subsections above it are updated to
match.

A diagram in `docs/diagrams.drawio` exported to `docs/ref/images/`: a device
subtree holding a Shelly device with its IO as the primary edge, a portal group
holding a mirror edge to the same IO node, one client icon on the primary side
and none on the mirror side. Rounded rectangles, the standard palette, 900x500.

`docs/user/ui.md` extends the mirror bullet under _Deleting, Moving, Mirroring,
and Duplicating nodes_ to say that mirroring a sensor or bus creates a view of
it rather than a second copy that runs, that some node types can only be
mirrored and not moved, and that deleting the original removes its mirrors.

`docs/ref/client.md` tells client authors that a client runs once per primary or
unmarked edge and never on a mirror, and how to decide which side a new node
type belongs on.

`CLAUDE.md` gains a line under _Client Architecture_ pointing at the reference
section, since "which edges start a client" is now a rule a contributor needs.

`CHANGELOG.md` under `## [Unreleased]`:

```markdown
- **Mirrored hardware nodes no longer run a second client.** Mirroring a Modbus
  IO, Shelly IO, GPIO line, MQTT connection, or other hardware node into a group
  now creates a view of it, so only the instance where the node actually lives
  talks to the device. For MQTT this also stops a mirrored connection from
  building a second copy of the node tree its topic schema creates. Nodes
  mirrored before this release keep the old behavior until the mirror is
  re-created. See the
  [data reference](docs/ref/data.md#primary-and-mirror-edges).
- **Deleting a node removes its mirrors.** Mirrors of a deleted sensor or bus no
  longer linger in the groups they were mirrored into.
- **Nodes that belong to a parent can only be mirrored, not moved.** A Modbus
  IO, Shelly IO, rule condition, and similar nodes are found through their
  parent, so moving one elsewhere left it inert. The UI now offers `mirror` for
  these instead.
```

## Out of Scope

**One table of valid parents.** `NodeTypeOwner`, the `parentTypes` argument to
`NewManager`, and the add-node type list in `Home_.elm` are three descriptions
of the same relationship, maintained separately. Unifying them is worth doing
and is a larger change than this issue.

**Promoting a mirror to primary.** Useful when hardware moves between instances,
and not needed to close this issue. Delete and re-create in the meantime.

**Mirroring a single point rather than a node.** The variable-node idea raised
early in the issue thread — exposing one point to a dashboard group without
exposing the whole node — remains a good feature and is independent of this
work.
