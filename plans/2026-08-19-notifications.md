# Plan: Notifications and Messaging Clients

**Branch:** `feat/notifications` **Branched from:** `d2bbb2a3`

## Context

Notification delivery has not worked since December 2022. Commit `d451cad0`
("move type to edge struct in DB") moved node storage under the edge struct, and
the notification code in the store depended on the old direct database calls
(`st.db.getNodes`, `st.db.edgeUp`, `st.db.node`). Rather than port it, the whole
of `store/notification.go` was wrapped in a block comment with the note
`// TODO this code is currently not used and needs to be moved to a client`, and
the two subscriptions that fed it were commented out in
`store/store.go:125-132`.

Both ends of the chain still publish into a void:

- `client/rule.go:692` — a rule with a `notify` action builds a
  `data.Notification` and publishes it to `node.<ruleID>.not`.
- `api/nodes.go:243` — the web UI's "send message to users" endpoint publishes
  to the same subject.

Nothing subscribes to `node.*.not` or `node.*.msg` except `client/debug.go`,
which logs them and discards them. Users can still create and configure a Twilio
`msgService` node in the UI, and it has no runtime effect.

What survives and works: `msg/twilio.go` (the Twilio wrapper itself) and
`cmd/send-sms/` (a standalone CLI that reads `TWILIO_SID`, `TWILIO_AUTH_TOKEN`,
and `TWILIO_FROM` from the environment).

This plan rebuilds notifications on points and clients, which is where the rest
of the system moved while this code sat idle.

## Design Decisions

**A notification is a point carrying a JSON payload, not a protobuf on a side
channel.** The current `data.Notification` travels on `node.<id>.not`, which
bypasses the store completely: it is never persisted, never synchronized between
instances, has no history, and is invisible to the standard client mechanism,
since `Client.Points()` only ever receives points. Encoding the notification as
a point makes all of that work with no new machinery.

`PointDataTypeJSON` already exists (`data/point.go:65`), `Txt()` already handles
it (`data/point.go:439`), and `client/serial-shell.go:325` already uses it to
carry structured payloads from an MCU. This follows an established pattern in
the codebase.

Note that `docs/ref/notifications.md` currently rejects this idea: "We could
encode the message as JSON in the Point text field, but it would be nice to have
something a little more descriptive." That was written when `Point` had a bare
`Text string` field and no `DataType`. With a typed `PointDataTypeJSON` the
objection no longer applies, and the paragraph should be rewritten so the
documentation agrees with the code.

**Notification and message points use a fixed key.** Points merge by (Type, Key)
into the node's point set, and `Points.Merge` appends any key it has not seen
before (`data/point.go:804`):

```go
if !pFound {
    *ps = append(*ps, pIn)
    modified = true
}
```

That set is the node's current state and never shrinks, so giving each
notification a unique key would grow the source node's point set without bound.
With a fixed key, each notification overwrites the previous one: the node
carries its most recent notification, and the history lives in the JetStream
stream where it belongs.

Collapsing the stored tip does not lose deliveries. Points are published on
`p.<nodeID>.<type>.<key>` and fanned out to subscribers independently of the
merge, so a client subscribed to `up.<parent>.>` sees every notification point
published, including two that arrive close enough together that the second
overwrites the first in stored state.

**Delivery happens in two hops, and the second hop is optional per service.** A
notification says what happened. A message says what happened _and_ who to send
it to. Services divide cleanly along that line:

- Twilio and SMTP need per-user addressing, so they consume message points.
- A service with a global destination — an ntfy.sh topic, a webhook, an MQTT
  topic — needs no user data at all, so it consumes notification points directly
  and fires even when no user node is in scope.

Which point type a service client listens for is therefore part of its
configuration, not a hardcoded path. This is the main improvement over the
original design, which assumed every service needed a per-user message.

**The user node performs the second hop.** `client/user.go` already defines the
`User` struct with `node:` and `point:` field tags, but there is no client
behind it. Adding one, subscribed to `up.<parent>.>` the way
`client/rule.go:219` does, gives each user node visibility of every notification
raised anywhere in its parent's subtree. It then emits a message point on its
own node carrying its own contact information, which flows up to the
`msgService` nodes above it.

