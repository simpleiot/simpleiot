# Frontend

Two web UIs are served during the move from Elm to Svelte: the Svelte UI at `/`
and the Elm UI at `/old`. They share one sign-in. The Elm UI will be removed in
a later release. Both use the connection described below; this page documents
the Svelte UI.

## How the browser talks to Simple IoT

The browser connects to the embedded NATS server, through a WebSocket the HTTP
port relays, and uses it for reading the tree, writing points, and live updates.
HTTP carries only sign-in and the node operations that change the tree's shape.

### Signing in and connecting

1. The page signs in with `POST /v1/auth` and receives a JWT, which it keeps in
   local storage.
2. It opens a NATS WebSocket on the HTTP port, which the server proxies to the
   NATS WebSocket listener on loopback. The user's node ID and the JWT are the
   NATS user and password.
3. It asks `auth.me` who it is. The reply names the user and the user's
   _anchors_: the groups the user sits directly under. The connection may reach
   the subtree under each anchor and nothing else.

A connection that fails is retried every five seconds. A refused credential
returns the page to sign-in.

### Reading and writing

Every read and write is a request on a subject that names the anchor and the
user:

- `u.<anchor>.<user>.nodes.<parent>.<id>` fetches a node, or a subtree to a
  given depth. On connect the page fetches each anchor with its children and
  grandchildren; expanding a node fetches its children with their children, so
  each child knows whether it can be expanded.
- `u.<anchor>.<user>.p.<id>.<type>.<key>` writes points to a node, and
  `u.<anchor>.<user>.ep.<id>.<parent>` writes points to an edge.

