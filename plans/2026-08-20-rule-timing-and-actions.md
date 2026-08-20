# Plan: Rule Timing and Action Changes

**Branch:** `cbrake/master` **Branched from:** `9032d3c3`

## Context

`docs/user/rules.md` describes four planned changes to rule timing, added in
commit `9032d3c3` after the documentation was found to describe behavior that
was never implemented:

1. Enforce `minActive` on a condition (a pending period).
2. Add a minimum inactive duration, so a rule that has gone active stays active
   until its conditions have been clear for a set time.
3. Add a repeat interval to the notify action, which both re-sends a reminder
   while a rule stays active and rate limits notifications.
4. Run actions only when the rule changes state.

The design follows the model Grafana alerting and Prometheus Alertmanager have
converged on — pending period, keep firing for, repeat interval — because it
addresses the same problems: noisy conditions, flapping, and notification
fatigue.

### What the code does today

`client/rule.go` holds the whole rule engine. A `RuleClient` evaluates in
`run(id, pts)`, which is called from four places in the main loop:

- `newRulePoints` — a point arrived on a node that a condition names. The rule
  is evaluated only if some condition has `NodeID` equal to the point's node ID.
- `scheduleTicker` — a 10 second ticker, running only when the rule has a
  schedule condition, that feeds a synthetic `trigger` point.
- `newPoints` / `newEdgePoints` — a configuration change on the rule or one of
  its children, which merges the point and then calls `run("", nil)`.

`ruleProcessPoints` walks the conditions, compares each against the incoming
points, sends an `active` point for any condition whose state changed, and then
computes the rule's own `active` state as "every enabled condition is active,
and at least one is enabled". It returns `(active, changed, error)`.

`run` returns early when `changed` is false — but only on the path where points
were supplied. The `run("", nil)` path from a configuration change skips that
check entirely, so editing an active rule, a condition, or an action re-runs the
actions and re-sends the notification. That is item 4.

`Condition.MinActive` is parsed from the `minActive` point, printed in
`Condition.String`, and shown in the UI as "Min active time (m)". Nothing else
reads it. That is item 1.

There is no timer that can move a rule's state without an inbound point, other
than the schedule ticker, which runs only for schedule conditions. Items 1, 2,
and 3 all need state changes that happen because time passed, so the evaluation
loop needs a timer of its own before any of them can work.

## Design Decisions

**A condition carries both durations; the rule carries none.** `minActive` is
already a condition point, and the documentation frames the new minimum inactive
duration as its mirror image, so `minInactive` becomes a second condition point
alongside it. A rule is active when all its conditions are active, so holding a
condition active for `minInactive` after its input clears holds the rule active,
which is the behavior the documentation describes. Putting the durations on the
condition also keeps them next to the input they debounce, which matters when a
rule has several conditions with different noise characteristics.

**Raw state lives in memory; held state is the `active` point.** Each condition
gets an in-process record of its raw comparison result and the time that result
last changed. The `active` point on the condition node continues to mean what it
means today — the condition is met as far as the rule is concerned — so the UI,
history, and the rule's own state computation need no changes. A rule waiting
out a pending period simply shows its condition as not yet active.

In-memory state resets when the client restarts, so a pending period restarts
after a restart or a configuration change that reloads the client. The rule's
own `active` point is persisted, so the rule itself resumes in the state it was
in. Grafana behaves the same way across a restart, and persisting a pending
timer would mean writing a point on every evaluation.

**Evaluation is driven by a deadline timer, not a faster poll.** Rather than
shortening the schedule ticker or running it for every rule, the client computes
the earliest time at which something could change on its own — a pending period
expiring, a hold expiring, a repeat notification coming due — and arms a
`time.Timer` for exactly that moment. When nothing is pending, no timer runs.
This keeps an idle instance idle, and it makes the timing exact rather than
rounded up to the next tick, which matters for testing at sub-second durations.

**Durations stay in minutes and stay `float64`.** `minActive` is already minutes
and already `float64`, and the UI labels it "(m)". Fractional values work, which
lets the tests exercise the timing at hundredths of a minute instead of adding a
clock abstraction.

