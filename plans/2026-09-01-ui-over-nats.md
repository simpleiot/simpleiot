# Plan: UI over NATS

**Branch:** `cbrake/master` **Branched from:** `118de673`

## Context

The web UI polls `GET /v1/nodes` every four seconds (`Home_.elm`,
`Time.every 4000 Tick`). That request runs `client.GetNodesForUser`, which walks
every node under every group the user belongs to and returns the whole tree as
JSON, whether or not any of it is on screen. The cost grows with the size of the
tree, not with what the user is looking at, and a change takes up to four
seconds to appear. Writes go back over HTTP with the JWT the user received at
sign-in.

The pieces for a better path already exist:

- The embedded NATS server has a WebSocket listener (`SIOT_NATS_WS_PORT`,
  default 9222), and the HTTP server already proxies WebSocket upgrades on `/`
  to it (`api/server.go`, `websocketproxy`). A browser can reach NATS on the
  same origin and port as the UI, so TLS terminated by Caddy or the HTTP server
  covers it and no extra port has to be exposed.
- The store answers `nodes.<parent>.<id>` requests and re-publishes every point
  at every ancestor as `up.<ancestor>.<node>.<type>.<key>` (and
  `up.<ancestor>.<node>.<parent>.<type>.<key>` for edge points). A subscription
  to `up.<group>.>` is exactly "everything that changes under this group".
- `frontend/lib/siot-nats.mjs` is a JavaScript client for NATS over WebSockets,
  though it predates the binary point encoding and needs to be brought up to
  date before it works against a current server.

What is missing is a credential a browser can present to NATS that grants only
what that user may see, and a wire protocol the store can scope to that user.
Today the WebSocket listener accepts the shared `SIOT_AUTH_TOKEN`, which grants
everything, and `auth.getNatsURI` hands that token to any connected client. The
[security cleanup plan](2026-08-24-security-cleanup.md) removes that handout
(item 5); this plan supplies the replacement. It is step 4 of the security
sequencing referenced in that plan, and it shares the in-process authorizer with
the [per-device credentials plan](2026-08-20-per-device-credentials.md).

## Goals

1. The UI opens one NATS connection over WebSocket and keeps it for the life of
   the session. No polling.
2. Nodes are fetched as the user expands the tree, one request per expansion.
3. Point changes for nodes on screen are pushed to the browser as they happen.
   Edits made in the UI are sent as points over the same connection.