This is a direct implementation of the behavior described in
`docs/user/notifications.md`: "At each parent node, users potentially listen for
notifications." Each user node decides for itself that a notification is in
scope and addresses a copy to itself. No central graph walk and no store
involvement.

**Scope comes from the subtree, not from an explicit recipient list.** Both the
user client and the service client subscribe to `up.<parent>.>`, so a Twilio
node under `Company XYZ` sees notifications and messages from anywhere under
`XYZ`, and a user under `Plant A` sees notifications from anywhere under
`Plant A`. Moving a node in the tree changes its notification scope, which is
the intended behavior.

**Duplicates are suppressed at delivery, keyed by (notification ID, address).**
Clients are keyed by parent and node (`client/manager.go:401`):

```go
func mapKey(node data.NodeEdge) string {
	return node.Parent + "-" + node.ID
}
```

A user mirrored into two groups therefore runs two client instances, and each
emits its own message point. The original code was aware of this — it is why
`data.Message` carries a `ParentID`, and why `store/notification.go` restricts
the service lookup to the message's own parent at the first level. That
mitigation depends on the shape of the graph, and it does not close the case
where two branches converge on the same service node.

Deduplicating at the point of delivery instead enforces the invariant that
actually matters: one message per destination address per notification,
regardless of topology. It also covers the multi-instance duplicate, since the
notification ID survives synchronization between instances.

## Point and Node Types

Added to `data/schema.go`:

| Constant                | Value          | Notes                  |
| ----------------------- | -------------- | ---------------------- |
| `PointTypeNotification` | `notification` | JSON payload, key `""` |
| `PointTypeMessage`      | `message`      | JSON payload, key `""` |
| `PointTypeTopic`        | `topic`        | ntfy topic             |
| `PointValueNtfy`        | `ntfy`         | new `service` value    |

Reused as-is: `PointTypeService`, `PointTypeSID`, `PointTypeAuthToken`,
`PointTypeFrom`, `PointTypeURL`, `PointTypePhone`, `PointTypeEmail`,
`PointValueTwilio`, `PointValueSMTP`, `NodeTypeMsgService`, `NodeTypeUser`.

Removed: `PointMsgAll` and `PointMsgUser` (`data/schema.go:199-200`). Both are
dead constants with no references anywhere in the tree.

## Payload Schemas

Both payloads are marshalled from a Go struct rather than a map. `Points.Merge`
compares payloads with `bytes.Equal(pIn.Data, p.Data)` (`data/point.go:791`), so
field ordering is significant and must be stable.

**Notification** — on the node that raised it:

```go
type Notification struct {
	ID         string `json:"id"`         // UUID, the deduplication key
	SourceNode string `json:"sourceNode"` // node that triggered it
	Subject    string `json:"subject"`
	Message    string `json:"message"`
}
```

**Message** — on the user node:

