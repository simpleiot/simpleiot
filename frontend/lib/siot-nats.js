import { connect as natsConnect, StringCodec } from "nats.ws";
import { Timestamp } from "google-protobuf/google/protobuf/timestamp_pb";
import { Points, Point } from "./protobuf/point_pb";
import { NodesRequest } from "./protobuf/node_pb";
import { Message } from "./protobuf/message_pb";
import { Notification } from "./protobuf/notification_pb";

const sc = new StringCodec();

// connect opens and returns a connection to SIOT / NATS via WebSockets
export async function connect(opts = {}) {
  const { servers = ["ws://localhost:4223"] } = opts;
  const nc = await natsConnect({ ...opts, servers });

  // Force SIOTConnection to inherit from `nc` prototype
  SIOTConnection.prototype = Object.create(
    Object.getPrototypeOf(nc),
    Object.getOwnPropertyDescriptors(SIOTConnection.prototype)
  );
  // Create new instance of SIOTConnection and then assign `nc` properties
  return Object.assign(new SIOTConnection(), nc);
}

// SIOTConnection is a wrapper around a NatsConnectionImpl
function SIOTConnection() {
  // do nothing
}

Object.assign(SIOTConnection.prototype, {
  // getNode sends a request to `node.<id>` to retrieve an array of NodeEdges for
  // the specified Node id.
  // - If `id` is "root", the root node is returned
  // - If `parent` is falsy or "none", the edge details are not included
  // - If `parent` is "all", all instances of the node are returned
  // - `opts` are options passed to the NATS request
  async getNode(id, { parent, opts } = {}) {
    const payload = sc.encode(parent || "none");
    const m = await this.request("node." + id, payload, opts);
    return decodeNodesRequest(m.data);
  },

  // getNodeChildren sends a request to `node.<parentID>.children` to retrieve
  // an array of child NodeEdges of the specified parent node.
  // - `type` - can be used to filter nodes of a specified type (defaults to "")
  // - `includeDel` - set to true to include deleted nodes (defaults to false)
  // - `recursive` - set to true to recursively retrieve all descendants matching
  //   the criteria. In this case, each returned NodeEdge will contain a
  //   `children` property, which is an array of that Node's descendant NodeEdges.
  //   Set to "flat" to return a single flattened array of NodeEdges.
  // - `opts` are options passed to the NATS request
  async getNodeChildren(
    parentID,
    { type, includeDel, recursive, opts, _cache = {} } = {}
  ) {
    const includeDelNum = includeDel ? 1 : 0;

    const points = [
      { type: "nodeType", text: type },
      { type: "tombstone", value: includeDelNum },
    ];

    const payload = encodePoints(points);
    const m = await this.request(
      "node." + parentID + ".children",
      payload,
      opts
    );
    const nodeEdges = decodeNodesRequest(m.data);
    if (recursive) {
      const flat = recursive === "flat";
      const nodeChildren = await Promise.all(
        nodeEdges.map(async (n) => {
          const children =
            _cache[n.id] ||
            (await this.getNodeChildren(n.id, {
              type,
              includeDel,
              recursive,
              opts,
              _cache,
            }));
          // update cache
          _cache[n.id] = children;
          if (!flat) {
            // If not flattening, add `children` key to `n`
            n.children = children;
          }
          return children;
        })
      );
      if (flat) {
        // If flattening, simply return flat array of node edges
        return nodeEdges.concat(nodeChildren.flat());
      }
    }
    return nodeEdges;
  },

  // getNodesForUser returns the parent nodes for the given userID along
  // with their descendants if `recursive` is truthy.
  // - `type` - can be used to filter nodes of a specified type (defaults to "")
  // - `includeDel` - set to true to include deleted nodes (defaults to false)
  // - `recursive` - set to true to recursively retrieve all descendants matching
  //   the criteria. In this case, each returned NodeEdge will contain a
  //   `children` property, which is an array of that Node's descendant NodeEdges.
  //   Set to "flat" to return a single flattened array of NodeEdges.
  // - `opts` are options passed to the NATS request
  async getNodesForUser(userID, { type, includeDel, recursive, opts } = {}) {
    // Get root nodes of the user node
    const rootNodes = await this.getNode(userID, { parent: "all", opts });

    // Create function to filter nodes based on `type` and `includeDel`
    const filterFunc = (n) => {
      const tombstone = n.edgepointsList.find((e) => e.type === "tombstone");
      return (
        (!type || n.type === type) &&
        (includeDel || !tombstone || tombstone.value % 2 === 0)
      );
    };

    const _cache = {};
    const flat = recursive === "flat";
    const parentNodes = await Promise.all(
      rootNodes.filter(filterFunc).map(async (n) => {
        const [parentNode] = await this.getNode(n.parent, { opts });
        if (!recursive) {
          return parentNode;
        }
        const children = await this.getNodeChildren(n.parent, {
          type,
          includeDel,
          recursive,
          opts,
          _cache,
        });
        if (flat) {
          return [parentNode].concat(children);
        }
        return Object.assign(parentNode, { children });
      })
    );
    if (flat) {
      return parentNodes.flat();
    }
    return parentNodes;
  },

  // subscribePoints subscribes to `p.<nodeID>` and returns an async
  // iterable for Point objects
  subscribePoints(nodeID) {
    const sub = this.subscribe("p." + nodeID);
    // Return subscription wrapped by new async iterator
    return Object.assign(Object.create(sub), {
      async *[Symbol.asyncIterator]() {
        // Iterator reads and decodes Points from subscription
        for await (const m of sub) {
          const { pointsList } = Points.deserializeBinary(m.data).toObject();
          // Convert `time` to JavaScript date and return each point
          for (const p of pointsList) {
            p.time = new Date(p.time.seconds * 1e3 + p.time.nanos / 1e6);
            yield p;
          }
        }
      },
    });
  },

  // sendNodePoints sends an array of `points` for a given `nodeID`
  // - `ack` - true if function should block waiting for send acknowledgement
  // - `opts` are options passed to the NATS request
  async sendNodePoints(nodeID, points, { ack, opts }) {
    const payload = encodePoints(points);
    if (!ack) {
      await this.publish("p." + nodeID, payload, opts);
    }

    const m = await this.request("p." + nodeID, payload, opts);

    // Assume message data is an error message
    if (m.data && m.data.length > 0) {
      throw new Error(`error sending points for node '${nodeID}': ` + m.data);
    }
  },

  // subscribeMessages subscribes to `node.<nodeID>.msg` and returns an async
  // iterable for Message objects
  subscribeMessages(nodeID) {
    const sub = this.subscribe("node." + nodeID + ".msg");
    // Return subscription wrapped by new async iterator
    return Object.assign(Object.create(sub), {
      async *[Symbol.asyncIterator]() {
        // Iterator reads and decodes Messages from subscription
        for await (const m of sub) {
          yield Message.deserializeBinary(m.data).toObject();
        }
      },
    });
  },

  // subscribeNotifications subscribes to `node.<nodeID>.not` and returns an async
  // iterable for Notification objects
  subscribeNotifications(nodeID) {
    const sub = this.subscribe("node." + nodeID + ".not");
    // Return subscription wrapped by new async iterator
    return Object.assign(Object.create(sub), {
      async *[Symbol.asyncIterator]() {
        // Iterator reads and decodes Messages from subscription
        for await (const m of sub) {
          yield Notification.deserializeBinary(m.data).toObject();
        }
      },
    });
  },
});

