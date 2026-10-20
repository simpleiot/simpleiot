# Plan: Enrollment Tokens Place Devices in Their Own Group

**Branch:** `cbrake/master` **Branched from:** `1f56a211`

## Context

An enrollment token (`enrollToken` node) lets a device with no credential ask
the upstream for one. Today the enroller (`server/enroll.go`) creates every new
device node under the root, wherever the token sits, and `siot cred token`
always creates the token under the root. A fleet split across customers or sites
ends up with every new unit in one flat list, and an operator moves each one by
hand.

The goal: an operator puts an enrollment token in any group, gives that token to
the devices meant for that group, and a new device appears in the group whose
token it presented. Several tokens can live side by side, one per group, and the
root keeps working as it does now.

The token decides only where a device first appears. After that the device is an
ordinary node: a user moves it to any group they are authorized for, using the
existing move operation, and enrollment never moves it again.

What the code already gives us:

- `refreshEnroll` in `server/auth.go` reads the token's edges, so it can record
  the token's parent with no extra lookup. `handleUp` already refreshes a token
  on an edge change, so moving a token is picked up.
- `enrollToken` is already tree scoped in `data/edge_role.go`.
- A device under a group already connects: `deviceLive` checks only that the
  credential sits under a live `device` node, and the grant does not include the
  parent, so moving a device does not close its connection.
- The root is a `device` node, so the add-node list under the root already
  offers an enrollment token; it is the `group` list that lacks one.

## Approach

1. **The token's parent is the destination.** The authorizer records the parents
   of each token's live edges, which it already has from `refreshEnroll`, and
   does no further lookup. The enroller reads the parent at enroll time and
   creates a new device node under it instead of under the root. A token is
   honored only under the root or a live `group` node; one anywhere else (under
   a device, say) enrolls nothing. The check has to be at enroll time:
   `DeleteNode` tombstones only the group's own edges, so a token under a
   deleted group keeps a live edge, and the authorizer never hears about the
   group's tombstone since `handleUp` keys on the token's node ID. Without the
   check a deleted group would keep collecting devices nobody can see.
2. **One destination per token.** A token mirrored into two places is ambiguous,
   so the enroller refuses it. Making a second token for the second group is the
   supported setup. Both refusals happen in the enroller, not at connect time,
   with the reason logged, so an operator can see why a token enrolls nothing.
3. **An existing device stays where it is.** A device that was enrolled and then
   moved still carries its original token in its image, so a re-enrollment (a
   new key after a re-flash) must not move it back or be refused for being
   elsewhere. The enroller adds the credential to the device wherever it now
   sits and leaves its edges alone.
4. **Auto-approve applies only to a device the enrollment creates.** For an
   existing device the credential is always pending, whatever the token says.
   Today that holds only when the device already has a live credential, so an
   auto-approve token holder who names an existing device with none gets an
   approved credential for it. That covers a device node created before the unit
   first connects, and a device whose credential was disabled to revoke it,
   which a new key would otherwise undo. With a token per group, the holder of
   group A's token could do this to a device in group B. A device whose
   credential was deleted so it can re-enroll waits for one approval.

   Two limits of this remain and are documented rather than closed here. The
   holder of group A's token can still name a device ID in group B and queue a
   pending credential there, which an operator approving by habit could grant;
   `enrolledBy` (item 6) is the mitigation, and the UI shows the token's group
   next to a pending credential so the mismatch is visible at approval time. The
   reply also tells an auto-approve token holder whether an ID exists:
   `approved` for a new device, `pending` for an existing one.

5. **The pending limit stays global.** `maxPendingEnrollments` (100) still
   counts all pending devices. It is checked only when a device is created, so
   credentials queued on existing devices are not bounded by it; that is
   unchanged, and with item 4 it is the only path for an existing device, so it
   is noted in the security follow-ups. A per-token limit is a possible
   follow-up; it is not needed for placement.
6. **Credentials record their token.** The enroller writes the token's node ID
   on the new `deviceCred` as `enrolledBy`, so an operator can see which token
   brought a device in and revoke accordingly. (`enrollToken` is already the
   secret point type on the sync node, hence the different name.)

### Relation to security follow-up item 10

[Item 10](2026-09-19-security-followups.md#10-limit-which-node-types-an-account-may-create)
proves that any account can create an auto-approve `enrollToken` in its own
group. This plan makes such a token more useful, since devices it enrolls now
land in that group. Item 10's proposed fix (only the shared token or a
root-anchored principal may create `enrollToken` nodes) closes that, and still
lets an admin place tokens in any group. Its note that "the authorizer can also
ignore enrollment tokens that are not under the root" becomes "not under the
root or a group". This plan does not wait on item 10: today such a token enrolls
devices to the root, and after Phase 2 they land in the user's own group, which
is less exposure, not more. Update item 10's text in Phase 1 and add the
unbounded pending credentials on existing devices (item 5 above) to the
follow-ups.