```go
type Message struct {
	NotificationID string `json:"notificationID"` // carried through for dedup
	UserID         string `json:"userID"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	Subject        string `json:"subject"`
	Message        string `json:"message"`
}
```

The message carries the notification ID unchanged. That is what lets a service
client recognize two messages generated from one notification by two instances
of a mirrored user node.

`Parent` is dropped from the notification, and `ParentID` from the message. Both
existed only to support the topology-dependent duplicate suppression that
delivery-side deduplication replaces.

## Client Configuration Structs

**User** — extend the existing struct in `client/user.go`. No new fields; the
client is what is missing.

**MsgService** — replaces `data/message-service.go`:

```go
type MsgService struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Service     string `point:"service"`   // twilio, smtp, ntfy
	SID         string `point:"sid"`       // twilio
	AuthToken   string `point:"authToken"` // twilio, ntfy
	From        string `point:"from"`      // twilio, smtp
	URL         string `point:"url"`       // ntfy server, default https://ntfy.sh
	Topic       string `point:"topic"`     // ntfy
	Error       string `point:"error"`
}
```

`data.NodeToMsgService` goes away — `data.MergePoints` with the `point:` tags
replaces it, matching how every other client handles configuration.

## Delivery and Deduplication

The service client keeps an in-memory set keyed by (notification ID, destination
address), with entries expiring on a time window rather than a count. A count
bound would be defeated by a burst; the window has to be long enough to cover a
synchronization catch-up after an outage, when the same notification can arrive
long after it was raised.

The initial window is one hour, with the set swept on a ticker. This is not
persisted across restarts. A restart within the window can therefore produce a
duplicate message, which is an acceptable trade for keeping the client free of
storage concerns — see [Open Questions](#open-questions).

## Implementation Plan

### Phase 1: Schema and Payloads

- Add the point types and values above to `data/schema.go`; remove `PointMsgAll`
  and `PointMsgUser`.
- Rewrite `data/notification.go` and `data/message.go` as plain structs with
  stable JSON marshalling and helpers to read and write them as points. Delete
  the protobuf conversion functions.
- Remove `Notification` and `Message` from the protobuf definitions and
  regenerate with `siot_protobuf`.
- Unit tests for round-tripping each payload through a point.

### Phase 2: Retire the Side Channel

- Delete `store/notification.go` entirely and the commented subscription block
  in `store/store.go:125-132`.
- Change `client/rule.go` so the `notify` action publishes a notification point
  on the rule node instead of publishing to `node.<id>.not`. This removes the
  `// TODO this notify code needs to be reworked` comment at
  `client/rule.go:684`.
- Change the `not` case in `api/nodes.go:225` to publish a notification point on
  the target node.
- Remove the `node.*.not` and `node.*.msg` subscriptions from `client/debug.go`;
  notifications now appear in the existing `p.>` output.

At the end of this phase notifications are visible as points and flow through
the store, with no delivery yet.

### Phase 3: User Client

- Add `NewUserClient` in `client/user.go` following the shape of
  `client/ntp.go`: `Run`, `Stop`, `Points`, `EdgePoints`, a stop channel, and a
  points channel.
- Subscribe to `up.<parent>.>` in `Run`, filtering for notification points,
  using the same subject decoding as `client/rule.go:219-240`.
- On each notification, emit a message point on the user's own node populated
  from the user's `phone` and `email` points. Skip when both are empty.
- Register it in `client/client.go` with `NewManager(nc, NewUserClient, nil)`.
- Tests: a notification raised on a sibling node under the same parent produces
  a message point on the user node; a notification raised outside the parent's
  subtree does not.

### Phase 4: Message Service Client

- Add `client/msg-service.go` with `NewMsgServiceClient` and the config struct
  above; delete `data/message-service.go`.
- Subscribe to `up.<parent>.>`. Consume message points when the service is
  `twilio` or `smtp`, and notification points when the service is `ntfy`.
- Deduplicate on (notification ID, address) with the expiring set described
  above.
- Twilio delivery calls the existing `msg.NewTwilio` and `SendSMS` unchanged.
- Report delivery failures on an `error` point on the service node, following
  the `processError` pattern in `client/rule.go:618`.
- Register it in `client/client.go`.
- Tests: a message point produces one send; the same notification arriving twice
  by different paths produces one send; a notification with no user in scope
  still reaches an ntfy service.

### Phase 5: ntfy and SMTP

- Add `msg/ntfy.go`. Publishing to ntfy is an HTTP POST of the message body to
  `<url>/<topic>`, with the subject in the `Title` header and an optional bearer
  token.
- Add `msg/smtp.go`. `PointValueSMTP` has existed in the schema since the
  original implementation with nothing behind it; this is the first working
  version.
