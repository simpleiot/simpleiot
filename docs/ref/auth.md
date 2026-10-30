# Authentication and authorization

Every connection to a Simple IoT instance answers two questions: who is this
(authentication), and what may they do (authorization). This page describes the
model from the outside, as a person, a device, or a developer meets it. The
mechanisms, the NATS subjects, and the audit record are in
[Security](security.md).

The model in one sentence: **the tree says where, the role says what.** Where a
user sits in the tree decides which nodes they can reach, the role on that
membership decides what they can do there, and each node type's schema says
which of its points that role may change.

## Status

Each part of the model is marked in the text as implemented or _planned_. In
summary:

| Part                                                               | Status                                                                                                                                                                  |
| ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Shared token, sign-in tokens, device keys, revocation              | Implemented                                                                                                                                                             |
| Enrollment tokens                                                  | Implemented; placing a device in the token's group is [in progress](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-10-20-enrollment-tokens-per-group.md) |
| Group scope: a user reaches the groups they belong to              | Implemented                                                                                                                                                             |
| Secret values blanked in replies to a browser                      | Implemented                                                                                                                                                             |
| Roles on memberships (`viewer`, `operator`, `technician`, `admin`) | Planned                                                                                                                                                                 |
| Point levels from the client schema                                | Planned                                                                                                                                                                 |
| Hidden node types for roles below `admin`                          | Planned                                                                                                                                                                 |
| Root-only node types                                               | Planned; [security follow-ups](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-09-19-security-followups.md) item 1                                        |
| Role-aware UI                                                      | Planned                                                                                                                                                                 |