// decodeNodesRequest decodes a protobuf-encoded NodesRequest and returns
// the array of nodes returned by the request
function decodeNodesRequest(data) {
  const { nodesList, error } = NodesRequest.deserializeBinary(data).toObject();
  if (error) {
    throw new Error("NodesRequest decode error: " + error);
  }

  for (const n of nodesList) {
    // Convert `time` to JavaScript date for each point
    for (const p of n.pointsList) {
      p.time = new Date(p.time.seconds * 1e3 + p.time.nanos / 1e6);
    }
    for (const p of n.edgepointsList) {
      p.time = new Date(p.time.seconds * 1e3 + p.time.nanos / 1e6);
    }
  }
  return nodesList;
}

// encodePoints returns protobuf encoded Points
function encodePoints(points) {
  const payload = new Points();
  // Convert `time` from JavaScript date if needed
  points = points.map((p) => {
    if (p instanceof Point) {
      return p;
    }
    let { time } = p;
    const { type, key, index, value, text, tombstone, data } = p;
    p = new Point();
    if (!(time instanceof Timestamp)) {
      let { seconds, nanos } = time;
      if (time instanceof Date) {
        const ms = time.valueOf();
        seconds = Math.round(ms / 1e3);
        nanos = (ms % 1e3) * 1e6;
      }
      time = new Timestamp();
      time.setSeconds(seconds);
      time.setNanos(nanos);
    }
    p.setTime(time);
    p.setType(type);
    if (key) {
      p.setKey(key);
    }
    if (index) {
      p.setIndex(index);
    }
    if (value || value === 0) {
      p.setValue(value);
    }
    if (text) {
      p.setText(text);
    }
    if (tombstone) {
      p.setTombstone(tombstone);
    }
    if (data) {
      p.setData(data);
    }
    return p;
  });
  payload.setPointsList(points);
  return payload.serializeBinary();
}
