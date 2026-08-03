# Plan: Configuration Provisioning from YAML Files

**Branch:** `feat/provisioning` **Branched from:** `54169492`

## Context

SIOT can already export and import a tree of nodes as YAML (`client.ExportNodes`
/ `client.ImportNodes`, exposed as `siot export` and `siot import`). That is a
good manual workflow, but it is a poor fit for deployments that are built from
an image or a configuration management system:

- Import is a one-shot push over NATS. The instance has to be running and
  someone (or some script) has to pipe the file in at the right moment.
- Import is not idempotent. By default every import assigns fresh IDs, so
  running the same file twice creates a second copy of everything. Running it
  with `-preserveIDs` avoids the duplicates, but only for files that carry IDs
  in the first place.
- Import at `root` replaces the root node and tombstones the old one, which
  takes the whole tree with it. That is a heavy operation to reach for when the
  intent is usually "make sure these nodes exist".

Grafana solves the same problem with provisioning: a directory of YAML files is
read at start-up, the objects described there are created or updated, a checksum
of each file is recorded, and the files are re-read when they change. The
configuration lives in version control and on disk, and the running system
converges on it.

This plan brings that model to SIOT, and applies it to import as well. There is
one file format and one way of applying it: nodes are matched by description
rather than by ID, so applying a file creates what is missing, updates what has
drifted, and does nothing when the tree already agrees. A file can also name
nodes to delete. `siot import` applies one file once; provisioning applies a
directory of them at start-up and whenever they change. The same file works as
an initial configuration on a fresh unit and as a patch against a unit that has
been in the field for a year, whichever way it is applied.

The result is that an image can ship `/var/lib/simpleiot/provisioning/*.yaml`,
and every unit that boots that image comes up with the same configuration, with
no import step and no operator involvement -- and that `siot import` stops being
a one-shot operation people are careful with.

### What this builds on

The branch carries recent work that makes an import at `root` safe: the client
manager re-resolves the root node on every scan, and the server exits when the
root node is replaced so a service manager can restart it (`server.watchRoot`,
`client.Manager.updateRoot`). This plan removes the need for the second of
those, because nothing in it replaces a node -- see "Nothing replaces the root
node any more" below. `updateRoot` stays useful regardless.

## Design Decisions

**Provisioning is a server concern, not a client node.** Everything else in SIOT
is configured by nodes in the tree, but provisioning has to work before there is
any configuration in the tree. Like `SIOT_DATA` and the NATS ports, it is set by
a command line flag and an environment variable and handled in `server.Run`.
This also matches Grafana, where provisioning is configured in `grafana.ini`
rather than in the database it populates.

**One node format, shared by provisioning, `siot export`, and `siot import`.**
The format that `siot export` emits today names a `type:` on every node and
spells every point as a list entry naming the field that carries its value. That
is the right shape for a round trip through the API, and the wrong shape for a
file someone writes once and maintains in version control for years. This plan
replaces it with a compact form: the node type is the key, the remaining keys
are point types, and the value's YAML kind determines where the value lands.
Provisioning reads that form, `siot import` reads it, and `siot export` writes
it, so there is one format to learn, one parser to maintain, and one set of
examples in the documentation. A header carries what provisioning needs and is
optional:

```yaml
# provisioning/20-sensors.yaml
apiVersion: 1 # optional, reserved for future format changes
nodes:
  - group:
      description: Sensors
      children:
        - modbus:
            description: Modbus sensors
            port: /dev/ttyS1
            baud: 9600
            debug: 0
```

Values map to points as follows, using the type resolution the YAML library
already performs when decoding into `any`:

| YAML value               | Point                                           |
| ------------------------ | ----------------------------------------------- |
| string (`hello`, `"10"`) | `Text`                                          |
| number (`10`, `1.5`)     | `Value`                                         |
| bool (`true`)            | `Value` 1 or 0                                  |
| mapping                  | one point per entry, map key becomes `Key`      |
| sequence                 | one point per element, `Key` is `"0"`, `"1"`, … |

Quoting is the escape hatch, and it is the reason this is safe: `port: 502` is a
numeric point, `port: "502"` is a text one, so a value that looks like a number
but belongs in `Text` is one pair of quotes away. `goccy/go-yaml` preserves that
distinction, verified in this repository: decoded into `yaml.MapSlice`, `9600`
comes back as `uint64` while `"9600"`, `'9600'`, and `!!str 9600` all come back
as `string`. The codec has to decode into `MapSlice` and `any` rather than into
typed struct fields for that to hold -- a `string` field would accept both
spellings and lose the distinction -- which is a constraint on the
implementation worth stating up front.

Point keys need no syntax of their own. A point value is always a scalar, so a
mapping under a point type can only mean a set of keyed points, and a sequence
can only mean an array:

```yaml
- metrics:
    metricSysCPUFreq:
      cpu0: 1400
      cpu1: 1600
    tag: [alpha, beta] # keys "0" and "1"
```

A null value writes a point with neither `Value` nor `Text` set, which is what
`- type: phone` means in an export today.

Five keys inside a node body are reserved: `id`, `parent`, `children`, `points`,
and `edgePoints`.

**The long form remains available per point, which is what keeps export
lossless.** A point that carries only a type, an optional key, and one of
`Value` or `Text` has a short spelling. A point that carries `Data` bytes, a
tombstone, an origin, both a value and text, or a type that collides with a
reserved key does not. Rather than force those into the short form, the node
body's `points:` and `edgePoints:` keys accept a list of full point structures,
and the two spellings mix freely in one node:

