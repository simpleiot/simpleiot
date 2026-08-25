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

**Primary means the node owns a physical thing.** A bus, a line, a socket, a
receiver, this host's clock or display or software image. The list:

| Primary                                                        | Owns                               |
| -------------------------------------------------------------- | ---------------------------------- |
| `modbus`, `modbusIo`                                           | a serial or TCP bus, a register    |
| `oneWire`, `oneWireIO`                                         | a bus, a sensor                    |
| `shelly`, `shellyIo`                                           | a device on the network            |
| `gpio`                                                         | a kernel GPIO line                 |
| `gps`                                                          | a receiver                         |
| `serialDev`, `canBus`                                          | a port                             |
| `particle`                                                     | a cloud device session             |
| `networkManager`, `networkManagerDevice`, `networkManagerConn` | this host's networking             |
| `ntp`                                                          | this host's clock                  |
| `browser`                                                      | this host's display                |
| `update`                                                       | this host's software               |
| `provisioning`                                                 | this host's provisioning directory |
| `sync`                                                         | an upstream connection             |
| `metrics`                                                      | this host's counters               |

Everything else — `device`, `user`, `group`, `rule`, `condition`, `action`,
`actionInactive`, `variable`, `db`, `msgService`, `file`, `signalGenerator`,
`mqtt`, `mqttSub`, `sparkplug*`, and any custom type — is unmarked. A custom
type a user invents is unmarked and therefore behaves as it does today, which is
the right default for a type the system knows nothing about.

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

Add the type tables and their lookups:

```go
// NodeTypeIsPrimary reports whether a node of this type owns something
// outside the tree -- a bus, a line, a socket, this host's clock --
// so that exactly one client may act on it.
func NodeTypeIsPrimary(typ string) bool

// NodeTypeOwner returns the parent type a node of this type must live
// under, or "" when the type may live anywhere. A modbusIo is found
// through its modbus bus, so moving it elsewhere leaves it inert.
func NodeTypeOwner(typ string) string
```

`NodeTypeOwner` entries: `modbusIo`→`modbus`, `oneWireIO`→`oneWire`,
`shellyIo`→`shelly`, `mqttSub`→`mqtt`, `condition`→`rule`, `action`→`rule`,
`actionInactive`→`rule`, `networkManagerDevice`→`networkManager`,
`networkManagerConn`→`networkManager`, `provisioningFile`→`provisioning`.

Tests cover `EdgeRole` for each of the three states and for an edge carrying
both points, and confirm the two tables agree with the node type constants (no
entry names a type that does not exist).

## Phase 2 — Setting the Points

Files: `client/node.go`, `api/nodes.go`, `client/node_test.go`.

**On creation.** `SendNode` already fills in a missing tombstone and node type
edge point. Add the same treatment for the role: when the node type is primary
and the caller supplied neither point, add `primary = 1`. A caller that supplies
its own role point — an import restoring a mirror, `MirrorNode` below — keeps
it. This covers every creation path, including the UI's add-node call, the
Shelly and 1-Wire clients discovering hardware, and `siot import`.

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
message so the UI can show it. The role points ride along with the edge and need
no change: a move rewrites the same role onto the new edge because it copies the
node type and tombstone already.

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
  IO, Shelly IO, GPIO line, or other hardware node into a group now creates a
  view of it, so only the instance where the node actually lives talks to the
  device. Nodes mirrored before this release keep the old behavior until the
  mirror is re-created. See the
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
