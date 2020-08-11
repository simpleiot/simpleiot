---
id: rules
title: Rules
sidebar_label: Rules
---

The Simple IoT application has the ability to run rules. That are composed of
one or more conditions and actions. All conditions must be true for the rule to
be active. Rules must currently be defined programatically by inserting a
[Rule](../data/rule.go) type into the database. These rules are then run every
10s. In the future, we hope to make rules event driven once the NATS.io
integration is finished.

## Notifications

Currently, the only rule action support are SMS notifications via
[Twilio](environment-variables.md).

A rule action has an optional Template field that can be used to populate a Go
template that can be used to customize the notification message as well as
include data from the [device state](../data/device.go). Data in the
[deviceTemplateData](../device/device.go) is available for use in templates.

## Example Rule

The below is an example of how to create a rule with a custom notification
template:

```go
var armedRule = data.Rule{
	Config: data.RuleConfig{
		Description: "IS Armed",
		Conditions: []data.Condition{
			data.Condition{
				SampleType: "armed",
				Value:      1,
				Operator:   "=",
			},
		},
		Actions: []data.Action{
			data.Action{
				Type:     data.ActionTypeNotify,
				Template: `Sentry Alert. {{.Description}} was ARMED with target flow rate of {{printf "%.1f" (index .Ios "flowRateTarget")}} and with tank level of {{printf "%.1f" (index .Ios "currentTankVolume")}}.`,
			},
		},
	},
}
```