```yaml
- user:
    firstName: admin
    email: admin@example.com
    edgePoints:
      - type: role
        text: admin
```

`siot export` writes the short spelling whenever it applies and falls back to
the long one per point, so an export is still readable and still carries
everything the tree holds. Edge points are usually just a role or a tombstone,
so most exports never grow an `edgePoints:` key at all.

A side effect worth noting: the short form makes a point's identity a mapping
key, so a file cannot express two points with the same type and key. That is
already an invariant of the data model, and the list form allowed violating it
silently.

**Export quotes what it must, and hand-written files carry one risk worth
stating.** On export the choice is mechanical: a point with `Text` set is
written quoted, a point with `Value` set is written bare, so a round trip is
exact. In a hand-written file it is a judgment call, and `baud: 9600` where the
client expects `Text` produces a point the client reads as empty, with no error.
The old format made that distinction explicit in the field name. Nothing in Go
holds a point-type-to-data-type map to validate against, so the mitigation for
now is documentation and the quoting rule. A point type schema that
`siot provision -check` and `siot import` could validate against would close it
properly, and is listed under later refinements.

**The old format is not read at all.** Keeping a second parser alive would mean
two code paths, two sets of tests, and documentation that has to explain which
file is which, for a format nobody should be writing after this lands. A file in
the old shape gets an error naming what it saw and what the new spelling looks
like. The migration path for anything still in a live instance is to upgrade and
re-export: the tree in the store is unchanged by the format change, so
`siot export` produces the new form from the same data. Files whose instance is
gone have to be converted by hand, which is a real cost and a small one at this
stage of the project.

**Import and provisioning are the same operation.** A provisioning pass and
`siot import` both mean "make the tree agree with this file", so they run the
same engine over the same format and differ only in what triggers them: import
applies one file when a person runs it, provisioning applies a directory of
files at start-up and whenever they change. Import becomes idempotent as a
consequence -- running it twice does what running it once did -- and gains
`delete:`, and provisioning gains nothing it has to maintain separately.

The engine therefore lives in `client`, next to `ImportNodes` and reachable from
`server`, since `server` already depends on `client` and the reverse would not
work. Provisioning is the layer above it: the directory scan, the file
checksums, and the state nodes.

Three things in `ImportNodes` have to go for this to hold:

- It appends `(import)` to the description of every top-level node. With
  descriptions carrying identity, a second import would look for
  `Sensors (import)`, not find it, and create `Sensors (import) (import)`.
  Idempotence and that suffix cannot both exist.
- `-preserveIDs`, and the `ReplaceIDs` and `checkIDs` helpers behind it, are no
  longer how identity works. Matching handles duplicates, so the flag has
  nothing left to protect against. What it was really for -- restoring a backup
  with its IDs intact -- becomes automatic: when an entry's `id:` is a UUID and
  no node matches, the node is created with that ID, and when it is a label like
  `sensors-group`, or the UUID is already taken elsewhere in the tree, a fresh
  one is generated. An exported file restores with its IDs; a hand-written file
  never pins one by accident.
- `-parentID` says something the file can say for itself, and says it in IDs,
  which is what this design is getting away from. A file that should land
  somewhere other than the root device node carries a `parent:` on the entries
  that go there.

What is left on `siot import` is the connection: `-natsServer` and `-token`. A
`-dryRun` that prints the plan without sending anything is worth adding, since
it changes nothing about what the file means. The result is that
`siot import < sensors.yaml` and dropping `sensors.yaml` in the provisioning
directory are the same operation, which is the only way this equivalence stays
true as both grow.

**A node is identified by its description, not by its ID.** Node IDs are UUIDs
that a fresh instance generates for itself, and a provisioning file that ships
in an image cannot know them. It should not have to: what the file means is "a
Modbus node called Modbus sensors, under the group called Sensors", and that is
exactly what a person reading the file in a pull request understands too. A node
in a file therefore matches an existing node when the parent and the
`description` point agree:

- **No match:** create the node with a fresh UUID and send every point the file
  names.
- **One match of the same type:** send only the points whose value differs from
  what the node already has. When nothing differs, nothing is sent.
- **One match of a different type:** report an error for that node and skip it
  along with its children. A file that says `modbus` where the tree holds a
  `group` of the same description is either a mistake or a rename that needs a
  `delete:` entry, and both are worth stopping on. Creating a second node beside
  the first would hide the problem.
- **More than one match:** report an ambiguity error for that node, record it on
  the file's state node, and skip the node and its children. Guessing here would
  silently configure the wrong device.

Type is therefore a check rather than part of the key, which keeps the rule
short to state: one description, one node, under a given parent.

A node entry with no `description` matches the single node of that type under
the parent, which is what you want for the singletons -- one `metrics` node, one
`serial` node under a device -- and is an ambiguity error if the tree already
holds more than one.

**Not every node has a description point, so the match key is the first
identifying point a node carries.** A user node has no `description`: it carries
`firstName`, `lastName`, and `email`, which is why `data.Points.Desc()` already
falls back to the name when no description is set. Matching uses the same idea,
as an ordered rule that needs no per-node-type table:

1. `description`
2. `email`
3. `firstName` and `lastName` joined by a space

The first one the entry supplies is its key, and it is compared against the same
key computed for the candidate node. For everything except users this is the
description and nothing changes. For a user it is normally the email address,
which is the right key: it is what the person logs in with, it is unique in
practice, and it survives the name changes that a first-and-last-name key would
not.

