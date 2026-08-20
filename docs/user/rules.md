# Rules

**Contents**

<!-- toc -->

The Simple IoT application has the ability to run rules - see the video below
for a demo:

<iframe width="640" height="360" src="https://www.youtube.com/embed/pb_a6oEdFJI" title="Simple IoT Rules Demo" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

Rules are composed of one or more conditions and actions. All conditions must be
true for the rule to be active.

Node point changes cause rules of any parent node in the tree to be run. This
allows general rules to be written higher in the tree that are common for all
device nodes (for instance device offline).

In the below configuration, a change in the SBC propagates up the node tree,
thus both the `D5 on rule` or the `Device offline rule` are eligible to be run.

![rules](images/rules.png)

## Node linking

Both conditions and actions can be linked to a node ID. If you copy a node, its
ID is stored in a virtual clipboard and displayed at the top of the screen. You
can then paste this node ID into the Node ID field in a condition or action.

![rule-linking](images/rule-copy-paste-node-id.png)

## Conditions

Each condition may optionally specify a minimum active duration (`minActive`)
before the condition is considered met. This allows timing to be encoded in the
rules. **Note:** `minActive` is accepted and stored today but not yet enforced —
see [Planned Changes](#planned-changes) below.

### Node state

A point value condition looks at the point value of a node to determine if a
condition is met. Qualifiers that filter points the condition is interested in
can set including:

- Node ID (if left blank, any node that is a descendant of the rule parent)
- Point type ("value" is probably the most common type)
- Point Key (used to index into point arrays and objects)

If the provided qualification is met, then the condition may check the point
value/text fields for a number of conditions including:

- number: `>`, `<`, `=`, `!=`
- text: `=`, `!=`, `contains`
- boolean: `on`, `off`

### Schedule

Rule conditions can be driven by a schedule that is composed of:

- start/stop time
- weekdays
- dates

If no weekdays are selected, then all weekdays are included.

When the dates are used, then weekdays are disabled.

Conversely, when a weekday is enabled, dates are disabled.

As a time range can span two days, the start time is used to qualify weekdays
and dates.

<img src="./images/rule-schedule.png" alt="image-20230721173842815" style="zoom:67%;" />

See also a video demo:

<iframe width="791" height="445" src="https://www.youtube.com/embed/WllM0acCOss" title="Creating an Alarm Clock with Simple IoT schedules" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## Actions

Actions run when the rule changes state. Actions of type `action` run on the
inactive to active transition, and actions of type `actionInactive` run on the
active to inactive transition. Editing a rule, a condition, or an action while
the rule is active does not re-run the actions — only a change of state does.

Disabling a rule makes it inactive, which is a transition like any other, so
disabling an active rule runs its inactive actions once and further edits while
it stays disabled run nothing.

Rule state is persisted, so a restart resumes in the state the rule was in and
does not re-run the actions or re-send the notification for a state that has not
changed.

### Notifications

A notify action publishes a [notification](notifications.md) point on the rule
node each time the rule goes active (or inactive, for an inactive action). The
notification carries the rule description as the subject and names the node that
triggered the rule in the message. From there it is delivered to users and
messaging services in scope as described in the
[notifications documentation](notifications.md).

Each state transition sends one notification. There is no rate limiting at the
rule yet, so a rule that cycles rapidly between active and inactive (a sensor
sitting right at a threshold, a device going on and off line) sends a
notification per transition — see [Planned Changes](#planned-changes).

### Set node point

Rules can also set points in other nodes. For simplicity, the node ID must be
currently specified along with point parameters and a number/bool/text value.

Typically a rule action is only used to set one value. In the case of on/off
actions, one rule is used to turn a value on, and another rule is used to turn
the same value off. This allows for hysteresis and more complex logic than in
one rule handled both the on and off states. This also allows the rules logic to
be stateful. If you don't need hysteresis or complex state, the rule "inactive
action" can be used, which allows the rule to take action when it goes both
active and inactive.

## Disable Rule/Condition/Action

![rule-disable](images/rule-disable.png)

### Disable Rule

A rule can be disabled. If the rule is disabled while active, then the rule
inactive actions are run so that things get cleaned up if necessary and the
actions are not left active.

### Disable Condition

If there are no conditions, or all conditions are disabled, the rule is
inactive. Otherwise, disabled conditions are simply ignored. For example, if
there is a disabled condition and a non-disabled active condition, the rule is
active.

### Disable Action

A disabled action is not run.

## Schema

The configuration of a rule with a point value condition, a schedule condition,
and an action for each direction:

```yaml
nodes:
  - rule:
      description: Tank low
      disabled: 0
      children:
        - condition:
            conditionType: pointValue
            description: Level below 10
            disabled: 0
            minActive: 5
            nodeID: Tank level
            operator: <
            pointKey: ""
            pointType: value
            value: 10
            valueType: number
        - condition:
            conditionType: schedule
            description: Working hours
            end: "17:00"
            start: "08:00"
            weekday:
              - 0
              - 1
              - 1
              - 1
              - 1
              - 1
              - 0
        - action:
            action: notify
            description: Tell the operators
        - actionInactive:
            action: setValue
            description: Clear the alarm
            nodeID: Alarm relay
            pointType: switchSet
            value: 0
            valueType: onOff
```

Conditions and actions are children of the rule, and an inactive action is a
child of type `actionInactive`, which is what lets one rule act in both
directions.

`nodeID` names the node a condition watches or an action writes to, and it is
written as that node's description rather than as an ID, so a rule can be moved
between instances. Leaving it out of a condition watches every node below the
rule's parent. See
[referring to another node](configuration.md#referring-to-another-node) for how
the name is resolved.

`conditionType` is `pointValue` or `schedule`. A point value condition qualifies
the points it is interested in with `pointType` and `pointKey`, and `valueType`
decides how it compares them: a `number` condition compares `value` using
`operator`, one of `>`, `<`, `=`, or `!=`; a `text` condition compares
`valueText` using `=`, `!=`, or `contains`; and an `onOff` condition matches a
`value` of `1` or `0` and needs no operator. `minActive` is how many minutes the
condition has to hold before the rule goes active (stored today, enforced as
part of the [planned changes](#planned-changes)).

A schedule condition uses `start` and `end`, written as text so `08:00` keeps
its leading zero, along with `weekday` and `date`. `weekday` is seven points,
Sunday first, each `1` or `0`. `date` is a list of dates, and a schedule carries
dates or weekdays rather than both.

`action` is `notify`, `setValue`, or `playAudio`. A `setValue` action names what
to write with `nodeID`, `pointType`, and `pointKey`, and what to write with
`valueType` and `value` or `valueText`. A `playAudio` action names the WAV file
to play with `filePath`, the ALSA device to play it on with `device`, and the
channel with `channel`.

The rule's `active` state, its most recent notification, and any error are
points the client maintains, so an export of a running rule carries them as
well.

## Planned Changes

The timing behavior around rule state and notifications is being improved. The
design follows the model that Grafana alerting and Prometheus Alertmanager have
converged on, since it has proven itself for exactly the problems rules have
here: noisy conditions, flapping, and notification fatigue.

- **Enforce `minActive` (pending period).** A condition with `minActive` set
  will have to hold continuously for that long before it is considered met. This
  is the equivalent of Grafana's pending period, which keeps a brief spike — a
  level that grazes a threshold, a device that drops offline for a few seconds —
  from activating the rule at all. Today the field is stored and shown in the UI
  but has no effect.

- **A minimum inactive duration to stop flapping.** The mirror image of
  `minActive`: once active, a rule will stay active until its conditions have
  been clear for a set duration. Grafana calls this "keep firing for" and gives
  the alert a recovering state. Without it, a value oscillating around a
  threshold resolves and re-fires repeatedly, and every cycle is a new
  notification. With it, one incident is one notification.

- **A repeat interval on the notify action.** Two related behaviors, both taken
  from Grafana's repeat interval (default there is 4 hours):
  - While a rule stays active, an optional reminder can be re-sent every repeat
    interval, so a long-running condition is not a single message that scrolled
    away hours ago.
  - The interval also acts as a rate limit: a rule will not notify more often
    than the repeat interval no matter how often it transitions, bounding the
    damage from a flapping condition that slips past the duration guards.

Grafana's remaining defenses — evaluating over an aggregation window instead of
raw samples, and a recovery threshold separate from the firing threshold —
already have equivalents here: smoothing belongs in the client producing the
point, and hysteresis is done with two rules or an inactive action as described
in [Set node point](#set-node-point) above.
