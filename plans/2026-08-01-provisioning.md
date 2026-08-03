# Plan: Configuration Provisioning from YAML Files

**Branch:** `feat/provisioning` **Branched from:** `54169492`

## Context

SIOT can already export and import a tree of nodes as YAML
(`client.ExportNodes` / `client.ImportNodes`, exposed as `siot export` and
`siot import`). That is a good manual workflow, but it is a poor fit for
deployments that are built from an image or a configuration management system:

- Import is a one-shot push over NATS. The instance has to be running and
  someone (or some script) has to pipe the file in at the right moment.
- Import is not idempotent. By default every import assigns fresh IDs, so
  running the same file twice creates a second copy of everything. Running it
  with `-preserveIDs` avoids the duplicates but never removes anything, so a
  node dropped from the file lives on in the tree.
- There is no record of where a node came from. Nothing in the tree says "these
  nodes are owned by `sensors.yaml`", so nothing can reconcile them later.

Grafana solves the same problem with provisioning: a directory of YAML files is
read at start-up, the objects described there are created or updated, a
checksum of each file is recorded, and the files are re-read when they change.
The configuration lives in version control and on disk, and the running system
converges on it.

This plan brings that model to SIOT. A provisioning directory is scanned at
start-up and watched while running. Each file describes a subtree in the
existing export format. Nodes get deterministic IDs derived from the file, so
applying a file repeatedly converges instead of duplicating. A checksum per
file means unchanged files cost one hash and nothing else. Nodes that disappear
from a file, and nodes belonging to a file that is deleted, are removed.

The result is that an image can ship `/var/lib/simpleiot/provisioning/*.yaml`,
and every unit that boots that image comes up with the same configuration, with
no import step and no operator involvement.

### What this builds on

The working tree currently carries uncommitted changes that make an import at
`root` safe: the client manager re-resolves the root node on every scan, and
the server exits when the root node is replaced so a service manager can
restart it (`server.watchRoot`, `client.Manager.updateRoot`). This plan assumes
those changes land first. It deliberately stays away from replacing the root
node, for reasons covered under Design Decisions.

## Design Decisions

**Provisioning is a server concern, not a client node.** Everything else in
SIOT is configured by nodes in the tree, but provisioning has to work before
there is any configuration in the tree. Like `SIOT_DATA` and the NATS ports, it
is set by a command line flag and an environment variable and handled in
`server.Run`. This also matches Grafana, where provisioning is configured in
`grafana.ini` rather than in the database it populates.

**The file format is the existing export format with an optional header.** A
provisioning file is what `siot export` produces, so the natural workflow is to
build a configuration in the UI, export it, drop it in the provisioning
directory, and commit it. The header adds only what provisioning needs and is
optional:

```yaml
# provisioning/20-sensors.yaml
apiVersion: 1 # optional, reserved for future format changes
parent: "" # optional node ID; default is the root node
prune: true # optional, default true
nodes:
  - id: sensors-group
    type: group
    points:
      - type: description
        text: Sensors
    children:
      - id: sensor-modbus
        type: modbus
        points:
          - type: description
            text: Modbus sensors
```

Since `nodes:` is the only required key, an unmodified export file is a valid
provisioning file.

**Node IDs are derived from the file, not generated.** This is what makes
re-applying a file an update instead of a duplication. For each node:

```
namespace = uuidv5(NameSpaceURL, "https://simpleiot.org/provisioning")
nodeID    = uuidv5(namespace, "<file path relative to provisioning dir>#<local key>")
```

The local key is the node's `id:` field when the file supplies one, and
otherwise its positional path in the file (`0`, `0/1`, `0/1/2`). Cross
references between nodes, meaning points of type `nodeID`, are resolved through
the same map, so the friendly IDs that already work for `siot import` work here
too, and a rule can refer to a Modbus node in the same file by name.

Two consequences worth documenting: renaming a file changes every ID under it,
which reads as "delete the old subtree, create a new one"; and files that rely
on positional keys are sensitive to reordering, so giving nodes explicit `id:`
fields is the recommended practice.