```yaml
nodes:
  - user:
      firstName: Admin
      lastName: User
      email: admin@example.com
      pass: $2a$... # bcrypt hash, or a ${SIOT_ADMIN_PASS} expansion later
      edgePoints:
        - type: role
          text: admin
```

Applied to a fresh instance this creates the user; applied again it matches on
`admin@example.com` and sends nothing; change `lastName` in the file and it
updates that one point on the existing user rather than creating a second
account. A file that supplies neither a description nor a user's identifying
points falls back to the singleton rule above.

This is also a small addition to `data`: a `Points.MatchKey()` alongside
`Desc()`, kept separate because `Desc()` prefers the name over the description
for display and matching should not inherit that.

**The match key is a node's identity, and changing one has consequences.**
Because provisioning and `siot import` both match on it, changing a node's
description in the UI -- or a user's email address -- detaches it from the file
that describes it: the next pass finds nothing to match and creates a second
node beside the renamed one. The same applies to a file that renames a node,
which reads as "create a new one" and leaves the old one in place. Neither case
is an error the software can detect, so both belong in the documentation, next
to the guidance to give provisioned nodes descriptions that are meant to last.
Renaming deliberately is a two-step change: `delete:` the old description in the
same file that introduces the new one.

Because matching does the work, `id:` in a hand-written file means "a label
other entries in this file can point at". Points of type `nodeID` are resolved
through those labels to whatever ID the matched or created node turned out to
have, so a rule can refer to a Modbus node in the same file without either one
knowing its UUID. In an exported file the same field holds a UUID, which is used
when creating a node that does not match anything and is otherwise just a label
like any other.

**Nodes attach under the instance's own device node, and `parent:` names a
description.** A top-level entry with no `parent:` is applied under the root
device node. A `parent:` names the description of a node anywhere in the tree,
which is how a file adds to a subtree it did not create, including one another
file created earlier. Not found, or found more than once, is an error recorded
against that entry. Children nest structurally, so `parent:` is only meaningful
on a top-level entry.

Entries apply in the order they appear, so a `parent:` naming a node the same
file creates has to come after the entry that creates it. Resolving the list in
dependency order instead would be friendlier, but "top to bottom" is a rule a
person can hold in their head while reading a file, and the error when it is
violated says exactly which description was not found.

**Applying a file patches; it does not own.** A file is a patch applied to the
tree: the nodes and points it names converge on what it says, and everything
else is left alone. Nothing tracks which nodes came from which file, and nothing
is removed because it stopped being mentioned. This is what makes one file
usable across a device's whole life -- the same mechanism initializes a fresh
unit, updates a fleet in the field, and patches one setting on a unit that has
diverged. It also means a file can be small: a file that sets one point on one
node is a perfectly good file to import or provision.

Points follow the data model here rather than needing a rule of their own: SIOT
does not remove points. A point a file stops mentioning keeps its last value,
where a client that no longer needs it ignores it, and a file that wants a
different value sets one. So reconciliation is a node-level concern, and
`delete:` names nodes.

**Node removal is explicit.** A `delete:` list names nodes to remove, matched
exactly the way `nodes:` entries are matched:

```yaml
delete:
  - modbus:
      description: Old sensors
  - group:
      description: Legacy
      parent: Sensors
```

A match is deleted along with its children, no match is a no-op, and more than
one match is an ambiguity error. Deletion in SIOT tombstones the edge rather
than erasing data, so the operation is recoverable. Explicit removal fits the
patch model: a release that retires a node ships a file that deletes it, and
that file is as safe to re-apply as any other.

**Nothing replaces the root node any more.** `client.ImportNodes` currently
treats the literal parent `root` as "create a new root node and tombstone the
old one", which is why `server.watchRoot` exists to exit the process afterwards.
Descriptive matching removes the need for that: importing an exported tree at
root matches the existing device node and updates it in place, and a restore
into a fresh instance still creates everything because nothing matches. This
plan removes the replacement path from `ImportNodes`, and with it the only thing
that triggers the root watcher. Removing `watchRoot` itself is proposed in the
same phase, since a watcher whose trigger no longer exists is a thing to explain
rather than a safeguard.

The one behavior that goes away is "restore this backup over the top of a
populated instance and have the old tree disappear". Under the new model that
restore merges, and anything the backup does not mention survives. Starting from
an empty instance, or naming the unwanted nodes in a `delete:` list, both give
the old result.

**Upstream sync is unaffected.** Provisioning creates a node once and updates
that same node from then on, so a node's ID is stable for the life of the store
and an upstream sees the same node it has always seen. The exception is deleting
the database and starting over, which generates new IDs for everything: at that
point all bets are off, and it is a new instance as far as an upstream is
concerned. That is true of a rebuilt instance however it was configured, not
something provisioning introduces.

**Provisioning state lives in the tree.** A `provisioning` node under the root
holds one `provisioningFile` child per file on disk, each carrying three points
of types that already exist:

- `description` -- the file's path relative to the provisioning directory. This
  is the file name, and putting it in `description` rather than the unused
  `filePath` type means the UI displays it and the node follows the same
  identity rule as every other node in the tree.
- `hash` -- the SHA-256 of the file contents, hex encoded, as text. Its
  timestamp doubles as "when this file was last acted on", since the hash is
  only written when it changes.
- `error` -- the last failure, empty when the file applied cleanly.

Files uploaded through the UI keep the same state on the file node itself,
covered under the next decision.

Storing state as nodes and points, rather than in a side table or a JSON file
next to the database, keeps it consistent with the nodes it describes: wiping
the store wipes the state. It is also visible in the UI, which makes "why did
this file not take effect" answerable without reading log files.

