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
