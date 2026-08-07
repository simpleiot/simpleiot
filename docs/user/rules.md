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

Each condition may optionally specify a minimum active duration before the
condition is considered met. This allows timing to be encoded in the rules.

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

Every action has an optional repeat interval. This allows rate limiting of
actions like notifications.

### Notifications

Notifications are the simplest rule action and are sent out when:

- All conditions are met
- Time since last notification is greater than the notify action repeat
  interval.

Every time a notification is sent out by a rule, a point is created/updated in
the rule with the following fields:

- `id`: node of point that triggered the rule
- `type`: "`lastNotificationSent`"
- `time`: time the notification was sent

Before sending a notification we scan the points of the rule looking for when
the last notification was sent to decide if its time to send it.

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

The configuration of a rule with a point value condition, a schedule
condition, and an action for each direction:

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

`conditionType` is `pointValue` or `schedule`. A point value condition
qualifies the points it is interested in with `pointType` and `pointKey`, and
`valueType` decides how it compares them: a `number` condition compares `value`
using `operator`, one of `>`, `<`, `=`, or `!=`; a `text` condition compares
`valueText` using `=`, `!=`, or `contains`; and an `onOff` condition matches a
`value` of `1` or `0` and needs no operator. `minActive` is how many minutes
the condition has to hold before the rule goes active.

A schedule condition uses `start` and `end`, written as text so `08:00` keeps
its leading zero, along with `weekday` and `date`. `weekday` is seven points,
Sunday first, each `1` or `0`. `date` is a list of dates, and a schedule
carries dates or weekdays rather than both.

`action` is `notify`, `setValue`, or `playAudio`. A `setValue` action names
what to write with `nodeID`, `pointType`, and `pointKey`, and what to write
with `valueType` and `value` or `valueText`. A `playAudio` action names the WAV
file to play with `filePath`, the ALSA device to play it on with `device`, and
the channel with `channel`.

The rule's `active` state, the time the last notification was sent, and any
error are points the client maintains, so an export of a running rule carries
them as well.