The fallback in `Pages/Home_.elm` would render these through `NodeRaw.elm`, but
that shows every point as an editable input with a delete button, so recorded
state reads as configuration a user should change, and it never shows the
timestamp that records when a file was last applied. Provisioning gets proper
components instead, in a phase of their own.

In the store this is nothing new. Each node is a row in `edges` carrying the
node type -- root to `provisioning`, `provisioning` to each `provisioningFile`
-- and each point is a row in `node_points` keyed by the node's ID. Only the
node type constants are new; no schema change and no migration.

It does mean the state replicates upstream, since `client.Sync` walks every
local node from the root without filtering by type. That reads as a feature
rather than a cost -- an upstream can see which file version each unit applied
and whether any of them failed -- but it is worth knowing that these nodes are
fleet-visible rather than local bookkeeping.

**Provisioning files can also live in the tree, uploaded through the UI.** A
`file` node child of the `provisioning` node is a provisioning file, applied the
same way a file on disk is. This needs almost nothing new: the `file` node type
exists, `NodeFile.elm` already uploads and downloads contents, and
`client/file.go` maintains the size and hash of whatever is stored. What it adds
is a way to provision a unit whose filesystem you cannot reach -- a device in
the field, a container someone else operates -- through the same interface used
to look at it.

It also opens a path worth naming even though this plan does not depend on it:
because these are ordinary nodes, an upstream instance that syncs with a device
can add a file node under that device's `provisioning` node, and the device
applies it. That is configuration pushed to a fleet without shell access on any
of it. The sync client walks and replicates the whole tree, so the mechanism is
already there, but it deserves a test of its own before it is documented as a
supported workflow.

The two sources behave identically, with three rules to settle the places they
could differ:

- **Order.** Files on disk apply first, in lexical order by path, then file
  nodes, oldest first by the time they were created. An image ships the baseline
  and the tree layers on top, which is also the order that makes a `parent:`
  reference from an uploaded file to a shipped group work. Files on disk keep
  the `10-`, `20-` prefix convention, which is how a person reading a directory
  sees the order; file nodes apply in the order they were added, which is how a
  person watching a UI expects them to behave.
- **Change detection.** The engine hashes the decoded contents with SHA-256
  itself rather than reading the `hash` point the file client maintains, so both
  sources are compared the same way and neither depends on the other's timing.
- **Names.** A source's name appears in the `Origin` of every point it applies,
  and labels its state node when it has one. For a file on disk it is the path
  relative to the provisioning directory, and it orders the file as well. For a
  file node it is the `description` point, falling back to `name` when there is
  no description -- the same preference `MatchKey` uses, and the one
  `NodeFile.elm` makes editable. It names the source without ordering it, since
  `created` does that. Two sources sharing a name is an error recorded against
  the second, on the same grounds as an ambiguous node match: the alternative is
  applying one of them and leaving the operator to work out which.

**File nodes record when they were created, and that is what orders them.** A
new `created` point on the `file` node holds the time the node came into
existence, written once and never rewritten. `client/file.go` sets it when it
first sees a file node without one, so it covers nodes created through the UI,
over the API, by an import, or by sync, and it does not depend on the browser's
clock.

Written once is the point of it. The `data` point already carries a timestamp,
but that one moves on every upload, so ordering by it would push a corrected
file behind files it used to precede -- fixing a typo would change the result. A
`created` point survives re-uploads, so a file keeps its place in the order for
as long as it exists.

Two details fall out. Nodes that predate this point, or arrive without one, sort
after those that have one, by name, so the order stays deterministic. And a
device whose clock is wrong when a file node is created carries that wrong time
for the life of the node, which is worth a line in the documentation: on
hardware without an RTC, uploads made before the clock settles can order
strangely, and re-creating the node is the fix.

State is the one place the two sources are recorded differently, and
deliberately so. A file on disk has nowhere in the tree to keep its status, so
it gets a `provisioningFile` node. A file node is already in the tree, so its
status goes on the node itself: a `provisionHash` point holding the SHA-256 the
engine last applied, and an `error` point. `hash` is unavailable for this
because the file client already uses it for the md5 of the contents, which is a
different question -- what the file is, rather than what provisioning has done
with it.

Keeping the two in one node is what makes the UI honest. An operator uploads one
file and sees one node, with its status on it. The alternative, a state node
beside every uploaded file, means two nodes per upload, a `nodeID` point to
associate them, UI work to display them as one thing, and a cleanup path for
when a file node is deleted and its state node is left behind. Putting the
points on the file node removes all four: deleting the file deletes its status
with it.

**Points carry the origin of whatever applied them.** Every point the engine
sends gets an `Origin`: `provision:<source name>` from a provisioning pass and
`import` from `siot import`, which is what import already sets today. Point
origin is part of the data model and is recorded in history, so the answer to
"what changed this point" includes the file that changed it.

**User edits are not fought over.** Provisioning acts on a source only when its
checksum changes, as described below. A value edited in the UI stays edited
until the file that describes it changes, at which point the file wins. This is
the one place the two callers differ in effect rather than mechanism:
`siot import` applies the file it is given every time, because a person asked it
to. Marking provisioned nodes read-only in the UI is a larger change and is out
of scope here. A deployment that would rather reassert its configuration on
every pass wants an option in the file header, not a different default; that is
listed under later refinements.