**A repeat interval of zero keeps today's behavior.** With no repeat interval
set, a notify action sends one notification per transition and is not rate
limited, exactly as it does now. Setting an interval turns on both behaviors at
once: reminders while the rule stays active, and a floor on the time between
notifications from that action.

**Reminders are for `action` nodes, not `actionInactive` nodes.** An inactive
action fires when a rule resolves, and a resolved rule is the normal state, so
repeating that notification would mean a reminder forever. The rate limit
applies in both directions; the reminder applies only while the rule is active.

## Phases

Each phase is a commit, and each updates `CHANGELOG.md` and the affected part of
`docs/user/rules.md`, removing the corresponding bullet from that page's
"Planned Changes" section as it lands. When the last phase is done, the section
is gone.

### Phase 1 — Run actions only on state transitions

The smallest and most user-visible fix, and it stands on its own.

- Restructure `run` so that every path computes the rule's state and then acts
  only if the state changed from the client's previous view of it. The
  configuration-change path (`run("", nil)`) currently ignores `changed`; fold
  it into the same check.
- Keep the "disabled forces inactive" behavior, and make it a transition:
  disabling an active rule runs the inactive actions once, and further edits
  while disabled run nothing.
- Add an explicit first-evaluation case. On startup the client has a persisted
  `active` point but has never run its actions in this process; decide, and
  document in the code, that a client whose computed state matches its persisted
  state does not re-assert `setValue` outputs. Anything else re-sends a
  notification on every restart.

Tests: editing a rule's description, a condition's value, and an action's
description while the rule is active produces no additional notification point
and no additional `setValue` write; disabling an active rule runs the inactive
actions exactly once.

### Phase 2 — Deadline-driven evaluation

A refactor with no behavior change of its own, laying the foundation for the
remaining phases.

- Split `ruleProcessPoints` into three pieces: raw condition evaluation against
  incoming points, the held-state pass that decides what each condition's
  `active` point should be, and the rule-level state computation that already
  exists at the end of the function. Only the first piece needs points; the
  other two run from a timer with nothing new to compare.
- Add `nextDeadline()`, returning the earliest pending deadline across all
  conditions and actions, and arm a `time.Timer` from it in the main loop.
  Re-arm after every evaluation. With no deadlines, the timer stays stopped.
- Leave the schedule ticker as it is. It answers a different question — has the
  wall clock crossed a schedule boundary — and the schedule conditions still
  need to be re-evaluated against a fresh trigger point.

Tests: existing rule tests continue to pass unchanged. Add a unit test for
`nextDeadline()` over a set of conditions and actions.

### Phase 3 — Enforce `minActive`

- Add per-condition raw state: the last raw comparison result and when it last
  changed.
- In the held-state pass, a condition whose raw result is true but whose held
  state is false becomes active only once the raw result has held for
  `minActive` minutes. A raw result that drops back to false before then clears
  the pending period, and the condition never activates.
- Report the pending deadline through `nextDeadline()` so the condition
  activates at the moment the period expires, with no inbound point.
- A `minActive` of zero keeps the current immediate behavior.
- Handle the configuration edge cases: `minActive` changed while pending (the
  new value applies against the existing raw-change time), the condition
  disabled while pending, and the rule disabled while pending (clears it).

Tests: a value that crosses the threshold and returns before `minActive` expires
never activates the rule; a value that stays across activates it after the
period with no further points arriving; a change to `minActive` while pending
takes effect.

### Phase 4 — Minimum inactive duration

- Add `PointTypeMinInactive = "minInactive"` to `data/schema.go`, the matching
  `typeMinInactive` in `frontend/src/Api/Point.elm`, and a `MinInactive float64`
  field on `Condition`.
- Add the mirror rule to the held-state pass: a condition whose raw result is
  false but whose held state is true stays active until the raw result has been
  false for `minInactive` minutes. A raw result that returns to true within the
  window cancels the hold, and the condition never deactivates — one incident,
  one notification.
- Report the hold deadline through `nextDeadline()`.
- Add the input to the `pointValue` branch of `NodeCondition.elm`, next to "Min
  active time (m)", labelled "Min inactive time (m)".
