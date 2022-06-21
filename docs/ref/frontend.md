# Frontend

## Elm Reference Implementation

The reference Simple IoT frontend is implemented in Elm as a Single Page
Application (SPA) and is located in the
[`frontend/`](https://github.com/simpleiot/simpleiot/tree/master/frontend)
directory.

### Code Structure

The frontend is based on [elm-spa](https://www.elm-spa.dev/), and is split into
the following directories:

- **Api**: contains core data structures and API code to communicate with
  backend (currently REST).
- **Pages**: the various pages of the application
- **Components**: each node type has a separate module that is used to render
  it. `NodeOptions.elm` contains a struct that is used to pass options into the
  component views.
- **UI**: Various UI pieces we used
- **Utils**: Code that does not fit anywhere else (time, etc)

We'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible.

### Creating Custom Icons

SIOT icons are 24x24px pixels (based on feather icon format). One way to create
them is to:

- create a 24x24px drawing in InkScape, scale=1.0
- draw your icon
- if you use text
  - convert text to path: select text, and then menu Path -> Object to Path
  - make sure fill is set for path
- save as plain SVG
- set up a new Icon in `frontend/src/UI/Icon.elm` and use an existing custom
  icon like `variable` as a template.
- copy the SVG path strings from the SVG file into the new Icon
- you'll likely need to adjust the scaling transform numbers to get the icon to
  the right size

(I've tried using: https://levelteams.com/svg-to-elm, but this has not been real
useful, so I usually end up just copying the path strings into an elm template
and hand edit the rest)

## SIOT JavaScript library using NATS over WebSockets

This is a JavaScript library avaiable in the
[`frontend/lib`](https://github.com/simpleiot/simpleiot/tree/master/frontend/lib)
directory that can be used to interface a frontend with the SIOT backend.

Usage:

```js
import { connect } from "./lib/nats";

async function connectAndGetNodes() {
  const conn = await connect();
  const [root] = await conn.getNode("root");
  const children = await conn.getNodeChildren(root.id, { recursive: "flat" });
  return [root].concat(children);
}
```

This library is also published on NPM (in the near future).

(see [#357](https://github.com/simpleiot/simpleiot/pull/357))

(Note, we are not currently using this yet in the SIOT frontend -- we still poll
the backend over REST and fetch the entire node tree, but we are building out
infrastructure so we don't have to do this.)

## Custom UIs

The current SIOT UI is more an engineering type view than something that might
be used by end users. For a custom/company product IoT portal where you want a
custom web UI optimized for your products, there are several options:

1. modify the existing SIOT frontend.
1. write a new frontend, mobile app, desktop app, etc. The SIOT backend and
   frontend are decoupled so that this is possible.