**Watch with fsnotify, subscribe for the tree, rescan on a timer as a
backstop.** fsnotify is already a dependency and is already used this way in
`client/serial.go` and `client/gps-serial.go`. Events are debounced so that a
directory written by a package manager or an editor is processed once rather
than per file. File nodes are watched the way everything else in SIOT is, with a
NATS subscription for point changes under the `provisioning` node, so an upload
applies as soon as it lands. A periodic rescan (default 60 seconds) backstops
both: it covers what fsnotify does not see, such as network and overlay
filesystems, and a missed point message, and costs one hash per source when
nothing has changed.

**The checksum decides whether to apply; idempotence makes applying safe.**
These are two mechanisms doing two jobs, and it is worth being explicit that
both are wanted.

Idempotence is what makes re-applying harmless. Without any checksum the system
would still be correct: every pass would fetch the tree, plan each source, find
nothing differs, and send nothing.

The checksum is what makes not re-applying the default, and that is a policy
choice rather than an optimization. A pass that reapplied every source every
minute would overwrite a value someone edited in the UI within the minute.
Recording a hash per source, and acting only when it changes, is precisely what
"user edits are not fought over" means in code. The saved work is a side
benefit: when no hash has changed, the pass does no tree fetch at all.

For file nodes the same hash does one more job. The point subscription makes an
upload apply immediately, but a message can arrive while the server is down, or
during a restart, or from sync before the provisioning actor starts, so "has
this been applied" cannot be answered by having seen a message. It is answered
by comparing the source's current hash against the one recorded on its state
node, which is also what makes a restart correct rather than merely quiet.

The engine hashes decoded contents itself rather than reading the `hash` point
that `client/file.go` maintains. That point is md5 written asynchronously after
a `data` point lands, so depending on it would tie provisioning to another
client's timing for no gain.

**Sources are processed in order, and one bad file does not stop the rest.**
Ordering makes cross-file dependencies expressible with the familiar `10-`,
`20-` prefix convention: a file that attaches to a `parent:` created by an
earlier file only needs to sort after it. A source that fails to parse or apply
is logged, records its error on its state node, and leaves the others alone.

## Design Sketch

Three layers, each in the package that can reach the ones below it.

**The format, in `data`.** New file `data/nodeyaml.go`. Both `client` and
`server` parse and write files, and the structures being encoded already live
here:

```go
// NodeFile is a whole file: what siot export writes, and what import
// and provisioning read.
type NodeFile struct {
    APIVersion int        `yaml:"apiVersion,omitempty"`
    Nodes      []NodeYAML `yaml:"nodes,omitempty"`
    Delete     []NodeYAML `yaml:"delete,omitempty"`
}

// NodeYAML wraps NodeEdgeChildren with the YAML spelling described above.
// MarshalYAML writes each point in its short form where that is lossless
// and falls back to a points: entry otherwise.
type NodeYAML struct {
    NodeEdgeChildren
    Label  string // id: -- a UUID from an export, or a local label
    Parent string // parent: -- a match key, not an ID
}

func (n NodeYAML) MarshalYAML() (any, error)
func (n *NodeYAML) UnmarshalYAML(b []byte) error
```

`data.NodeFile` replaces `client.SiotExport`.

**The engine, in `client`.** New file `client/apply.go`, shared by `ImportNodes`
and by provisioning. Matching needs to see the tree, so it works against a
snapshot fetched once per pass, which keeps the planning half pure and testable
without NATS:

```go
type ApplyOptions struct {
    Origin string // "import", or "provision:<path>"
    DryRun bool
}

// what a pass has to do
type ApplyPlan struct {
    Send   []data.NodeEdge // create or update, parent before child
    Delete []string        // node IDs to remove, with their children
    Errors []error         // per-entry failures; the rest of the plan still runs
}

func (p ApplyPlan) String() string // what siot provision and -dryRun print

// find the nodes matching an entry under parentID, by Points.MatchKey()
// or by type when the entry has no key; the caller checks the type of a
// single match against the entry
func match(tree []data.NodeEdge, entry data.NodeYAML, parentID string) []data.NodeEdge

// pure: given the file and a snapshot of the tree, what has to happen
func planApply(f data.NodeFile, tree []data.NodeEdge, root string, o ApplyOptions) ApplyPlan

// fetch the tree, plan, send, and delete
func Apply(nc *nats.Conn, f data.NodeFile, o ApplyOptions) (ApplyPlan, error)
```

`ImportNodes` becomes a parse and an `Apply`.

**Provisioning, in `server`.** New file `server/provision.go`: the directory
scan, the file node scan, the checksums, and the watcher. It parses each source,
calls `client.Apply` with `Origin` set to `provision:<name>`, and records what
happened:

```go
// one source for a pass, from either the directory or the tree
type provisionSource struct {
    Name     string    // relative path, or the file node's description
    Created  time.Time // zero for a file on disk, which sorts by name
    NodeID   string    // set when the source is a file node
    Contents []byte
    Hash     string // hex SHA-256 of Contents
}

// what was recorded last time, read from a provisioningFile node for a
// file on disk and from the file node's own points for a file node
type provisionState struct {
    Hash    string
    Applied time.Time
    Error   string
}

func (p *provisioner) sources() ([]provisionSource, error)
func (p *provisioner) state(s provisionSource) (provisionState, error)
func (p *provisioner) record(s provisionSource, st provisionState) error
```

Only `state` and `record` care which kind of source they have, and only to pick
where the points go.