A useful side effect is that the same file applied to two instances produces
the same IDs on both, which is what you want when those instances sync to a
common upstream.

**Provisioning never replaces the root node.** `client.ImportNodes` treats the
literal parent `root` as "create a new root node and delete the old one". If
provisioning did that, it would trip the root watcher, the server would exit,
the restarted server would find its provisioning state gone with the old tree,
and it would do the whole thing again on every boot. Provisioning therefore
resolves the actual root node ID and imports underneath it. A file that
specifies `parent: root` is rejected with an error pointing at
`siot import -parentID root`, which remains the way to do a full restore.

**Provisioning state lives in the tree.** A `provisioning` node under the root
holds one `provisioningFile` child per file, carrying the relative path, the
SHA-256 of the file contents, the resolved parent, the list of node IDs the
file owns, and the last error if the file failed to load. Storing state as
nodes and points, rather than in a side table or a JSON file next to the
database, keeps it consistent with the nodes it describes: wiping the store
wipes the state, so the tree and the record of how the tree was built can never
disagree. It is also visible in the UI, which makes "why is this node here"
answerable without reading log files. Unknown node types already render through
`NodeRaw.elm`, so this needs no frontend work to be inspectable.

**Points carry a provisioning origin.** Every point applied by provisioning
gets `Origin` set to `provision:<relative path>`. Point origin is already part
of the data model and is recorded in history, so the answer to "what changed
this point" includes the file that changed it.

**Removal is the default.** A node that a file used to own and no longer
describes is deleted, and deleting a file deletes everything it owned. A
declarative directory that only ever adds is not much better than the current
import. `prune: false` in a file header opts out for cases where SIOT owns part
of a subtree at runtime. Deletion in SIOT tombstones the edge rather than
erasing data, so the operation is recoverable.

**User edits are not fought over.** Provisioning acts on a file only when its
checksum changes. A value edited in the UI stays edited until the file that
describes it changes, at which point the file wins. Marking provisioned nodes
read-only in the UI is a larger change and is out of scope here.

**Watch with fsnotify, rescan on a timer as a backstop.** fsnotify is already a
dependency and is already used this way in `client/serial.go` and
`client/gps-serial.go`. Events are debounced so that a directory written by a
package manager or an editor is processed once rather than per file. A periodic
rescan (default 60 seconds) covers the cases fsnotify does not see, such as
network and overlay filesystems, and costs one hash per file when nothing has
changed.

**Files are processed in lexical order, and one bad file does not stop the
rest.** Ordering makes cross-file dependencies expressible with the familiar
`10-`, `20-` prefix convention. A file that fails to parse or apply is logged,
records its error on its state node, and leaves the other files alone.

## Design Sketch

New file `server/provision.go`, with the work split so the interesting part is
testable without a server:

```go
// parsed from a provisioning YAML file
type provisionFile struct {
    APIVersion int    `yaml:"apiVersion"`
    Parent     string `yaml:"parent"`
    Prune      *bool  `yaml:"prune"`
    Nodes      []data.NodeEdgeChildren `yaml:"nodes"`
}

// what a file owns, as recorded in the tree
type provisionState struct {
    Path    string
    Hash    string
    Parent  string
    NodeIDs []string
    Error   string
}

// pure: given the file on disk and the previous state, what has to happen
type provisionPlan struct {
    Send   []data.NodeEdge // create or update, in parent-before-child order
    Delete []string        // node IDs to remove
}

func provisionIDs(nodes []data.NodeEdgeChildren, sourceKey string) // deterministic IDs, in place
func planFile(f provisionFile, sourceKey, parent string, prev provisionState) (provisionPlan, error)
func (p *provisioner) apply(plan provisionPlan) error // NATS side effects
```

Wiring in `server.Run` follows the root watcher already there: an actor added
to the `run.Group` that waits on `siotStore.WaitStart`, does an initial pass,
then serves the fsnotify channel and the rescan ticker until cancelled. The
actor is a no-op when `Options.ProvisioningDir` is empty.

Creating nodes does not disturb the root watcher, which only acts when the root
node ID itself changes.

## Phases

