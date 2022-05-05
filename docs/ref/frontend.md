# Frontend

## Reference Implementation

The reference Simple IoT frontend is implemented in Elm and located in the
[`frontend/`](https://github.com/simpleiot/simpleiot/tree/master/frontend)
directory.

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

## Creating Custom Icons

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