Wiring in `server.Run` follows the pattern already there: an actor added to the
`run.Group` that waits on `siotStore.WaitStart`, does an initial pass, then
serves the fsnotify channel and the rescan ticker until cancelled. The actor is
a no-op when `Options.ProvisioningDir` is empty.

## Phases

Commit after each phase, updating the changelog and any affected documentation
as part of that phase, per `CLAUDE.md`.

### Phase 1 -- Node YAML codec, export, and import

`data/nodeyaml.go` and `data/nodeyaml_test.go`, then `client/node.go` and
`server/server.go`.

- `data.NodeFile` and `data.NodeYAML` with `MarshalYAML` and `UnmarshalYAML`:
  node type as key, point types as keys, value kinds mapped as above, keyed and
  array points, reserved `id` / `parent` / `children` / `points` / `edgePoints`,
  and the per-point fallback to the long form when the short spelling would lose
  something. Decode through `yaml.MapSlice` and `any` so that `9600` and
  `"9600"` stay distinguishable.
- Emit points in a stable order, sorted by type then key, so that exports of an
  unchanged tree produce identical files and diffs stay readable.
- Reject a file in the old format with an error naming what it saw.
- Switch `client.ExportNodes` to write `data.NodeFile`, and retire
  `client.SiotExport`.
- Update the format example in the `ExportNodes` doc comment.

Unit tests cover each value kind, quoted numbers landing in `Text`, keyed and
array points, points that must fall back to the long form (`Data`, tombstone,
origin, a point of type `id`), edge points, the old-format error, and a round
trip from a node tree through YAML and back that compares equal. The shared
fixture under Testing below is parsed here too, so that a file the later phases
depend on is known to decode before anything applies it.

### Phase 2 -- Apply engine and import

`client/apply.go`, `client/apply_test.go`, `client/node.go`, `cmd/siot/main.go`,
and `server/server.go`.

- Add `data.Points.MatchKey()`: description, else email, else first and last
  name, with tests including a user node and a node that has none of them.
- Match entries against a tree snapshot by parent and match key, falling back to
  type for an entry with no key; resolve `parent:` keys; report a type mismatch,
  an ambiguous match, and a missing parent per entry.
- Resolve `id:` labels and `nodeID` point references to real node IDs, and use
  an entry's `id:` as the new node's ID when it is a UUID that is not already
  taken.
- Build and apply a plan: create missing nodes, send only the points that
  differ, delete matched `delete:` entries with their children. `ApplyPlan`
  prints itself for `-dryRun`.
- Rebuild `ImportNodes` on `Apply`: drop the `(import)` description suffix, the
  root replacement path, and the `ReplaceIDs` and `checkIDs` helpers with their
  tests. Remove `server.watchRoot` and its test, whose trigger is now gone.
- Drop `-parentID` and `-preserveIDs` from `siot import`, add `-dryRun`.

Unit tests over a fixed tree snapshot, where `planApply` is pure and needs no
server: create when absent, no-op when identical, update only the differing
point, match under a `parent:` key, ambiguous match, a match whose type differs
from the file, missing parent, singleton match with no key, a user matched by
email across a changed last name, `delete:` hitting and missing, `nodeID`
resolution between two entries in one file, an entry whose UUID `id:` is honored
on create and one whose UUID is already taken, and a second pass over the result
of the first producing an empty plan.

The import and export tests against a running test server -- the ones built on
the shared fixture -- are described under Testing below and land in this phase.

### Phase 3 -- Provisioning sources

`server/provision.go` and `server/provision_test.go`.

- Add `data.PointTypeCreated` and `data.PointTypeProvisionHash`, and `Created`,
  `ProvisionHash`, and `Error` fields on `client.File`. `client/file.go` sets
  `created` when it sees a file node without one, written once and never
  rewritten on a later upload; provisioning owns the other two.
- Collect the sources for a pass: files in the directory, in lexical order by
  relative path, then `file` node children of the `provisioning` node, oldest
  `created` first, with any that lack one sorting last by name. Name each source
  -- the relative path, or the file node's `description` falling back to `name`
  -- and hash its contents with SHA-256; report a duplicate name against the
  second source that carries it.
- Parse each source, call `client.Apply` with `Origin` set to
  `provision:<source name>`, and record the outcome.
- Record state per source: a `provisioningFile` node carrying description, hash,
  and error for a file on disk; `provisionHash` and `error` points written back
  onto the file node itself for a file node.
- Reconcile state nodes with the directory, so that a file that is gone takes
  its state node with it. File nodes need no equivalent, since their state goes
  when they do.
- Skip sources whose hash is unchanged, and isolate per-source failures.
- Reject an unknown `apiVersion`.

### Phase 4 -- Lifecycle

- Add `Options.ProvisioningDir` and `Options.ProvisioningInterval` and the
  `run.Group` actor in `server.Run`.
- fsnotify watch on the directory, with debounce, plus the rescan ticker.
- Subscribe to point changes under the `provisioning` node so that an uploaded
  or edited file node triggers a pass.
- Handle the directory not existing yet: log once, keep watching the parent so
  that creating it later starts provisioning without a restart. A server with no
  provisioning directory at all still serves file nodes.
- Add an options-taking variant of `server.TestServer` so tests can start a
  server with a provisioning directory.

The provisioning tests under Testing below are written against this helper and
land in this phase. They use the same fixture the import tests use, split across
two files for the ordering case.

### Phase 5 -- CLI and configuration

- `-provisioningDir` flag and `SIOT_PROVISIONING_DIR` environment variable in
  `server.Args`, defaulting to `<SIOT_DATA>/provisioning` when that directory
  exists so that images can simply ship the directory.