The store answers these requests. The full subject formats are in the
[API reference](api.md#nats).

### Live updates

The page subscribes to the nodes on screen: `up.<anchor>.<id>.*.*` for a node's
points and `up.<anchor>.*.<id>.*.*` for the edges of its children. The
subscriptions follow the tree as it is expanded and collapsed. Incoming points
are batched per animation frame, keeping the newest value per point, so a burst
from a device is one update. An edge for a child the page does not have is a new
node, and the page fetches it.

Live updates are best effort: a message sent while the connection is down, or
while the tab is hidden (a hidden tab drops its subscriptions), is not
delivered. The store holds the current state, so after a reconnect, or when the
tab is shown again, the page asks `auth.me` again (the user's groups may have
changed) and fetches everything it shows.

### Operations over HTTP

Adding, deleting, moving, mirroring, and duplicating a node, sending a
notification, and generating a key go over the [HTTP API](api.md#http), with the
JWT as the `Authorization` header. The routes apply the same scope as the NATS
connection.

### Design

- **No protocol gateway.** The HTTP port relays the WebSocket to the NATS
  listener, and the browser speaks NATS to the server end to end. The server
  enforces permissions on each connection, so there is no broad service
  credential with authorization in code, and no upstream connection opened on
  the browser's behalf. The jobs a gateway would do are split: the proxy carries
  the transport, the authorizer establishes identity, and the store validates
  requests.
- **The credential proves identity, not scope.** The JWT names the user. The
  server's authorizer computes the user's permissions from the tree when the
  connection is made, and closes the connection when the user's groups change,
  the password changes, or the JWT expires. This is the role NATS
  [auth callout](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_callout)
  plays in a NATS deployment, done in process; see
  [How the authorizer works](security.md#how-the-authorizer-works).
- **Requests, not raw subjects.** The browser may publish only `u.*` requests
  and `auth.me`, and subscribe only to `up.*` and its own reply inbox. It never
  reaches JetStream or the subjects the server's clients use. The store answers
  each request and checks its target against the anchor, and sets the origin of
  every point to the user, so the store is the service boundary every write
  passes through.
- **The store is the source of truth.** Live updates keep the page current, and
  a fetch after any gap makes it correct.

[Security](security.md#browser) lists exactly what the connection is granted and
what the store checks.

## Svelte UI

The Svelte UI is a [SvelteKit](https://svelte.dev/docs/kit) single-page app in
[`ui/`](https://github.com/simpleiot/simpleiot/tree/master/ui), built with
`adapter-static` and embedded in the `siot` binary (`ui/embed.go`). The server
answers any path it has no file for with `index.html`, so a link to a node loads
the app. See `ui/README.md` for the build, test, and dev server commands.

### Code Structure

- `src/routes`: the pages. `/` and `/nodes/[id]` both render the tree; the
  second opens the tree down to one node.
- `src/lib/siot`: the client layer. `connection.ts` owns the NATS connection
  (built on `frontend/lib`), `tree.ts` holds the pure tree code, and
  `tree.svelte.ts` is the store the page reads: the tree, pending edits, and the
  node operation in progress. `http.ts` has the operations that go over HTTP,
  and `schema.ts` the point and node types, which a Go test checks against
  `data/schema.go`.
- `src/lib/nodes`: one directory per node type, with its editor and a descriptor
  (`index.ts`) that says where the type may be added, whether it takes children,
  which parent owns it, and how it sorts. The registry collects the directories,
  so the tree page names no node types; adding a type is adding a directory.
- `src/lib/inputs`: the inputs editors are built from. Each reads its point from
  the node and reports an edit; none talks to the server.
- `src/lib/components`: the tree, its rows, the operation panel, and the shared
  editor frame.

Editors take props and report through callbacks, and only the page and the store
talk to the server, so an editor can be reused and tested on its own. Edits stay
pending until saved, and a save shows at once, before the server confirms it.

### Checks

TypeScript runs in `strict` mode, and `svelte-check` fails the build on any
error or warning. `npm test` runs the vitest unit tests for the tree, point,
connection, and time code; `npm run lint` runs prettier and eslint.

### File upload

The [File node](../user/file.md) editor uploads a file from the browser into the
node's `data` point, with its name in the `name` point, as an ordinary point
write. With the binary option set, the contents are base64 encoded first. A
point travels in one NATS message, so the server's maximum payload (1 MB by
default) limits the file size.

## SIOT JavaScript library using NATS over WebSockets

[`frontend/lib`](https://github.com/simpleiot/simpleiot/tree/master/frontend/lib)
is `simpleiot-js`, the client the web UI uses to talk to the backend over NATS
WebSockets, and it can be used by any other JavaScript frontend. It connects as
a signed-in user with the JWT from `POST /v1/auth`, learns which groups the user
belongs to, fetches nodes one subtree at a time, subscribes to live points, and
writes points. The `README.md` in that directory documents the API; the subjects
are in the [API reference](api.md#nats).

The library has one dependency, `nats.ws`, and no build step. The Svelte UI
bundles both with Vite.

`codec.js` mirrors the binary point and node encoding in `data/point.go` and
`data/node.go`. The Go test `data/point_fixture_test.go` writes fixtures into
`frontend/lib/testdata`, and `npm test` in `frontend/lib` decodes and re-encodes
them, so the two encoders are checked against the same bytes. Run the Go test
with `UPDATE_FIXTURES=1` after changing the encoding.

## Custom UIs

The current SIOT UI is more an engineering type view than something that might
be used by end users. For a custom/company product IoT portal where you want a
custom web UI optimized for your products, there are several options:

1. Modify the existing SIOT frontend.
1. Write a new frontend, mobile app, desktop app, etc. The SIOT backend and
   frontend are decoupled so that this is possible.

### Passing a custom UI to SIOT

There are ways to use a custom UI with SIOT at the app and package level:

1. **Application:** pass a directory containing your public web assets to the
   app using: `siot serve -customUIDir <your web assets>`
1. **Package:** populate `CustomUIFS` with a
   [`fs.FS`](https://pkg.go.dev/io/fs#FS) in the SIOT
   [server options`](https://pkg.go.dev/github.com/simpleiot/simpleiot/server#Options).

In both cases, the filesystem should contain a `index.html` in the root
directory. If it does not, you can use the
[`fs.Sub`](https://pkg.go.dev/io/fs#Sub) function to return a subtree of a
`fs.FS`.