Until the planned parts land, every member of a group can read and write
everything under it; see
[What an account can do](security.md#what-an-account-can-do).

## Who connects

| Principal                 | Presents                             | Reaches                                                     |
| ------------------------- | ------------------------------------ | ----------------------------------------------------------- |
| The operator's tools      | the shared token (`SIOT_AUTH_TOKEN`) | everything on the instance                                  |
| A person in a browser     | email and password, then a token     | the groups the person belongs to, at the role held in each  |
| A device syncing upstream | its device key                       | its own subtree on the upstream                             |
| A device with no key yet  | an enrollment token                  | one request: ask for a credential                           |
| The server's own clients  | the server's connection              | everything, with each client checking its own write targets |

The shared token is for the `siot` command line and scripts on the host. It is
not something a person or a device is given.

## What a person experiences

**Signing in.** A person signs in with their email and password. The tree they
see starts at the groups they belong to, and nothing above or beside those
groups exists for them. A user directly under the root sees the whole instance.

**Roles** (_planned_). Each place a user sits in the tree carries a role on that
membership. One person can be an admin of one site and a viewer of another. The
four roles form a ladder, and each includes everything below it:

| Role         | Can                                                                                                      |
| ------------ | -------------------------------------------------------------------------------------------------------- |
| `viewer`     | see nodes and live values                                                                                |
| `operator`   | also change the values that run the system: a setpoint, an output, whether a rule is enabled             |
| `technician` | also change descriptions and configuration, and add, move, and delete ordinary nodes                     |
| `admin`      | also manage users, groups, enrollment tokens, and device credentials in the group, and set roles on them |

Every user can edit their own profile and password. A missing role reads as
`viewer`, so a membership nobody set a role on can look and not touch. The
default `admin` account created on first start is an admin under the root.

Some examples of how the roles are meant to be used:

- A customer's dashboard account is a `viewer` in that customer's group.
- A plant operator is an `operator` in the site group: they can change a
  setpoint, switch an output, and silence a rule, but cannot rename a sensor or
  change a Modbus address.
- A commissioning technician is a `technician` in the site group: they can add a
  Modbus device, write the rules, and rename things, but cannot see who else has
  access or hand out credentials.
- A customer's own administrator is an `admin` in the customer group: they add
  their staff, choose each person's role, approve the devices that enroll there,
  and never see another customer.
- The operator's staff are `admin` under the root.

**Which points each role can change** (_planned_). A node is a list of points,
and a role may change some of them and not others. The schema of each node type
says which role a point needs, so the rule is the same on every instance and
needs no setup. A point that is not marked needs `technician`, so configuration
is the default and only the handful of points meant for day-to-day operation are
opened to an `operator`. The UI shows a point a user cannot change as read only
rather than letting them try.

For example, on a Modbus output:

| Point         | Needed role  | Why                     |
| ------------- | ------------ | ----------------------- |
| `value`       | `operator`   | switching it is the job |
| `description` | `technician` | naming is configuration |
| `address`     | `technician` | wiring is configuration |

Node structure follows the same ladder: adding, moving, or deleting an ordinary
node needs `technician`, and doing the same to a user, group, enrollment token,
or device credential needs `admin`.

**Setting a role** (_planned_). The user node shows one role selector for each
group it sits in. Only an admin of that group can change it. In a node file the
role is an edge point, since it belongs to the membership and not to the user:

```yaml
nodes:
  - user:
      parent: Site 1
      email: joe@example.com
      pass: his-password
      edgePoints:
        role: admin
```

**What an admin does not get** (_planned_). A group admin runs the group, not
the host. Nodes that act on the machine the instance runs on, such as software
update, time, network, kiosk browser, and upstream sync, work only directly
under the root. An admin of a group can create one, but it does nothing there
and says so.

**What is hidden** (_planned_). A role below `admin` does not see other users,
device credentials, or enrollment tokens in their group. Those nodes are absent
from the tree rather than shown greyed out, and their live values are not
delivered either. A user always sees their own user node.

**Secrets** (implemented). Secret values such as passwords and tokens are never
sent to a browser; the field shows blank and typing in it sets a new value.

**Losing access.** Removing a user from a group, deleting the user, or changing
their password signs them out of that scope within seconds.

## What a device experiences

**First contact.** A device with no credential presents an enrollment token. It
is created as a device node in the group the token sits in (in progress; today
under the root), with a credential that is either approved at once or waiting
for an admin, depending on the token. A device node that already exists always
waits for approval.

**Afterward.** The device connects with its own key and can read and write its
own subtree on the upstream, and nothing else. It cannot see other devices,
users, or the upstream's configuration beyond what is sent to it.

**Revocation.** An admin disables or deletes the credential. The device is
disconnected, keeps running on its own, records why on its sync node, and
retries each minute. Enabling the credential again lets it back in with what it
queued while away.

## What a developer experiences

**Command line and scripts on the host** use the shared token and reach
everything. `siot` commands take it from `SIOT_AUTH_TOKEN`.

**The HTTP API** takes an `Authorization` header: a person's sign-in token or a
device's signed token. The request is limited to what that principal reaches in
the browser or over sync, and a node outside that scope answers with an error
rather than an empty result.

**The NATS API** needs no configuration from a client. A connection is granted
exactly the subjects its credential allows, and a request for a node outside
scope answers `not in scope`. A client written against the Go library and the
shared token sees the full subject space described in the
[API reference](api.md).

**Writing a client** inside Simple IoT needs no scope handling for reads and
ordinary writes; clients run on the server's connection. Three things to decide
when adding a node type:

- If a value from a point names a write target, a period, or a path, check it
  through the helpers in `client/scope.go` and `client/paths.go`, so a user
  cannot aim the client at a node they do not own.
- Mark the points an operator may change (_planned_); every other point needs
  `technician`. See [Marking points in a client](#marking-points-in-a-client).
- If the type manages people, credentials, or the host, add it to the privileged
  or root-only list in `data/` (_planned_), so the UI hides it from the wrong
  roles and the server refuses to run it in the wrong place.

### Marking points in a client

_Planned._ A client declares its points with struct tags, and the role a point
needs rides alongside as a second tag on the same field. The Modbus IO client:

```go
type ModbusIo struct {
	ID              string  `node:"id"`
	Parent          string  `node:"parent"`
	Description     string  `point:"description"`
	Address         int     `point:"address"`
	DataFormat      string  `point:"dataFormat"`
	Scale           float64 `point:"scale"`
	Value           float64 `point:"value" role:"operator"`
	ValueSet        float64 `point:"valueSet" role:"operator"`
	Disabled        bool    `point:"disabled" role:"operator"`
	ErrorCount      int     `point:"errorCount"`
	ErrorCountReset bool    `point:"errorCountReset" role:"operator"`
}
```

A rule, where an operator may silence it and nothing more:

```go
type Rule struct {
	ID          string      `node:"id"`
	Parent      string      `node:"parent"`
	Description string      `point:"description"`
	Disabled    bool        `point:"disabled" role:"operator"`
	Active      bool        `point:"active"`
	Error       string      `point:"error"`
	Conditions  []Condition `child:"condition"`
	Actions     []Action    `child:"action"`
}
```

A variable, where the value is the whole point of the node:

```go
type Variable struct {
	ID           string             `node:"id"`
	Parent       string             `node:"parent"`
	Description  string             `point:"description"`
	VariableType string             `point:"variableType"`
	Value        map[string]float64 `point:"value" role:"operator"`
}
```

How to read the tags:

- **A field with no `role` tag needs `technician`.** Description, address, and
  scale are untagged, so configuration is the default and a new field is closed
  until someone opens it.
- **`role:"operator"` is the only tag most clients use**: setpoints, outputs,
  disable flags, and counter resets. A rule's `active` and `error` stay untagged
  because the client writes them, not a person.
- **`role:"admin"` is for the rare point a technician must not touch** on an
  otherwise ordinary node. None of the structs above need it; a user's `pass`
  and `email` are the likely case, with the user's own node exempt.
- **Writes from the client itself are unaffected.** The Modbus client writes
  `value` on the server's connection, and the role check applies only to a
  person's write. Tagging `value` says a person may set it, and nothing about
  the client.

The manager already reflects over each client's struct to decode nodes, so the
tags are collected into one table per node type at registration, with no second
declaration. The frontend fetches the same table from the server, which is what
lets it render an untagged field read only for an operator.

A tag cannot express a level that depends on another point. The `readOnly` flag
on a Modbus IO is the example: an operator may write `valueSet` on an output and
should not on an input. The store accepts the write either way and the client
ignores it on an input, as it does today. That stays with the client rather than
growing the tag.

### How a write is checked

_Planned._ The tag becomes a table at start-up, and the table is checked at the
two places a person's write enters the system. Nothing travels on the wire.

**Telling a person from a device.** The path a point takes is fixed by the NATS
server when the connection is made, so the store never has to guess who wrote
it:

- A person's write arrives on the user namespace,
  `u.<anchor>.<user>.p.<node>.<type>.<key>`. A sign-in token is granted publish
  on that prefix and nothing else, so a browser cannot put a point anywhere
  else, and the store's user handler reads the anchor and the user from the
  subject.
- A device's write never arrives as a point request at all. A device key is
  granted publish only on its own replica stream, `inst.X.X.>`, which the store
  consumes and merges where it stays inside boundary `X`. A device is refused on
  `p.>` and on `u.>`.
- The server's own clients and the shared token publish on the plain
  `p.<node>.<type>.<key>` subject, which only a full-access connection holds.
- Over HTTP the route resolves the `Authorization` header to a principal first:
  a user token, a device token, or the shared token. The scope and role checks
  hang off that principal.

The origin on a point is set by the server afterward for the record, and nothing
a client puts there is trusted.

**The table.** The manager already reflects over each client's struct to learn
its node type; the same pass reads the `point` and `role` tags and registers
them in a map in `data/` keyed by node type and point type. Node types with no
client, such as group, device, and user, get a literal entry there. One function
answers which role a point needs, with `technician` for anything unlisted,
including a custom node type.

**The writer's role.** Both enforcement points already resolve a user's anchors
from the user's edges in the store's cache. The `role` point sits on that same
edge, so the role per anchor comes from the same lookup. A node reachable
through two anchors with different roles takes the higher one.

**The NATS path.** The store's user handler already checks that the node is
under the anchor. The added step looks up the node's type in the edge cache,
looks up the role each point needs, and refuses the request when the writer's
role on that anchor is below any of them. Viewers never reach this code, since
their permission set omits the point and edge subjects. Edge writes follow the
structure rule in the same handler: an ordinary node needs `technician`, a
privileged node or the `role` edge point needs `admin`.

**The HTTP path.** The point route checks scope and then publishes on the
server's own connection, so the store's user handler never sees it. It makes the
same check before publishing, through one shared function with the NATS path, so
the two cannot drift.

**Who is not checked.** The shared token, the server's clients, and a device's
stream are not people. A rule action written by a technician runs with the
server's authority, so rule targets on privileged nodes need their own rule.

**The UI.** On sign-in the frontend fetches the table once and renders a field
the user cannot change as read only. The store remains the enforcement, so a UI
that is out of date gets a refusal rather than a silent success.

**Role changes.** The authorizer already closes a user's connections when their
edges change, so a demotion refreshes the permission set within seconds, and the
store's lookup reads the edge cache live.

## Setting up an instance

In order:

1. Set `SIOT_AUTH_TOKEN` and change the `admin` password.
2. Make a group per customer or site, and add users to each with a role.
3. Put an enrollment token in each group and give it to the devices meant for
   that group. Prefer tokens that leave credentials pending.
4. Once every device has a credential, set `SIOT_DEVICE_AUTH=required`.

The [deployment checklist](security.md#deployment-checklist) has the full list
for an instance that others can reach.
