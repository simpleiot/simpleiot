# Plan: Generate New IDs as Crockford Base32

**Branch:** `cbrake/master` **Branched from:** `ee8f9054` **Issue:**
[#824](https://github.com/simpleiot/simpleiot/issues/824)

## Context

New node and instance IDs are UUIDs written as 36 characters. The same 128
random bits in lowercase Crockford base32 take 26 characters, which saves about
50 bytes per point on the sync link (three IDs in a point's subject, two in the
acknowledged stream name). See
[Shorter IDs](../docs/ref/bandwidth.md#shorter-ids).

This applies only to IDs generated from now on. Existing UUID IDs are not
converted. IDs are opaque strings everywhere except one place (NetworkManager,
below), so old and new IDs live side by side in the same tree, store, and sync
link with no migration.

What the code already gives us:

- Every generated ID comes from `uuid.New().String()`, at 19 call sites:
  `cmd/siot/main.go` (2), `server/enroll.go`, `server/provision.go` (2),
  `api/nodes.go` (node insert and notification), `store/jetstream.go` (root and
  admin user), `client/sparkplug.go` (3), `client/apply.go`,
  `client/onewire.go`, `client/mqtt-schema.go`, `client/rule.go`,
  `client/node.go` (duplicate), `client/shelly.go`, and `client/iio.go`.
- The UIs never generate IDs; they post a node with an empty ID and
  `api/nodes.go` fills it in.
- No code outside the NetworkManager client parses or validates an ID as a UUID.
  Two tests do (`client/node_test.go`, `client/apply_test.go`).

Not IDs, and unchanged:

- The Shelly RPC source name in `client/shelly-rpc.go` (a random suffix, not a
  node ID).
- NetworkManager's own connection UUIDs.

## Approach

1. **One generator, `data.NewID()`.** It reads 16 bytes from `crypto/rand` and
   encodes them with Go's `encoding/base32` using the alphabet
   `0123456789abcdefghjkmnpqrstvwxyz` and no padding. That gives 26 lowercase
   characters with the 2 leftover bits as zeros at the end, so the last
   character is one of `048cgmrw`. This matches the example in the issue and the
   bandwidth reference, and needs no custom encoder. Nothing decodes IDs, so the
   bit placement is documented but has no other effect. The result has 128
   random bits, up from the 122 of a version 4 UUID.
2. **Replace the 19 call sites.** Each `uuid.New().String()` that makes a node,
   instance, user, or notification ID becomes `data.NewID()`. The `google/uuid`
   import stays only where NetworkManager needs it.
3. **NetworkManager connections store their UUID in a point.** NetworkManager
   requires a UUID for each connection, and the client today uses the
   `networkManagerConn` node ID as that UUID, refusing a managed connection
   whose ID does not parse (`client/network-manager.go:617`). A connection added
   in the UI would now get a base32 ID and be refused. Instead, the
   NetworkManager UUID becomes its own point, `uuid`, a `UUID` field on
   `NetworkManagerConn`, and the node ID is just a node ID:
   - The client uses `UUID` wherever it crosses into NetworkManager: `DBus()`
     (the `uuid` setting), `ResolveNetworkManagerConn`, the `treeConnIDs` set
     and `nmConns` lookup, `deletedConns` keys, and the active-connection
     comparison. The parse check and its "invalid UUID" error go away.
   - A connection found in NetworkManager gets a node ID from `data.NewID()` and
     its NetworkManager UUID in the `uuid` point.
   - A node with no `uuid` point, such as one added in the UI, is given a fresh
     random UUID by the client on its next sync, and the point is written back
     before the connection is added to NetworkManager.
   - The connection editor shows the UUID read-only, so a node can be matched to
     `nmcli connection show` output.
4. **Comments that say IDs are UUIDs** are reworded to say IDs are opaque and
   contain no `_`: `client/dump.go:91`, `store/jetstream.go:113`,
   `store/store.go:66`, `server/server.go:81`, `data/node.go:36`, and
   `data/notification.go:16`.
5. **IDs chosen by hand are checked.** Today nothing stops an ID that breaks the
   system: a `.`, `*`, `>`, or space splits or wildcards a subject, and a `_` in
   an instance ID makes `inst_<boundary>_<origin>` ambiguous. Add
   `data.ValidID(id string) error`, which accepts 1 to 64 characters of ASCII
   letters, digits, and `-`, not starting with `-`. UUIDs and the new base32 IDs
   both pass. It is enforced where an ID is chosen rather than generated:
   - `--id` and `server.Options.ID` / `store.Params.ID`, at startup, so an
     instance with a bad ID refuses to start with a clear message instead of
     creating streams under it.
   - `insertNode` in `api/nodes.go`, when the caller supplies an ID,
     returning 400.
   - `SendNode` in `client/node.go`, before any points are sent, so a Go client
     does not leave points behind for a node that is then refused.
   - `handleEdgePoints` in `store/store.go`, for the node and parent ID of every
     edge message. This covers every client that writes edges over NATS.
     Replicated streams from a downstream are merged directly
     (`store/replica.go`) and not checked here; a downstream checks its own ID
     at startup. Checking every edge rather than only new nodes needs no lookup;
     generated IDs have always been UUIDs, so only a hand-picked ID that already
     breaks stream names is affected.

   Node files do not set node IDs (`client/apply.go` matches existing nodes or
   generates one), so they need no check.

## Phases

### Phase 1: Generator and call sites ✅

- Add `data.NewID()` in `data/id.go` with a test: length 26, only alphabet
  characters, last character in `048cgmrw`, and no repeats across a few thousand
  calls.
- Replace the 19 call sites.
- Add the `uuid` point to `NetworkManagerConn`, use it in the NetworkManager
  client, and show it read-only in the connection editor. Like the other
  NetworkManager point types, it is a literal in the struct tag and the editor
  rather than a schema constant. A unit test covers filling in a missing UUID.
- Update `client/node_test.go` and `client/apply_test.go` to check the new form
  rather than `uuid.Parse`.
- Update the comments listed above.
- Check: `go test -race ./...`, `golangci-lint run`.

### Phase 2: Check chosen IDs ✅

- Add `data.ValidID()` to `data/id.go` with a table test: UUIDs, base32 IDs, and
  IDs such as `plant-07` pass; empty, too long, leading `-`, and IDs with `_`,
  `.`, `*`, `>`, space, or non-ASCII fail.
- Check `--id` and the server and store ID options at startup.
- Check a supplied ID in `insertNode`, with an API test for the 400.
- Check every edge in `handleEdgePoints`, and the ID in `SendNode`. A server
  test (`server/id_check_test.go`) covers the API, a raw edge over NATS, and
  startup.
- Check: `go test -race ./...`, `golangci-lint run`.

### Phase 3: Documentation and changelog ✅

- `docs/ref/bandwidth.md`: change "Shorter IDs" from a proposal to what happens,
  and drop it from the list of items left out near line 390.
- Replace the UUID example IDs in `docs/ref/api.md`,
  `docs/user/configuration.md`, and `docs/user/mqtt.md` with base32 ones. Leave
  the ADRs as written.
- Note in the reference docs (`docs/ref/data.md`) that IDs are opaque strings,
  that new ones are 26-character Crockford base32, and that older trees keep
  their UUIDs, and the rule for IDs chosen by hand. `--id` is not documented
  elsewhere, so the rule lives there.
- Changelog entries (below).

## Incompatibilities for the changelog

- NetworkManager connection nodes carry their NetworkManager UUID in a `uuid`
  point. Existing connection nodes are not converted: delete them before
  upgrading, and the client adds the host's connections back as unmanaged nodes;
  mark the ones Simple IoT should manage again.
- New nodes, instances, users, and notifications get 26-character Crockford
  base32 IDs instead of UUIDs. Existing IDs are unchanged. Anything outside
  Simple IoT that expects IDs to be UUIDs needs to accept the new form.
- An instance started with an `--id` outside the allowed characters no longer
  starts, and a node with such an ID no longer accepts edge changes.

Draft entries:

```markdown
- **New IDs are shorter.** Nodes and instances created from now on get
  26-character IDs in Crockford base32 instead of 36-character UUIDs, which
  saves about 50 bytes per point on the sync link. Existing IDs are not changed.
  Tools outside Simple IoT that expect UUIDs need to accept the new form.
- **IDs chosen by hand are checked.** An `--id` or a node ID supplied through
  the API or NATS may use only letters, digits, and `-`, up to 64 characters,
  and may not start with `-`. An instance whose `--id` breaks this rule no
  longer starts, and a node with such an ID no longer accepts edge changes.
```
