---
name: siot-security-audit
description:
  Use when auditing Simple IoT for security issues, reviewing a change that
  touches authentication, authorization, enrollment, sync, or a network-facing
  parser, or checking whether an open item in the security cleanup plan still
  reproduces. Triggers on requests like "do a security audit", "is this scoped
  correctly", "can a user reach X", "re-check the security plan", or "prove this
  finding". Covers who the principals are, where enforcement lives, how to prove
  a finding against a test server, and where results are recorded. It does not
  list the findings themselves; those live in the plan.
---

# Auditing Simple IoT

This skill holds what a fresh session cannot derive quickly: the principals, the
enforcement points, and a test harness that works. It is deliberately not a
checklist. The most useful findings so far came from reading an entry point with
no list in hand and asking what each principal could do there.

## Order of work

1. **List the entry points and principals as they are today**, from the code,
   before reading any earlier findings. Every new feature adds one or the other,
   and this skill will not know about it. Start from `server/server.go` (what is
   started), `server/nats-server.go` (listeners), `api/` (HTTP routes), the
   `nc.Subscribe` calls in `store/store.go` and `server/`, and
   `client.DefaultClients`.
2. **Read each enforcement point and form hypotheses.** Write down what you
   expect to be refused and why.
3. **Prove each hypothesis** with a throwaway test (below). A finding that was
   only read in code is labeled that way.