- `SIOT_PROVISIONING_INTERVAL` for the rescan period.
- `siot provision -dir <dir>` subcommand that prints the plan for a whole
  directory against the running instance without applying it, which is
  `siot import -dryRun` over more than one file. A `-check` mode that only
  parses and reports, for use in CI, where there is no instance to match
  against.

### Phase 6 -- Provisioning UI

`frontend/src/Components/`, `frontend/src/Api/Node.elm`, and
`frontend/src/Pages/Home_.elm`.

- `NodeProvisioning.elm`: the parent node, showing how many sources are
  configured and how many are in error, so a problem is visible without
  expanding anything.
- `NodeProvisioningFile.elm`: read-only status for a file on disk -- file name,
  short hash, when it was applied, and the error in red when there is one. This
  is what the raw point table cannot say, and it is the reason this phase is
  here rather than in the refinements.
- Show `created` in `NodeFile.elm` for every file node, next to the file name
  and size it already shows. The file client writes that point on all of them,
  not only the ones under `provisioning`, and "when was this file added" is
  worth answering anywhere. For a provisioning file it is also the sort key, so
  showing it makes the apply order legible from the file itself.
- Show the provisioning status in `NodeFile.elm` only when a `provisionHash`
  point is present, so that file nodes used for anything else -- a CAN bus
  database, a serial configuration -- are untouched: when it was last applied,
  the short hash, and the error in red when there is one. "Uploaded but not
  applied" then reads as the `data` point's timestamp being newer than the
  applied one.
- Add `typeProvisioning` and `typeProvisioningFile` to `Api/Node.elm`, a
  `nodeDescProvisioning` label, and both types to the dispatch in
  `Pages/Home_.elm`.
- Allow uploading provisioning files: add `provisioning` to
  `nodeTypesThatHaveChildNodes` so the node grows an add button, and offer
  `file` under a `provisioning` parent in `viewAddNode`, the way `canBus` and
  `serialDev` already do. `NodeFile.elm` handles the upload itself.

`elm-review` and `elm-test` pass, and the components are exercised by hand
against a running instance, since the frontend has no test harness for views.

### Phase 7 -- Documentation and examples

- Rewrite the export and import examples in `docs/user/configuration.md` in the
  new format, with the value mapping table and the quoting rule, since those
  examples now describe provisioning files as well.
- Document import and provisioning together: one format, one set of rules for
  applying it, one file that works either way. The old import documentation
  describes flags that no longer exist.
- New "Configuration provisioning" section in the same file covering both
  sources -- the directory and file nodes uploaded through the UI -- the order
  they apply in, the match key and what makes a match ambiguous, `parent:`
  lookup, the patch model, `delete:`, and the state nodes. Say plainly that file
  nodes apply in creation order, that re-uploading does not change a file's
  place, and that on hardware without an RTC a file created before the clock
  settles carries that time until the node is re-created.
- A prominent note in that section, and in the import documentation, that the
  match key is how a file finds a node: changing a description, or a user's
  email, detaches it from the file, and the next pass creates a second node
  beside it. Include the two-step rename -- `delete:` the old key in the file
  that introduces the new one -- and the advice to choose descriptions that are
  meant to last.
- A worked example of provisioning a user, since users match on email rather
  than description and that is the one place the rule is not obvious.
- Document the new environment variables in the same file's variable list.
- Update any other node YAML examples in `docs/`, including `docs/ref/data.md`
  and the client pages that show a node in YAML.
- Note in `docs/ref/architecture-app.md` that provisioning is a server-level
  concern rather than a client, with a pointer to the user documentation.
- An example directory under `contrib/` or `docs/user/assets/` with two files
  that demonstrate ordering and a cross-file `parent:`.
- Changelog entries, this being a release with breaking changes worth spelling
  out: the provisioning feature and its defaults; the new file format, with the
  note that old files are not read and that re-exporting from an upgraded
  instance produces the new form; and the import changes -- idempotent, no
  `(import)` suffix, no `-parentID` or `-preserveIDs`, and no root replacement.

### Phase 8 -- Optional refinements

Worth doing only if the earlier phases prove out and there is demand:

- `${ENV_VAR}` expansion in provisioning files, so tokens and passwords come
  from the environment rather than from a file in version control. Grafana does
  this and it is the usual reason provisioning files stay committable.
- `disabled: true` in a file header, to keep a file in place without applying
  it.
- `enforce: true` in a file header, applying that file on every pass rather than
  only when its checksum changes, for a deployment that wants configuration
  reasserted over anything a user changes in the UI. The engine already plans
  every pass the same way; this only bypasses the hash comparison.
- Editing a provisioning file node in the UI rather than only uploading one,
  which turns the provisioning node into a place to write configuration as well
  as apply it.
- Pushing a provisioning file from an upstream to a downstream instance over
  sync, which the tree-source design already allows and which needs a test
  before it is documented.
- Marking provisioned nodes in the UI, using the provisioning origin already on
  their points, so it is visible that a node is managed from disk.
- A point type schema, so a file that writes a number where a client expects
  text is reported by `siot provision -check` and by `siot import` instead of
  producing an empty value. This is the one gap the short form opens, and it
  would also give the UI and the documentation a single place to look up what a
  point type means.
- A stable match key stamped on provisioned nodes, preferred over the
  description when matching, so that a rename in the UI keeps the node attached
  to its file. Worth doing only if renaming turns out to be a problem the
  documentation does not solve.

## Testing

### One fixture, three paths

