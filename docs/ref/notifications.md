# Notifications

(see notification [user documentation](../user/notifications.md))

Notifications are messages that are sent to users. There are several concerns
when processing a notification:

1. The message itself and how it is generated.
2. Who receives the messages.
3. Mechanism for sending the message (Twilio SMS, SMTP, ntfy, etc.)
4. State of the notification
   1. Sequencing through a list of users
   2. Tracking if it was acknowledged and by who
5. Distributed concerns (more than one SIOT instance processing notifications)
   1. Synchronization of notification state between instances.
   2. Which instance is processing the notification.

Notifications can be initiated by:

1. Rules
2. Users sending notifications through the web UI

## Notification Data Structures

Notifications and messages travel as points carrying a JSON payload (`DataType`
is JSON and the payload lives in the point `Data` field). This means they get
everything points get for free: they are persisted in the store, synchronized
between instances, recorded in history, and visible to clients through the
standard `Points()` mechanism and `up.<parent>.>` subscriptions. No side channel
is involved.

Two payload types divide the work (see `data/notification.go` and
`data/message.go`):

- **Notification** (point type `notification`) says _what happened_. It is
  published on the node that raised it — a rule node when a notify action fires,
  or any node targeted by the web UI's message function. It carries a UUID, the
  source node, a subject, and a message.
- **Message** (point type `message`) says what happened _and who to send it to_.
  It is published on a user node and carries the notification ID plus the user's
  phone and email.

Both point types use a fixed (empty) key, so the point merge collapses each new
notification over the previous one: a node's state carries only its most recent
notification, and the full history lives in the JetStream stream. Delivery is
not affected by this collapse because clients receive every published point
through their subscriptions, independent of the merged state.

## Delivery

Delivery happens in up to two hops, and the second hop is optional per service:

1. Each user node runs a client subscribed to `up.<parent>.>`. When a
   notification point appears anywhere in the parent's subtree, the user client
   emits a message point on its own node, populated with the user's contact
   information. Users with no phone or email emit nothing.
2. Each messaging service node (`msgService`) runs a client with the same
   subscription. Twilio and SMTP need per-user addressing, so they consume
   message points. A service with a global destination — an ntfy topic —
   consumes notification points directly and works with no user nodes in scope.

Scope comes from position in the tree: a service or user sees notifications
raised anywhere in its parent's subtree, so moving a node changes its
notification scope.

## Deduplication

A user mirrored into two groups runs two client instances and emits two message
points, and two branches of the tree can converge on the same service node. The
service client deduplicates at the point of delivery, keyed by (notification ID,
destination address), which enforces the invariant that matters — one message
per destination per notification — regardless of topology. The notification ID
survives synchronization, so this also covers duplicates arriving from another
instance.

Deduplication state is held in memory with entries expiring after one hour. It
is not persisted, so a service client restart inside the window can produce a
duplicate delivery. This is a bounded and accepted trade for keeping the client
free of storage concerns.

## Notification State

Acknowledgement and escalation — sequencing through a list of users and tracking
who acknowledged — are not implemented. That state does not fit a JSON payload
on a single point, because it would require read-modify-write from several
instances with no conflict resolution. Implementing it means promoting the
notification to a node with ordinary points; that is the trigger for revisiting
this design.