- Extend `cmd/send-sms/` into `cmd/send-msg/`, or leave it as-is — see
  [Open Questions](#open-questions).

### Phase 6: Frontend

- `frontend/src/Components/NodeMessageService.elm` currently offers Twilio only
  and shows all fields unconditionally. Add `ntfy` and `smtp` to the service
  option list and show only the fields relevant to the selected service, as
  `NodeGps.elm` does for its source selection.
- Add the new point types to `frontend/src/Api/Point.elm`.
- Show the most recent notification on a node, reading the notification point
  and decoding the JSON payload.
- Run `npx elm-review` and `npx elm-test`.

### Phase 7: Documentation and Changelog

- Rewrite the "Notification Data Structures" section of
  `docs/ref/notifications.md`. The Point-versus-protobuf discussion, the
  separate `Notification` and `Message` protobuf types, and the note about JSON
  in a text field all describe a design that no longer exists.
- Update `docs/user/notifications.md`. It currently describes this as working
  functionality, which has been misleading for some time. Add the ntfy case,
  where no user node is required.
- Note in `docs/ref/api.md` that `node.<id>.not` and `node.<id>.msg` are gone;
  they are documented at `docs/ref/api.md:79-83`.
- Changelog entry under `## Next`.

## Files Touched

| File                                             | Change                                |
| ------------------------------------------------ | ------------------------------------- |
| `data/schema.go`                                 | new point types, remove dead ones     |
| `data/notification.go`                           | rewrite as JSON payload struct        |
| `data/message.go`                                | rewrite as JSON payload struct        |
| `data/message-service.go`                        | delete                                |
| `internal/pb/*`                                  | remove Notification and Message       |
| `store/notification.go`                          | delete                                |
| `store/store.go`                                 | remove commented subscriptions        |
| `client/rule.go`                                 | notify action publishes a point       |
| `client/user.go`                                 | add the user client                   |
| `client/msg-service.go`                          | new                                   |
| `client/client.go`                               | register both clients                 |
| `client/debug.go`                                | drop the two subject subscriptions    |
| `api/nodes.go`                                   | `not` endpoint publishes a point      |
| `msg/ntfy.go`, `msg/smtp.go`                     | new                                   |
| `frontend/src/Components/NodeMessageService.elm` | service selection, conditional fields |
| `frontend/src/Api/Point.elm`                     | new point types                       |
| `docs/ref/notifications.md`                      | rewrite the data structures section   |
| `docs/user/notifications.md`                     | correct, and cover ntfy               |
| `docs/ref/api.md`                                | remove the retired subjects           |

## Risks

**Deduplication state is per-process and not persisted.** A service client
restart inside the deduplication window can send a duplicate. Bounded and
acceptable, but worth stating plainly.

**Notification storms.** A flapping rule generates a notification on every
inactive-to-active transition, and every one of them is delivered. The rule
client already has `minActive` (`data/schema.go:185`), which limits this at the
source, but nothing rate-limits at the service. If this proves to be a problem
in practice, a minimum interval point on the service node is the natural place
to add one.

**Contact information is written into the message payload.** Phone numbers and
email addresses land in points, which are persisted to JetStream and
synchronized between instances. They are already stored as points on user nodes
today, so this does not expose anything new, but it does copy them into a second
location with its own retention.

**Loss of the transient-point distinction.** The comment at `data/schema.go:197`
describes points "not stored in the state of any node, but recorded in the time
series database." Notification and message points are stored in node state under
this design, holding the most recent value. If keeping node state free of them
matters, that is a change to how the store treats a point type, and it should be
handled separately rather than folded into this plan.

## Open Questions

1. **Deduplication window length.** One hour is a guess. The right value depends
   on how far behind a synchronization catch-up can run after an outage, which
   Stage 3 synchronization work should be able to answer.

2. **Should a notification also reach users above its own parent?** Today's
   subscription gives a user node visibility of its parent's subtree only. A
   notification raised deep in the tree reaches users beside it and users in
   ancestor groups, because those users' own parents contain the source. This is
   believed to match the documented intent, and is worth confirming against the
   example hierarchy in `docs/user/notifications.md` before Phase 3.

3. **Acknowledgement and escalation.** `docs/ref/notifications.md` lists
   notification state — sequencing through a list of users, tracking who
   acknowledged — as a concern. None of it is implemented today and none of it
   is in this plan. It does not fit a JSON payload on a single point, because it
   would mean read-modify-write from several instances with no conflict
   resolution. Building it means promoting the notification to a node with
   ordinary points. That is the trigger for revisiting this design.

4. **What to do with `cmd/send-sms/`.** It works and is useful for testing
   Twilio credentials outside a running instance. It could be generalized to
   cover ntfy and SMTP, or left alone.