## Phases

### Phase 1: Authorizer tracks the token's parent

- [x] Add `parents []string` to `enrollEntry`. In `refreshEnroll`, collect the
      parents of the live edges; no type lookup here. The entry is refreshed
      when the token's edges change (already wired through `handleUp`).
- [x] Replace `EnrollToken(token) (autoApprove, ok bool)` with a method that
      returns the token's node ID, parents, and `autoApprove` together, so the
      enroller reads one consistent entry.
- [x] Update item 10 in `plans/2026-09-19-security-followups.md` as described
      above, and add a follow-up for the unbounded pending credentials on
      existing devices.
- [x] Tests in `server/auth_test.go`: a token under a group reports that group;
      a mirrored token reports both parents; moving a token from the root to a
      group changes its parent.
- [x] `go test -race ./server/...`, `golangci-lint run`. Commit.

### Phase 2: Enroller creates the device in the token's group

- [x] In `enroller.enroll`, refuse a token with more than one parent, and read
      the parent with one `GetNodes` call; refuse it unless it is the root or a
      live `group`. Log the reason on each refusal. Create a new device under
      that parent instead of `e.auth.root()`. "Live" means a path of live edges
      leads to the root, so a group under a deleted group is refused too.
- [x] For an existing device, leave its edges alone and create the credential
      pending regardless of `autoApprove` (replaces the `hasLive` special case).
- [x] Write `enrolledBy` (token node ID) on the new `deviceCred`. Add
      `PointTypeEnrolledBy` to `data/schema.go` and the field to
      `client.DeviceCred`.
- [x] Tests, alongside `TestSyncEnroll` in `client/sync_enroll_test.go`: two
      tokens in two groups, each enrolls a device into its own group; a root
      token still enrolls under the root; a token under a device is refused; a
      mirrored token is refused; a token under a deleted group is refused; a
      token that is moved between enrollments sends the next new device to the
      new place; a device enrolled by group A's token and moved to group B keeps
      its credential and connection, and a re-enrollment with a new key leaves
      it in group B with a pending credential; an auto-approve token naming an
      existing device with no live credential gets a pending credential.
- [x] `go test -race ./server/... ./client/...`, `golangci-lint run`. Commit.

### Phase 3: CLI and UI

- [x] `siot cred token -parent ID|DESCRIPTION` creates the token under that node
      (default: the root), resolving a description the way `findDevice` does for
      devices. Refuse a parent that is not the root or a group.
- [x] `siot cred list` shows `enrolledBy` for each credential.
- [x] Frontend: offer **Enrollment token** in the add-node list under a `group`
      (`viewAddNode` in `frontend/src/Pages/Home_.elm`; it is offered only under
      devices today). Show `enrolledBy` on the device credential view
      (`Components/NodeDeviceCred.elm`) as the token's description and the group
      it sits in, so a pending credential from another group's token stands out.
      A user who cannot see the token's group sees the raw ID.
- [x] `siot_test_frontend`, `cd frontend && npx elm-review`. Commit.

### Phase 4: Documentation and changelog

- [x] `docs/user/sync.md`, "Devices that enroll themselves": tokens may sit in
      the root or any group, and a new device lands in the group of the token it
      used; after that it can be moved like any node. Add a short example with
      one token per customer group.
- [x] `docs/ref/security.md`: an enrollment token is honored only under the root
      or a live group; mirrored tokens are refused; auto-approve covers only a
      device the enrollment creates, and a key for an existing device always
      waits for an operator. Note what a token holder can still do to a device
      in another group (queue a pending credential, learn that its ID exists)
      and that `enrolledBy` is how an operator tells.
- [x] Update the `NodeTypeEnrollToken` comment in `data/schema.go` ("kept under
      the upstream root").
- [x] `CHANGELOG.md` entry under `[Unreleased]`.
- [x] Commit.

## Incompatibilities to note in the changelog

- A token that is not under the root or a live group, or that is mirrored into
  more than one place, stops enrolling devices. Tokens created by
  `siot cred token` or the UI so far sit under the root and keep working.
- An auto-approve token no longer approves a key for a device node that already
  exists, such as one created by hand before the device first connects; that
  credential waits for an operator. A device that already has a live credential
  was pending in this case before, so it is unaffected.