Commit after each phase, updating the changelog and any affected documentation
as part of that phase, per `CLAUDE.md`.

### Phase 1 -- Engine

`server/provision.go` and `server/provision_test.go`.

- Parse a provisioning file, including the optional header, rejecting
  `parent: root` and unknown `apiVersion` values.
- Derive deterministic IDs and resolve `nodeID` cross references.
- Read and write provisioning state nodes under the root.
- Compute a plan from (file, previous state) and apply it: send nodes
  parent-first, delete owned nodes that are no longer described.
- Scan a directory once, in lexical order, skipping files whose hash is
  unchanged, isolating per-file failures.

Unit tests cover ID stability across runs and across identical files in
different directories, cross-reference resolution, and the plan for each case:
first apply, unchanged, changed point, added node, removed node, removed file,
`prune: false`.

### Phase 2 -- Lifecycle

- Add `Options.ProvisioningDir` and `Options.ProvisioningInterval` and the
  `run.Group` actor in `server.Run`.
- fsnotify watch on the directory, with debounce, plus the rescan ticker.
- Handle the directory not existing yet: log once, keep watching the parent so
  that creating it later starts provisioning without a restart.
- Add an options-taking variant of `server.TestServer` so tests can start a
  server with a provisioning directory.

Integration tests using that helper: a file appears and its nodes appear; a
point edited in the file is applied; a node dropped from the file is removed; a
deleted file removes its subtree; a server restarted against the same store and
directory does no work and creates no duplicates.

### Phase 3 -- CLI and configuration

- `-provisioningDir` flag and `SIOT_PROVISIONING_DIR` environment variable in
  `server.Args`, defaulting to `<SIOT_DATA>/provisioning` when that directory
  exists so that images can simply ship the directory.
- `SIOT_PROVISIONING_INTERVAL` for the rescan period.
- `siot provision -dir <dir>` subcommand that validates and prints the plan
  without touching a running system, so a build can fail on a bad file. A
  `-check` mode that only parses and reports, for use in CI.

### Phase 4 -- Documentation and examples

- New "Configuration provisioning" section in `docs/user/configuration.md`
  covering the directory, the file format, ID derivation and why file names and
  `id:` fields matter, removal behavior, the relationship to `siot import`, and
  the state nodes.
- Document the new environment variables in the same file's variable list.
- Note in `docs/ref/architecture-app.md` that provisioning is a server-level
  concern rather than a client, with a pointer to the user documentation.
- An example directory under `contrib/` or `docs/user/assets/` with two files
  that demonstrate ordering and a cross-file reference.
- Changelog entry describing the feature and its defaults.

### Phase 5 -- Optional refinements

Worth doing only if the earlier phases prove out and there is demand:

- `${ENV_VAR}` expansion in provisioning files, so tokens and passwords come
  from the environment rather than from a file in version control. Grafana does
  this and it is the usual reason provisioning files stay committable.
- `disabled: true` in a file header, to keep a file in place without applying
  it.
- Marking provisioned nodes in the UI, using the provisioning origin already on
  their points, so it is visible that a node is managed from disk.

## Testing

- `go test -race ./server/... ./client/...`
- `golangci-lint run`
- Manual: run `siot serve` with a provisioning directory, edit a file while the
  server runs, confirm the change appears in the UI within the debounce window,
  delete the file, confirm the nodes are removed and the state node is gone.

## Open Questions

- **Default for `prune`.** This plan defaults to removing nodes, on the grounds
  that a declarative directory should converge. The conservative alternative is
  to default to `false` and require opting in. Worth a decision before Phase 1
  since it shapes the documentation more than the code.
- **Point-level removal.** The plan reconciles nodes: a node that leaves the
  file is deleted. It does not reconcile points within a node, so a point
  removed from a file keeps its last value. Adding point-level removal means
  tombstoning points that the file no longer lists, which is a larger change
  and may surprise users whose runtime values live on provisioned nodes.
- **Provisioning and upstream sync.** When an instance both provisions from
  disk and syncs to an upstream that also has these nodes, the deterministic
  IDs make the two agree, but the interaction deserves a test before it is
  documented as supported.
