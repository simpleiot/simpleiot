# Frontend

## Elm Reference Implementation

The reference Simple IoT frontend is implemented in Elm as a Single Page
Application (SPA) and is located in the
[`frontend/`](https://github.com/simpleiot/simpleiot/tree/master/frontend)
directory.

### Code Structure

The frontend is based on [elm-spa](https://www.elm-spa.dev/), and is split into
the following directories:

- `Api`: contains core data structures and API code to communicate with the
  backend. `Api.Nats` is the port module the node tree is read and written
  through; the HTTP calls in `Api.Node` and `Api.Auth` remain for sign-in and
  node operations.
- `Pages`: the various pages of the application
- `Components`: each node type has a separate module that is used to render it.
  `NodeOptions.elm` contains a struct that is used to pass options into the
  component views.
- `UI`: Various UI pieces we used
- `Utils`: Code that does not fit anywhere else (time, etc.)

We'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible.

### The node tree over NATS

Elm 0.19 has no WebSocket support, so `public/main.js` owns the NATS connection
and `Api.Nats` carries a small JSON protocol over one port pair. JavaScript owns
the connection; Elm owns the tree.

Commands from Elm: `connect {token}`, `fetch {anchor, parent, id, depth}`,
`watch [subjects]`, `sendPoints {anchor, id, points}`, and `disconnect`. Events
to Elm: `connected {userId, anchors}`, `disconnected`, `authFailed`,
`nodes {anchor, parent, id, depth, nodes}`, `points [{nodeId, points}]`,
`edgePoints [{nodeId, parentId, points}]`, and `error {message}`. Points cross
the port in the JSON shape the HTTP API used, so `Api.Point.decode` and every
component are unchanged.

An _anchor_ is a group the user belongs to; the connection may reach the subtree
under each of its anchors and nothing else. On `connected`, the page fetches
each anchor with its children and grandchildren. Expanding a node fetches its
children with their children, so each child knows whether it can be expanded,
and collapsing keeps what is there. `Utils.NodeTree` holds the pure tree code:
merging a fetched subtree into the tree while keeping expansion state and the
deeper levels a reply did not reach, applying live points, and deriving the
watch list. The watch list is every node on screen (`up.<anchor>.<id>.*.*` for
its points and `up.<anchor>.*.<id>.*.*` for the edges of its children), sent to
JavaScript whenever it changes; JavaScript brings its subscriptions in line and
batches incoming points per animation frame. An edge for a child the tree does
not have is a new node: the page fetches it. A reconnect, or a tab coming back
into view, refetches everything loaded, since messages in between were missed.

### Creating Custom Icons

SIOT icons are `24x24px` pixels (based on feather icon format). One way to
create them is to:

- Create a `24x24px` drawing in Inkscape, scale=1.0
- draw your icon
- if you use text
  - Convert text to path: select text, and then menu Path -> Object to Path
  - Make sure fill is set for path
- save as plain SVG
- set up a new Icon in `frontend/src/UI/Icon.elm` and use an existing custom
  icon like `variable` as a template.
- Copy the SVG path strings from the SVG file into the new Icon
- You'll likely need to adjust the scaling transform numbers to get the icon to
  the right size

(I've tried using: https://levelteams.com/svg-to-elm, but this has not been real
useful, so I usually end up just copying the path strings into an elm template
and hand edit the rest)

### File upload

The [File node UI](../user/file.md) has the capability to upload files in the
browser and then store them in a node point. The default max payload of NATS is
1MB, so that is currently the file size limit, but NATS
[can be configured](https://docs.nats.io/reference/faq#is-there-a-message-size-limitation-in-nats)
for a payload size up to 64MB. 8MB is recommended.

Currently the payload is stored in the Point `String` field for simplicity. If
the binary option is selected, the data is base64 encoded. Long term it may make
sense to support JetStream Object store, local file store, etc.

The [elm/file](https://package.elm-lang.org/packages/elm/file/latest/) package
is used upload a file into the browser. Once the data is in the browser, it is
sent to the backup as a standard point payload. Because we are currently using a
JSON API, binary data is base64 encoded.

The process by which a file is uploaded is:

- The `NodeOptions` struct, which is passed to all nodes has an `onUploadFile`
  field, which is used to triggers the `UploadFile` message which runs a browser
  file select. The result of this select is a `UploadSelected` message.
- This message calls `UploadFile node.node.id` in `Home_.elm`.
- `File.Select.file` is called to select the file, which triggers the
  `UploadContents` message.
- `UploadContents` is called with the node id, file name, and file contents,
  which then sends the data via points to the backend.

## SIOT JavaScript library using NATS over WebSockets

[`frontend/lib`](https://github.com/simpleiot/simpleiot/tree/master/frontend/lib)
is `simpleiot-js`, the client the web UI uses to talk to the backend over NATS
WebSockets, and it can be used by any other JavaScript frontend. It connects as
a signed-in user with the JWT from `POST /v1/auth`, learns which groups the user
belongs to, fetches nodes one subtree at a time, subscribes to live points, and
writes points. The `README.md` in that directory documents the API; the subjects
are in the [API reference](api.md#nats).

The library has one dependency, `nats.ws`, and no build step. The web UI loads
it as native ES modules: `siot_build_frontend_js` copies `siot-nats.js`,
`codec.js`, and `nats.ws`'s bundle into `frontend/public/dist`, and an import
map in `index.html` resolves `nats.ws` to that copy. `siot_build_frontend` gzips
them alongside `elm.js`, and the server decompresses on request.

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