4. The browser connection is authenticated as the signed-in user and can reach
   only the subtrees that user belongs to. Nothing the browser can do over NATS
   exceeds what it can do over HTTP today, and several things it can do today
   (read every user's points, publish as any origin) it can no longer do.

## Design Decisions

**Transport is nats.ws through the existing `/` WebSocket proxy.** The browser
connects to `ws(s)://<ui host>/`, the same origin the page came from. The proxy
forwards to the NATS WebSocket listener on localhost. The direct listener on
port 9222 stays for other clients. `SIOT_NATS_WS_PORT=0` disables both.

**The user's JWT is the NATS credential.** Sign-in stays on `POST /v1/auth` and
returns the JWT as it does now. The browser then connects with the user's node
ID as the NATS `user` and the JWT as the `pass` in `CONNECT`. The server
validates the JWT in-process with `server.Options.CustomClientAuthentication`,
the same authorizer the
[per-device credentials plan](2026-08-20-per-device-credentials.md) describes
(see "One authorizer serves devices and the browser" there for the shared pieces
and the `Check` order), confirms the JWT's subject matches `user`, and computes
the connection's permissions from the user. The JWT travels in `pass` rather
than `token` so that `Connz` reports the user ID as `AuthorizedUser`, which is
what revocation keys on. The browser also sets nats.ws's `inboxPrefix` to
`_INBOX_<U>` so its reply inboxes are its own; `_INBOX.>` is shared by every
client on the server and would expose every other client's replies. Two
consequences:

- The WebSocket listener never accepts an unauthenticated connection, and no
  browser ever holds the shared token. `auth.getNatsURI` is removed.
- A page on another origin can open a WebSocket to the server (browsers do not
  apply CORS to WebSockets) but cannot authenticate, because the JWT lives in
  this origin's `localStorage`. Credential-in-CONNECT is what makes that safe;
  cookie-based auth would not be.

**Scope is proven by the subject, not by a header.** NATS permissions match
subject patterns, and the existing subjects put the node ID where a pattern
cannot check membership (`nodes.<parent>.<id>`, `p.<id>.<type>.<key>`). The
browser therefore uses a user namespace whose first tokens carry the identity
the server has already verified:

| Purpose          | Subject                               | Direction | Store check                                          |
| ---------------- | ------------------------------------- | --------- | ---------------------------------------------------- |
| Fetch nodes      | `u.<G>.<U>.nodes.<parent>.<id>`       | request   | `<id>` (or `<parent>` when `id=all`) is G or under G |
| Send node points | `u.<G>.<U>.p.<id>.<type>.<key>`       | publish   | `<id>` under G; `Origin` set to U                    |
| Send edge points | `u.<G>.<U>.ep.<id>.<parent>`          | publish   | `<parent>` under G (new edge) or `<id>` under G      |
| Who am I         | `auth.me`                             | request   | JWT in the payload; reply is U and its anchors       |
| Live node points | `up.<G>.<node>.<type>.<key>`          | subscribe | none needed, the pattern is the scope                |
| Live edge points | `up.<G>.<node>.<parent>.<type>.<key>` | subscribe | none needed                                          |
| Request replies  | `_INBOX_<U>.>`                        | subscribe | the connection's own inbox prefix                    |

G is an _anchor_: a node the user node sits directly under (the parents of the
user's live edges, the same set `GetNodesForUser` treats as top level today). U
is the user's node ID. A user in two groups has two anchors and gets a
permission line for each. The permission set for a connection is:

- publish allow: `u.<G>.<U>.>` for each anchor, `auth.me`
- subscribe allow: `up.<G>.>` for each anchor, `_INBOX_<U>.>`

Nothing else: no `p.>`, `nodes.>`, `ep.>`, `$JS.>`, `auth.user`, or `admin.>`.
The NATS server guarantees the connection may only speak for `(G, U)`; the store
guarantees the target node is under G. Neither side needs the other's data
structures, and no header can be omitted to get a different outcome.

**The store rejects out-of-scope requests and stamps the origin.** One new
subscription, `u.*.*.>`, dispatches to the existing handlers after stripping the
prefix, checking the subtree with the edge cache, and overwriting `Point.Origin`
with U on every point. The existing handlers do not change.

**Nodes are fetched by depth.** `nodes` requests gain an optional `depth` point
(default 0, today's behavior). Expanding a node requests its children with
`depth=1`, so the children arrive along with their own children and the UI knows
which of them get an expand arrow. Sign-in fetches each anchor the same way.
Collapsing does not refetch; expanding again does, which doubles as a refresh.

**Live updates follow what is rendered.** Elm derives a subject list from the
tree and sends it to JavaScript whenever it changes; JavaScript reconciles its
NATS subscriptions against that list. For each rendered node N under anchor G
the list contains `up.G.N.*.*` (its points), and for each expanded node P it
contains `up.G.*.P.*.*` (edge points of P's children, which is how new, deleted,
and re-roled children are noticed). A new child edge triggers a fetch of that
child. A tab showing 200 nodes holds a few hundred subscriptions, which NATS
handles without difficulty, and only matching messages cross the socket.
JavaScript batches incoming points per animation frame before handing them to
Elm so a burst from a device is one render, not fifty.

**Node replies use the binary point encoding, not protobuf.** This landed ahead
of the plan: `data.EncodeNodes` / `data.DecodeNodes` frame a reply as a version
byte, an error string, and nodes made of id, type, parent, points, and edge
points (hash omitted), and `node.proto` and `point.proto` are gone. The
JavaScript codec in Phase 2 mirrors that frame.

**JavaScript owns the connection; Elm owns the tree.** Elm 0.19 has no WebSocket
support, so a single port pair carries a small JSON protocol. Commands from Elm:
`connect`, `fetch {anchor, parent, id, depth}`, `watch [subjects]`,
`sendPoints {anchor, id, points}`,
`sendEdgePoints {anchor, id, parent, points}`, `disconnect`. Events to Elm:
`connected {userId, anchors}`, `disconnected {reason}`, `authFailed`,
`nodes {anchor, parent, nodes}`, `points {nodeId, points}`,
`edgePoints {nodeId, parentId, points}`, `ack {reqId, error}`, `error`. Points
cross the port in the same JSON shape the HTTP API used, so `Api.Point.decode`
and every component are untouched.

**Node operations stay on HTTP for now.** Add, delete, move, mirror, duplicate,
and notify are multi-step operations in `client/node.go` behind JWT-protected
HTTP routes. Moving them to NATS is a separate step once `userCanAccess` exists
for the HTTP API too (security cleanup, deliberately excluded item). This plan
changes reads and point writes only.

**Membership and expiry are enforced by disconnecting.** Permissions are
computed at connect time. The authorizer's single `up.root.>` subscription
(shared with the credentials plan's credential index) watches user nodes and
disconnects a user's connections when a user edge changes (anchor added or
removed, user tombstoned) or the password changes. It also schedules a
disconnect at the JWT's `exp`. The browser reconnects, calls `auth.me`, and
rebuilds from the new anchors; an expired or refused JWT sends the user to
sign-in. Finding the connections is the same path the credentials plan uses for
devices: `Connz(User: true)` filtered on `AuthorizedUser` (the user ID), then
`DisconnectClientByID`.

**The bundle is built with esbuild.** The frontend has no bundler; `main.js` is
a plain script. Add `esbuild` as a dev dependency, keep the client in
`frontend/lib` (it is the published `simpleiot-js` package and the right home
for the codec), and have `siot_build_frontend` bundle `frontend/src-js/index.js`
to `public/dist/siot.js.gz`. That file follows the same rule as `elm.js.gz`:
regenerated constantly, committed only at release.

## Security

What a browser connection can and cannot do after this plan, compared with
today.

| Concern                                 | Today                                                      | After                                                        |
| --------------------------------------- | ---------------------------------------------------------- | ------------------------------------------------------------ |
| Credential on the WebSocket             | Shared token, full access, handed out by `auth.getNatsURI` | User JWT; `auth.getNatsURI` removed                          |
| Anonymous WebSocket connection          | Accepted when no token is configured                       | Refused when a token is configured; open mode stays open     |
| Read scope                              | Any node in the instance                                   | `up.<G>.>` and `nodes` under the user's anchors only         |
| Write scope                             | Any subject, any origin                                    | Points under the user's anchors; origin forced to the user   |
| JetStream, admin, auth subjects         | Reachable                                                  | Not in the permission set                                    |
| Cross-origin page opening the WebSocket | Could authenticate with the shared token if it had it      | Cannot authenticate without the JWT, which is origin-bound   |
| User removed from a group or deleted    | Access until the JWT expires (up to 7 days)                | Disconnected within seconds; reconnect recomputes or refuses |
| JWT expiry                              | Not enforced on a live NATS connection                     | Scheduled disconnect at `exp`                                |
| HTTP node API                           | Unscoped per node                                          | Unchanged (tracked in the security cleanup plan)             |

Hardening that is cheap and included: `Websocket.AllowedOrigins` set to the UI's
origin when `--httpOrigin` is configured (optional, off by default); `MaxSubs`
left at the NATS default but noted in the configuration docs; failed JWT
authentications logged with the remote address, feeding the rate limiter from
security cleanup item 7 when it lands.

What this plan does not fix: the HTTP node routes remain unscoped; the direct
WebSocket port 9222 still listens on all interfaces (security cleanup item 6
pattern applies if wanted); JWT lifetime and renewal are unchanged (security
cleanup item 8 shortens it to 24 hours, and a long-lived tab will then be signed
out once a day until renewal exists).

## Changes to note in the changelog

- **`simpleiot-js` gets a new major version** for the binary node and point
  codec (the server-side encoding change is already released).
- **`auth.getNatsURI` is removed.** Nothing in the repository depends on it
  after this plan.
- **`GET /v1/nodes` is removed.** The UI was its only caller. The other node
  routes stay.
- **New subjects**: `u.*.*.>`, `auth.me`. New `nodes` request parameter:
  `depth`.
- The shared token continues to work for devices, the CLI, and MQTT clients,
  exactly as the credentials plan describes; the server's own client connects
  in-process and needs no token. A deployment that sets no token still runs open
  on every listener, WebSocket included, so browser scoping only limits anything
  once a token is configured. The UI always presents its JWT either way.
- No stored data changes.

## Phases

Commit after each phase. Each phase updates `CHANGELOG.md` and the docs it
affects.

### Phase 0: Spike

Prove the four things the design rests on before building on them. The Go parts
stay as regression tests.

1. A nats.go client connecting with `ws://` through the HTTP server's `/` proxy,
   presenting the user ID and JWT as user and password, is accepted by a
   `CustomClientAuthentication` check and refused when the JWT is invalid,
   expired, or issued to a different user. `Connz(User: true)` reports the user
   ID for the connection. (`server/ws_auth_spike_test.go`)
2. With subscribe permission `up.G.>` only, that client receives points for
   nodes under G and a permissions violation, not silence, for a subscription to
   `up.H.>`.
3. A hand-written page in `frontend/lib/` connects with nats.ws through the
   proxy, decodes a binary points message, and prints it. Confirm the
   `AllowedOrigins` behavior through the proxy (does the origin header survive?)
   and note the answer in the plan.
4. Bundle size with esbuild: nats.ws plus the codec, minified and gzipped. Under
   60 KB is the expectation.

Decide here whether the credentials plan's Phase 1 has landed. If not, this plan
builds the shared pieces listed under "One authorizer serves devices and the
browser" in that plan (`server/auth.go` with the in-process, shared token, open
mode, and JWT branches; the three token options dropped; the in-process server
client; the token-only start; the `up.root.>` subscription; the `Connz`-based
disconnect) and the credentials plan adds the NKey and enrollment branches
later.

### Phase 1: Server side

- `server/auth.go`: `authorizer` implementing `server.Authentication`, with the
  `Check` branch order from the credentials plan. This plan adds the JWT branch:
  `user` is a user node ID, `pass` is a JWT valid for that user →
  `userPermissions(userID)`. Starts in token-only mode; `SetStore` is called
  once the store is up and supplies the JWT key and an
  `anchors(userID) []string` function. Connections that arrive in between are
  refused and retry.
- `userPermissions`: publish `u.<G>.<U>.>` per anchor plus `auth.me`; subscribe
  `up.<G>.>` per anchor plus `_INBOX_<U>.>`.
- Revocation: from the authorizer's `up.root.>` subscription, disconnect a
  user's connections on edge change, tombstone, or `pass` point, found through
  `Connz(User: true)` by `AuthorizedUser`; schedule disconnect at JWT `exp`.
- `server/nats-server.go`: set `CustomClientAuthentication`; drop
  `Authorization`, `Websocket.Token`, and `MQTT.Token` (the authorizer owns all
  three). Optional `AllowedOrigins`. `server/server.go`: the server's own client
  connects with `nats.InProcessServer` when the NATS server is embedded.
- `store/store.go`: subscribe `u.*.*.>`; `handleUserRequest` parses
  `(G, U, op, rest)`, checks scope with `db.isUnder(id, G)` (edge cache walk
  with a visited set; memoized per request), rewrites the subject to the plain
  form, stamps origin, and calls the existing handler. Add `handleAuthMe`. Add
  the `depth` parameter to `handleNodesRequest` and `db.getNodes`.
- Remove `auth.getNatsURI` (security cleanup item 5).
- `docs/ref/api.md`: the `u.*` namespace, `auth.me`, `depth`, the reply
  encoding, and the removals.

Tests (`server/auth_test.go`, `store/store_test.go`):

- Table test for `userPermissions` against the subject table.
- Scope: a JWT for a user under G can fetch, subscribe, and write under G; the
  same operations against a node under H fail with a permissions violation
  (subscribe) or an error reply (request), and no point lands.
- Origin: a point published with `origin` set to another ID is stored with
  origin U.
- Depth: `nodes.G.all` with `depth=1` returns children and grandchildren and
  nothing deeper.
- Revocation: remove the user's edge mid-session; the connection closes within a
  bounded time; reconnect is refused. Re-add; reconnect succeeds with the new
  anchors. A JWT with `exp` ten seconds out is disconnected at ten seconds.
- With a token configured, anonymous connections to the WebSocket listener are
  refused, while the in-process server client, a token-only device, and an MQTT
  client presenting the token still connect. With no token configured, an
  anonymous connection gets full permissions on every listener.
- A browser connection subscribed to `_INBOX.>` gets a permissions violation and
  does not see another client's replies.

### Phase 2: JavaScript client

- `frontend/lib/codec.mjs`: `encodePoints`, `decodePoints`, `decodeNodes`
  mirroring `data/point.go` and `data.EncodeNodes` (little-endian, uint16 length
  prefixes, int64 nanosecond time, one-byte data type, int values in 1/2/4/8
  bytes, float64 values, a version byte on the node frame). Points are returned
  in the shape the UI already decodes:
  `{type, key, time, dataType, value, text, tombstone, origin}`.
- `frontend/lib/siot-nats.mjs`: rewrite on the codec.
  `connect({url, user, pass})`, which sets `inboxPrefix` to `_INBOX_<user>`;
  `me()`; `getNodes(anchor, parent, id, {depth, type, includeDel})`;
  `sendNodePoints(anchor, id, points)`;
  `sendEdgePoints(anchor, id, parent, points)`; `subscribe(subject)` yielding
  decoded points with the subject tokens split out. Drop `google-protobuf`. Bump
  the package major version.
- `frontend/src-js/index.js`: Elm bootstrap (moved from `public/main.js`), the
  port protocol above, subscription reconciliation, per-frame batching, and
  reconnect handling. On nats.ws `reconnect`, emit `connected` again so Elm
  refetches every loaded subtree, since messages during the gap were missed.
- Build: `esbuild` dev dependency; `siot_build_frontend` bundles to
  `public/dist/siot.js` and gzips it; `index.html` loads it; `.gitignore` and
  the CLAUDE.md commit rule cover `siot.js.gz` alongside `elm.js.gz`.
- Tests: `frontend/lib/test.mjs` codec round trips against fixtures a Go test
  writes to `frontend/lib/testdata/` (`data/point_fixture_test.go`), so the two
  encoders are checked against the same bytes.

### Phase 3: Elm

- `Api/Nats.elm` (port module): typed encoders for commands and a decoder for
  events; `send : Command -> Cmd msg`,
  `receive : (Result Error Event -> msg) -> Sub msg`.
- `Home_.elm`:
  - `init` sends `connect`. On `connected` fetch each anchor with `depth=1`.
  - `NodeView` gains `anchor` and `loading`. `ToggleExpChildren` on a collapsed
    node fetches `depth=1` and sets `loading`; the arrow shows a spinner until
    the reply merges.
  - `nodes` events merge a subtree into the existing tree by `(id, parent)`,
    preserving expansion state (extend `mergeNodeTree`).
  - `points` events apply `Point.updatePoints` to every tree node with that ID
    (the optimistic path already does this). `edgePoints` update tombstone and
    role, and fetch a child that is not yet in the tree.
  - After any tree change, derive the subject list (pure function,
    `watchSubjects : List (Tree NodeView) -> List String`) and send `watch` only
    when it differs from the last one sent.
  - `ApiPostPoints` and `UploadContents` send over NATS; the optimistic update
    stays. An `ack` with an error pops the existing error banner.
  - `Tick` keeps `now` fresh at one second and no longer fetches.
  - `disconnected` shows a reconnecting badge in the header; `authFailed` signs
    out, replacing the 401 path.
- `Api/Node.elm`: remove `list` and `postPoints`; the HTTP node operations stay.
- Tests (`frontend/tests`): subtree merge preserves expansion and replaces stale
  children; `watchSubjects` for a small tree with one expanded node matches the
  expected list; a tombstone edge point hides a node.

Docs: `docs/user/ui.md` (live updates, spinner, connection badge),
`docs/user/status.md` (UI section), `docs/ref/frontend.md` (architecture, port
protocol, build, `simpleiot-js`), `docs/ref/high-availability.md` (frontend
row).

### Phase 4: Hardening and wrap-up

- Reconnect storm: nats.ws reconnect uses jittered backoff by default; confirm
  the settings and cap `maxReconnectAttempts` at unlimited with a visible badge
  rather than giving up.
- Large child lists: an anchor with thousands of direct children still arrives
  in one reply. Measure with a generated tree (the metrics client has a
  generator pattern to copy); if a single reply is too large for the NATS
  payload limit, page by `depth=0` first and fill children on demand. Decide
  from the measurement rather than in advance.
- Hidden tab: on `visibilitychange` to hidden, send an empty `watch`; on
  visible, restore and refetch loaded subtrees. Optional; include if it is a few
  lines.
- Remove `GET /v1/nodes` and `client.GetNodesForUser` if nothing else uses them
  after Phase 3.
- `docs/ref/security.md`: a "Browser" section with the table above.
  `docs/user/configuration.md`: the proxy path, `SIOT_NATS_WS_PORT=0`, and
  `AllowedOrigins`. `CLAUDE.md`: build steps and the `siot.js.gz` commit rule.
  `plans/plans.md` status. Changelog review.

### Follow-ups (not in this plan)

- Move node operations (add, delete, move, mirror, duplicate, notify) to
  `u.<G>.<U>.op.*` and scope the HTTP routes with the same `isUnder` check.
- JWT renewal over the live connection so a tab is not signed out at expiry.
- Per-device credentials plan Phase 1 adds the NKey branch to the same
  authorizer.
- Notifications display in the UI (deferred by the notifications plan) becomes
  straightforward once the browser has a live subscription.

## Testing Strategy

- Server behavior is tested with `server.TestServer`, a nats.go client using a
  `ws://` URL through the HTTP proxy, and JWTs minted with the store's key, so
  the tests exercise the real authorizer, proxy, and store dispatch. A helper
  returns a signed-in user (node, anchors, JWT) to keep each test short.
- Negative tests assert both on the NATS error and on the absence of the point
  in the store, so a permission that fails open is caught.
- Codec fixtures are generated by Go and consumed by the JavaScript tests, so
  the two implementations cannot drift silently.
- Elm tests cover the pure tree and subject functions. The end-to-end path is
  checked by hand with `siot_watch`: expand, edit, watch a signal generator
  update without a refresh, remove the user from a group in a second tab and
  watch the first one sign out.
- `go test -race ./...`, `golangci-lint run`, `siot_test` before the wrap-up
  commit.

## Open Questions

- **nats.ws versus the newer `@nats-io/transport-websocket`.** nats.ws 1.x is
  what `frontend/lib` already depends on and is stable; the v3 packages are
  smaller and where development continues. Decide in Phase 0 by bundle size and
  API stability; the port protocol does not depend on the choice.
- **Where the anchor for a mirrored node goes.** A node visible under two
  anchors appears in two trees and is subscribed twice, once per anchor. That is
  correct and simple; if the duplicate subscriptions ever matter, JavaScript can
  dedupe by node ID.
- **`hasChildren` without depth.** Fetching `depth=1` on every expansion is
  simple and doubles as a refresh, at the cost of fetching grandchildren that
  may never be shown. A `children` count on the edge would avoid that but
  changes `NodeEdge`. Start with depth; revisit only if the measurement in Phase
  4 shows a problem.

## Key Files

- `server/auth.go` (new), `server/auth_test.go` (new),
  `server/ws_auth_spike_test.go` (new), `server/nats-server.go`,
  `server/server.go`.
- `store/store.go` (user namespace dispatch, `auth.me`, `depth`),
  `store/jetstream.go` (`isUnder`, `getNodes` depth), `store/store_test.go`.
- `data/point_fixture_test.go` (new).
- `client/node.go` (`GetNodes` decoder, remove `GetNodesForUser`),
  `api/nodes.go` (remove `GET /v1/nodes`).
- `frontend/lib/codec.mjs` (new), `frontend/lib/siot-nats.mjs`,
  `frontend/lib/test.mjs`, `frontend/lib/package.json`.
- `frontend/src-js/index.js` (new, replaces `public/main.js`),
  `frontend/package.json` (esbuild), `frontend/public/index.html`, `envsetup.sh`
  (`siot_build_frontend`), `.gitignore`.
- `frontend/src/Api/Nats.elm` (new), `frontend/src/Api/Node.elm`,
  `frontend/src/Pages/Home_.elm`, `frontend/tests/`.
- `docs/ref/api.md`, `docs/ref/security.md`, `docs/ref/frontend.md`,
  `docs/user/ui.md`, `docs/user/status.md`, `docs/user/configuration.md`,
  `docs/ref/high-availability.md`, `CHANGELOG.md`, `CLAUDE.md`,
  `plans/plans.md`.