- Extend the schema section of `docs/user/rules.md` and the example YAML.

Tests: a value oscillating across a threshold faster than `minInactive` produces
one activation and one notification rather than one per cycle; the condition
deactivates on its own once the input has been clear for the full duration.

### Phase 5 — Notify repeat interval

- Add `PointTypeRepeatInterval = "repeatInterval"` to `data/schema.go`, the
  matching `typeRepeatInterval` in `frontend/src/Api/Point.elm`, and a
  `RepeatInterval float64` field on `Action` and `ActionInactive`.
- Track the last notification time per action in memory.
- Rate limit: a notify action does not send if it sent less than
  `repeatInterval` minutes ago. The transition still happens and the rule state
  is still correct; only the notification is dropped.
- Reminder: while the rule is active and an `action` node has a repeat interval,
  re-send the notification every interval, through `nextDeadline()`.
- Show the field in `NodeAction.elm` only when the action type is `notify`,
  following the existing pattern that hides `setValue` and `playAudio` fields.
- Document the two behaviors and the zero default in `docs/user/rules.md`.

Tests: a rule that transitions repeatedly notifies at most once per interval; a
rule that stays active re-notifies at the interval and stops when it resolves;
an inactive action does not repeat.

### Phase 6 — Documentation and changelog sweep

- Remove the "Planned Changes" section from `docs/user/rules.md`, folding
  anything still worth saying into the "Conditions" and "Notifications"
  sections.
- Verify the schema section matches the point types actually implemented, and
  that the example YAML round-trips through `siot import`.
- Confirm the `CHANGELOG.md` entries added along the way read as one coherent
  set for someone upgrading, and call out that `minActive` starts taking effect
  on rules that already have it set.

## Upgrade Impact

`minActive` is stored on existing rules and has never done anything. After phase
3, any rule with a non-zero `minActive` starts waiting that long before
activating. That is the documented intent, and it is a behavior change for
existing configurations, so it needs an explicit changelog note.

The other three changes only remove notifications that a user would not have
asked for: repeated notifications from configuration edits, from flapping, and
from a rule transitioning faster than its repeat interval.

## Testing

`client/rule_test.go` has a `ruleTestServer` harness that stands up a test
server with a rule, conditions, actions, and input and output variable nodes,
and a `checkVout` helper that polls for an expected output value with a one
second timeout. The timing tests need:

- Durations expressed as fractions of a minute (0.01 minutes is 600 ms) so the
  suite stays fast.
- A helper that asserts an output does _not_ change within a window, which is
  the shape most of these tests take.
- A way to count notification points on the rule node, for the rate limit and
  reminder tests. The notification point uses a fixed key, so counting means
  subscribing rather than reading the current value.

Run `go test -race ./client/...` for each phase and `siot_test` before the final
commit.

## Open Questions

1. **Should a wildcard condition ever be evaluated on an inbound point?** The
   `newRulePoints` handler only runs the rule if some condition has `NodeID`
   equal to the point's node ID, so a condition with a blank `NodeID` — which
   `docs/user/rules.md` documents as watching every node below the rule's parent
   — is never evaluated from an incoming point. Either the code or the
   documentation is wrong. It is adjacent to this work rather than part of it,
   and fixing it means thinking about the feedback loop the current check is
   guarding against, so it is called out here rather than scheduled.

2. **Should the rate limit defer a notification or drop it?** The plan drops it.
   Deferring would mean a rule that resolved before the interval expired still
   sends a notification about being active, which reads as stale. Grafana drops.

3. **Should `minActive` and `minInactive` apply to schedule conditions?** The
   held-state pass is uniform, so they would work, but the UI shows them only
   for point value conditions and a debounced schedule boundary is hard to
   reason about. The plan keeps the behavior uniform in the engine and the
   fields hidden in the UI for schedules.

4. **Is a `lastNotificationSent` point worth persisting after all?** The removed
   documentation claimed one existed. In-memory state means the rate limit
   resets on restart. The notification point's own timestamp on the rule node is
   effectively this value already, so reading it back at startup is a possible
   refinement if restart-resistant rate limiting turns out to matter.
