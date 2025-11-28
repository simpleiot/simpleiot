# Frontend

## Elm Reference Implementation

The reference Simple IoT frontend is implemented in Elm as a Single Page
Application (SPA) and is located in the
[`frontend/`](https://github.com/simpleiot/simpleiot/tree/master/frontend)
directory.

### Code Structure

The frontend is based on [elm-spa](https://www.elm-spa.dev/), and is split into
the following directories:

- `Api`: contains core data structures and API code to communicate with backend
  (currently REST).
- `Pages`: the various pages of the application
- `Components`: each node type has a separate module that is used to render it.
  `NodeOptions.elm` contains a struct that is used to pass options into the
  component views.
- `UI`: Various UI pieces we used
- `Utils`: Code that does not fit anywhere else (time, etc.)

We'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible.

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

This is a JavaScript library available in the
[`frontend/lib`](https://github.com/simpleiot/simpleiot/tree/master/frontend/lib)
directory that can be used to interface a frontend with the SIOT backend.

Usage:

```js
import { connect } from "./lib/nats"

async function connectAndGetNodes() {
	const conn = await connect()
	const [root] = await conn.getNode("root")
	const children = await conn.getNodeChildren(root.id, { recursive: "flat" })
	return [root].concat(children)
}
```

This library is also published on NPM (in the near future).

(see [#357](https://github.com/simpleiot/simpleiot/pull/357))

(Note, we are not currently using this yet in the SIOT frontend we still poll
the backend over REST and fetch the entire node tree, but we are building out
infrastructure so we don't have to do this.)

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