4. **Only then** open
   [`plans/2026-08-24-security-cleanup.md`](../../../plans/2026-08-24-security-cleanup.md)
   and the [Known limitations](../../../docs/ref/security.md#known-limitations)
   table, and use them as a regression pass: which open items still reproduce,
   which closed items stay closed, and which of your findings are new.
5. **Record results** (below).

Reading the plan first anchors the audit on what was already found. Resist it.

## Principals

Each is supposed to be limited as described. The audit question for every entry
point is what each one can reach there.

| Principal                     | How it authenticates                                | Intended limit                                                    |
| ----------------------------- | --------------------------------------------------- | ----------------------------------------------------------------- |
| Anonymous, no token set       | nothing; `SIOT_AUTH_TOKEN` is empty by default      | none: an instance with no token is open on every listener         |
| Shared-token holder           | `SIOT_AUTH_TOKEN` on NATS, WebSocket, MQTT, HTTP    | none; loopback only under `SIOT_DEVICE_AUTH=required`             |
| Credentialed device           | NKey in a `deviceCred` node, or a device-signed JWT | its own boundary `inst.X.X.>`, its own subtree over HTTP          |
| Enrolling key                 | enrollment token plus an NKey that signs the nonce  | publish `enroll.request`, its own inbox                           |
| Browser user                  | user node ID plus sign-in JWT                       | `u.<anchor>.<user>.>` and `up.<anchor>.>` for each group it is in |
| User replicated from a device | same sign-in, on the upstream                       | the device subtree                                                |
| Fieldbus or LAN peer          | none: Modbus, serial, CAN, mDNS, a scraped endpoint | the values it reports                                             |
| Web page on another origin    | none                                                | nothing, unless the instance has no token                         |

The question that has found the most: **can this principal change the set it is
checked against?** A scope check that reads the tree is only as strong as the
principal's inability to write the part of the tree it reads. Edges, user nodes,
`deviceCred` nodes, and anything a device replicates upstream are all tree data.

The second: **whose connection carries the write?** Clients run in the server
process and publish on its full-access connection, so a value in a point that
names a target (a node ID, a URL, a file path, a recipient) is acted on with the
server's authority, not the writer's.

The third: **what happens when the value is absurd?** No goroutine recovers from
a panic, so a point or a frame that panics one client stops the process, and a
stored point does so again at every start.

## Where enforcement lives

- `server/auth.go`: `Check` and the three permission builders
  (`devicePermissions`, `userPermissions`, `enrollPermissions`). Loopback rule
  in `checkToken`.
- `store/store.go`: `handleUserRequest` is the only thing between a browser and
  the store's handlers. `isUnder` and `UserAnchors` in `store/jetstream.go`.
  `userCheck` decides who may sign in.
- `api/nodes.go`: `authenticate`, then the routes. `api/key.go` for the JWT.
- `server/enroll.go`: what an enrollment request may create.
- `server/device-key.go`: the instance's own key.
- `api/server.go`: the WebSocket proxy to `ws://localhost`, which makes every
  proxied connection arrive from loopback.
- `store/replica.go`: what a downstream stream may write into the upstream tree.
- `client/manager.go` and each client's `Run`: what point values reach tickers,
  allocations, file paths, `os/exec`, and outbound connections.
- `modbus/`, `client/serial*.go`, `client/mqtt*.go`, `client/sparkplug.go`,
  `client/metrics-prom*.go`, `client/shelly*.go`: parsers of input from peers.

## Proving a finding

Write a test file named `zz_audit_poc_test.go` in `client/` (package
`client_test`, which has the sync helpers) or `server/`, run it, and delete it
in the same command so it cannot be left behind.

**Use your own ports and data directory.** `server.TestServerOptions` and
`TestServerOptions2` use ports 8900 to 8914 and store data in
`/tmp/siot-test-<ID>`, which a clean start deletes. Another session or a running
test loop on the same checkout will collide with both. A development instance
usually holds 4222, 4223, and 4224.

```go
optsU := server.Options{NatsPort: 8950, HTTPPort: "8951", NatsMonitorPort: 8952,
	NatsWSPort: 8953, NatsServer: "nats://localhost:8950", ID: "auditU",
	AuthToken: "upstream-token", DataDir: "<scratchpad>/auditU"}
nc, root, stop, err := server.TestServerOpts(optsU)
```

`nc` is the server's full-access connection: use it to build the tree and to
check results as the operator would see them. Connect separately as the
principal under test.

```go
// build a tree
client.SendNodeType(nc, client.Group{ID: "G", Parent: root.ID}, "test")
client.SendNodeType(nc, client.User{ID: "U", Parent: "G", Email: "u", Pass: "pw"}, "test")

// browser user: sign in the way the UI does, then connect with the JWT
nodes, _ := client.UserCheck(nc, "u", "pw") // the node of type data.NodeTypeJWT holds the token
user, _ := nats.Connect(uri, nats.UserInfo("U", jwt),
	nats.CustomInboxPrefix(client.InboxPrefix("U")), nats.NoReconnect())

// credentialed device, or an enrolling key
kp, _ := nkeys.FromSeed([]byte(seed))
dev, _ := nats.Connect(uri, nats.Nkey(pub, kp.Sign),
	nats.CustomInboxPrefix(client.InboxPrefix(pub)), nats.NoReconnect())

// points are built with constructors; data.Point has no exported Value or Text
pts := data.Points{
	data.NewPointFloat(data.PointTypeTombstone, "", 0),
	data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeVariable),
}
// a write: an empty reply is success, anything else is the error text
reply, err := user.Request("u.G.U.ep.<child>.<parent>", pts.Encode(), 2*time.Second)

// a read: the reply is a node frame, and a refusal arrives as its error
reply, err = user.Request("u.G.U.nodes.all.<id>", nil, 2*time.Second)
nodes, err := data.DecodeNodes(reply.Data)
```

Helpers already in `client/*_test.go`: `credUpstream`, `enrollDevice`,
`startDeviceSync`, `makeEnrollToken`, `devicePubKey`, `waitFor`. They use the
shared test ports, so copy what they do with your own options when another test
run may be active.

- A permission violation on publish is asynchronous. Collect it with
  `nats.ErrorHandler` and wait a second; no error means the publish was allowed.
- To test a loopback rule, connect to a non-loopback address of the same machine
  (`ip -4 -o addr show scope global`). Every listener binds all interfaces.
- For HTTP, `POST /v1/auth` with form fields `email` and `password` returns the
  JWT; send it as `Authorization: Bearer <jwt>`.
- Send the test output to a file in the scratchpad and grep it. The server log
  is long, and a crashed test process loses whatever was only on screen.
- A proof that the process stops belongs in its own test, since it takes the
  other assertions down with it.

Check `pgrep -af 'go test|\.test'` and `git status` before starting. If another
session is editing the tree, leave its files alone and say so in the report.

## Tooling

```sh
go run golang.org/x/vuln/cmd/govulncheck@latest ./...          # source: only called symbols count
govulncheck -mode binary <release binary>                      # what actually shipped, and with which Go
go version <release binary>
cd frontend && npm audit --omit=dev                            # what reaches the browser
cd frontend/lib && npm audit --omit=dev
rg -n 'recover\(\)|InsecureSkipVerify|math/rand|exec\.Command|sh -c' --glob '*.go'
git ls-files | xargs rg -l 'BEGIN .*PRIVATE KEY|\bS[UA][A-Z2-7]{56}\b'
```

Release binaries are built with the Go version `go.mod` names, which can differ
from the one CI tests with; check both workflows in `.github/workflows/`.

Never read or print `device.nkey` or `local.sh`. They are local secrets and are
gitignored.

## Splitting the work

A full audit is too wide for one context. Reviewers that worked, run in
parallel, each told to give `file:line`, the principal required, a concrete
scenario, and what was checked and found sound:

- `store/`, `data/`, and the user, auth, and admin clients: scope checks, binary
  decoding, sign-in, replication trust.
- The rest of `client/` plus `modbus/`, `msg/`, `file/`, `network/`: every place
  a point or a peer's bytes reach `os/exec`, a path, a URL, an allocation, a
  ticker, or a log line.
- Frontend, dependencies, packaging, CI, repository hygiene, and the user docs
  compared with `server/args.go`.

Keep the HTTP API, the authorizer, and enrollment for yourself, and verify every
reported finding in the code, and with a test where it matters, before it goes
into a document. Reviewers are sometimes wrong about severity and about what is
reachable.

## Recording results

- **The plan** gets each finding as a numbered item with Problem, Change, and
  Verify, marked _proven_ when a test reproduced it, and a suggested order. Mark
  items complete there as they close.
- **`docs/ref/security.md`** gets corrections to any statement a test
  contradicted, and a row in Known limitations pointing at the plan item. Keep
  the deployment checklist to what an operator can do today.
- **`docs/user/`** gets corrected where a documented default or guarantee does
  not match the code.
- **`CHANGELOG.md`** gets one entry when the guidance to operators changes.
- Write in the project's documentation tone: neutral, precise, no dramatic
  wording. Describe what a principal can do, and the change that closes it.
- Do not add a vulnerability reporting contact or policy unless asked. Note its
  absence in the report instead.
- A broader design review lives outside this repository at `../security.md`.
  Mention where it has gone out of date; do not edit it without being asked.

Findings do not belong in this skill. If the harness, the principals, or the
enforcement points change, update the sections above.
