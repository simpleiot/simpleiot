+++
title = "Rules"
weight = 7
+++

The Simple IoT application has the ability to run rules. That are composed of
one or more conditions and actions. All conditions must be true for the rule to
be active.

Rules are defined by nodes and are composed of additional child nodes for
conditions and actions. See the node/point [schema](../data/rule.go) for more
details.

Node point changes cause rules of any parent node in the tree to be run. This
allows general rules to be written higher in the tree that are common for all
device nodes (for instance device offline).

All points should be sent out periodically, even if values are not changing to
indicate a node is still alive and eliminate the need to periodically run rules.
Even things like system state should be sent out to trigger device/node offline
notifications.

If a rule has not received points that meet the condition qualifications in 30m,
it is considered offline and marked as such in the UI. This helps us detect
stale rules, or rules that do not have working conditions.

## Conditions

Each condition may optionally specify a minimum active duration before the
condition is considered met. This allows timing to be encoded in the rules.

### Node state

A node state condition looks at the point value of a node to determine if a
condition is met. Qualifiers that filter points the condition is interested in
may be set including:

- node ID (if left blank, any node that is a descendent of the rule parent)
- point ID
- point type ("value" is probably the most common type)
- point index

If the provided qualification is met, then the condition may check the point
value/text fields for a number of conditions including:

- number: >, <, =, !=
- text: =, !=, contains
- boolean: on, off

## Actions

Every action has an optional repeat interval. This allows rate limiting of
actions like notifications.

### Notifications

Notifications are the simplest rule action and are sent out when:

- all conditions are met
- time since last notification is greater than the notify action repeat
  interval.

Every time a notification is sent out by a rule, a point is created/updated in
the rule with the following fields:

- id: node of point that triggered the rule
- type: "lastNotificationSent"
- time: time the notification was sent

Before sending a notification we scan the points of the rule looking for when
the last notification was sent to decide if its time to send it.

A rule notitifcation action has an optional Template field that can be used to
populate a Go template that can be used customize the notification to include
arbitrary points from source node points.

The below is an example of template:

```
Sentry Alert. {{.Description}} was ARMED with target flow rate of {{printf "%.1f" (index .Ios "flowRateTarget")}} and with tank level of {{printf "%.1f" (index .Ios "currentTankVolume")}}.
```

### Set node point

Rules can also set points in other nodes. For simplicity, the node ID must be
currently specified along with point parameters and a number/bool/text value.

Typically a rule action is only used to set one value. In the case of on/off
actions, one rule is used to turn a value on, and another rule is used to turn
the same value off. This allows for hysteresis and more complex logic than in
one rule handled both the on and off states. This also allows the rules logic to
be stateful.