Import, export, and provisioning are the same operation reached three ways, so
they are tested against one fixture rather than three sets of ad hoc nodes. The
fixture lives in `client/testdata/tree.yaml`, and `server/provision_test.go`
reads it from there and copies it into a temporary provisioning directory, so
that a change to the fixture is felt by every test that depends on it.

It is deliberately more than a flat list: nested groups so there is real depth,
top-level entries carrying `parent:` so description lookup is exercised rather
than only structural nesting, a cross-reference between two entries, a user node
so the email match key is covered, and an `edgePoints:` entry so the long form
is not left to a unit test on its own.

```yaml
# client/testdata/tree.yaml
apiVersion: 1
nodes:
  - group:
      description: Sensors
      children:
        - group:
            description: Tank farm
  # attaches by description to a group created above, not by nesting
  - modbus:
      parent: Tank farm
      description: Modbus sensors
      port: /dev/ttyS1
      baud: 9600
      children:
        - modbusIo:
            description: Tank level raw
            modbusIoType: holdingRegister
            address: 40001
            scale: 0.1
  - variable:
      parent: Tank farm
      id: tank-level
      description: Tank level
      variableType: number
  - group:
      description: Rules
      children:
        - rule:
            description: Tank low
            children:
              - condition:
                  description: Level below 10
                  conditionType: pointValue
                  nodeID: tank-level # resolved to the variable created above
                  pointType: value
                  operator: "<"
                  value: 10
              - action:
                  description: Notify on tank low
                  actionType: notify
  - user:
      firstName: Admin
      lastName: User
      email: admin@example.com
      edgePoints:
        - type: role
          text: admin
```

### Import, in `client/apply_test.go`

Against a `server.TestServer`:

- Applying the fixture to an empty instance builds the expected tree: walk from
  the root and compare types, descriptions, and parents; assert the two entries
  with `parent: Tank farm` landed under that group and not at the root; assert
  the condition's `nodeID` point holds the variable's actual ID.
- Applying it a second time is a no-op: the plan comes back empty, and the tree
  compares equal to what it was.
- Changing one point in the fixture and applying it updates that point and sends
  nothing else.
- An entry whose `parent:` names something that does not exist, one that names a
  description matching two nodes, and one whose type differs from the node it
  matched each produce an error naming the entry, while the rest of the file
  still applies.
- A `delete:` entry removes the node and its children; running it again is a
  no-op.

### Export, in `client/node_test.go`

- Export the tree built from the fixture and import it into a second empty
  instance: the two trees compare equal, including node IDs, since the exported
  file carries UUIDs.
- Export that second instance and compare the bytes to the first export. Equal
  output is what makes exports reviewable in version control, and it is the test
  that catches unstable point ordering.
- Export, import into the _same_ instance, and confirm nothing changed, which is
  import idempotence coming from the other direction.

### Provisioning from a directory, in `server/provision_test.go`

Using the options-taking `server.TestServer` from Phase 4:

- Start a server with the fixture in its provisioning directory: the tree
  appears, and the `provisioningFile` state node carries the file's description,
  hash, and an empty error.
- Restart against the same store and directory: nothing is sent, no node is
  duplicated, and the state node is unchanged.
- Edit a point in the file while the server runs: the change lands within the
  debounce window and the recorded hash changes with it.
- Add a `delete:` entry: the node goes away.
- Split the fixture into `10-groups.yaml` and `20-sensors.yaml`, where the
  second attaches by `parent:` to a group the first creates: both apply, in
  lexical order.
- A file with a syntax error records the error on its state node and leaves the
  other file's nodes alone.
- Remove a file: its nodes stay, since provisioning does not own them, and its
  state node goes away.

### Provisioning from the tree, in `server/provision_test.go`

The same fixture, sent as a `file` node rather than written to disk, so that the
two sources are demonstrably the same mechanism:

- Add a `file` node under the `provisioning` node with the fixture as its
  contents: the file client stamps a `created` point, the tree appears without
  waiting for the rescan ticker since the subscription fires, and the file node
  itself carries a `provisionHash` point and an empty `error`. No second node
  appears beside it.
- Send the same contents again: nothing is applied.
- Change the contents: the change lands, `provisionHash` follows, and `created`
  is unchanged.
- Two file nodes created in a known order, where the second attaches by
  `parent:` to a group the first creates: both apply, oldest first. Re-upload
  the older one and confirm it keeps its position, which is the case a `data`
  timestamp would get wrong.
- A file node with no `created` point sorts after those that have one.
- Delete the file node: the nodes it provisioned stay, since provisioning does
  not own them, and its status goes with it, matching what removing a file from
  the directory does.
- A file node whose contents do not parse records the error on its own `error`
  point, and a directory file in the same pass still applies. Fixing the
  contents clears the error.
- Run with a directory file and a file node together and confirm the order: the
  directory file applies first, and an uploaded file that names a `parent:` the
  directory file creates lands under it.
- Two sources with the same name: the second records an error and the first is
  applied.

### Everything else

- `go test -race ./data/... ./server/... ./client/...`
- `golangci-lint run`
- `cd frontend && npx elm-review && npx elm-test`
- Manual: run `siot serve` with a provisioning directory, edit a file while the
  server runs, and confirm the change appears in the UI.
- Manual: upload a provisioning file through the UI, confirm the nodes appear
  and the status shows next to the file, then upload a corrected version and
  confirm the error clears.
- Manual: provision an instance that syncs to an upstream, apply a changed file,
  and confirm the upstream sees updates to the same nodes rather than new ones.
